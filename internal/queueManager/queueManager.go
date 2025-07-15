package queueManager

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mitchs-dev/dislo/internal/configuration"
	"github.com/mitchs-dev/dislo/internal/handler"
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	log "github.com/sirupsen/logrus"
)

// Defaults
var (
	defaultNextInQueueTimeout = "15s"
)

var (
	clientQueueMapNameKey = "$INSTANCE:$NAMESPACE:$ID"    // The map of the client queues (in the code)
	clientQueueKey        = clientQueueMapNameKey + ":q"  // The map of the client queues (in redis)
	clientQueueNextKey    = clientQueueMapNameKey + ":qn" // The queue id of the next in queue (in redis)
	clientQueueTimeKey    = clientQueueMapNameKey + ":qt" // The time to base the timer on for the next in queue (in redis)
)

// A mutex map for local coordination
var lockMutexes sync.Map

// Formats the key based on instance, namespace and ID
func formatKey(key, instanceStr, namespace, id string) string {
	result := key
	result = strings.Replace(result, "$INSTANCE", instanceStr, -1)
	result = strings.Replace(result, "$NAMESPACE", namespace, -1)
	result = strings.Replace(result, "$ID", id, -1)
	return result
}

// Gets the timeout duration for the next client in queue
func nextInQueueTimeout(namespace string, instance int) (time.Duration, pb.Errors) {
	instanceConfig, ok := configuration.Instances[instance]
	if !ok {
		return 0, pb.Errors_INSTANCE_OUTSIDE_SERVER_RANGE
	}

	nsConfig, ok := instanceConfig.Namespaces[namespace]
	if !ok {
		return 0, pb.Errors_NAMESPACE_NOT_FOUND
	}

	timeoutStr := nsConfig.NextInQueueTimeout
	if timeoutStr == "" {
		timeoutStr = defaultNextInQueueTimeout
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Errorf("Error parsing timeout duration: %v", err)
		return 0, pb.Errors_INTERNAL_SERVER_ERROR
	}

	return timeout, pb.Errors_UNKNOWN_ERROR
}

// Adds a client to the queue for a specific lock ID
func addToQueue(ctx context.Context, lockId, namespace string, instance int, clientID uuid.UUID) (string, pb.Errors) {
	instanceStr := strconv.Itoa(instance)
	queueKey := formatKey(clientQueueKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	redisConn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return "", pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Generate a unique queue ID
	queueId := uuid.New().String()
	queueItem := queueId + ":" + clientID.String()

	// Check if queue exists
	exists, err := redisConn.Exists(queueKey)
	if err != nil {
		log.Errorf("Failed to check if queue exists: %v", err)
		return "", pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Add client to the queue using RPUSH
	// Since we don't have direct RPUSH, we need to:
	// 1. Get current queue (if exists)
	// 2. Append new item
	// 3. Save updated queue
	var queueItems []string
	if exists {
		currentQueue, err := redisConn.Get(queueKey)
		if err != nil {
			log.Errorf("Failed to get queue: %v", err)
			return "", pb.Errors_INTERNAL_SERVER_ERROR
		}
		if currentQueue != "" {
			queueItems = strings.Split(currentQueue, ",")
		}
	}

	queueItems = append(queueItems, queueItem)
	newQueueValue := strings.Join(queueItems, ",")

	err = redisConn.Set(queueKey, newQueueValue)
	if err != nil {
		log.Errorf("Failed to update queue: %v", err)
		return "", pb.Errors_INTERNAL_SERVER_ERROR
	}

	// If this is the first client in queue, set them as next
	if len(queueItems) == 1 {
		nextKey := formatKey(clientQueueNextKey, instanceStr, namespace, lockId)
		err = redisConn.Set(nextKey, queueId)
		if err != nil {
			log.Errorf("Failed to set next in queue: %v", err)
			return "", pb.Errors_INTERNAL_SERVER_ERROR
		}
	}

	return queueId, pb.Errors_UNKNOWN_ERROR
}

// Get the client's position in the queue
func getPositionInQueue(ctx context.Context, lockId, namespace string, instance int, queueId string) (int64, pb.Errors) {
	instanceStr := strconv.Itoa(instance)
	queueKey := formatKey(clientQueueKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	redisConn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Check if queue exists
	exists, err := redisConn.Exists(queueKey)
	if err != nil {
		log.Errorf("Failed to check if queue exists: %v", err)
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Get queue items
	queueValue, err := redisConn.Get(queueKey)
	if err != nil {
		log.Errorf("Failed to get queue: %v", err)
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Process queue items
	if queueValue == "" {
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	items := strings.Split(queueValue, ",")

	// Find the position of the queue ID
	for i, item := range items {
		// Item format is "queueId:clientId"
		parts := strings.Split(item, ":")
		if len(parts) > 0 && parts[0] == queueId {
			return int64(i), pb.Errors_UNKNOWN_ERROR
		}
	}

	return -1, pb.Errors_INTERNAL_SERVER_ERROR
}

// When a lock has been unlocked, the next client in the queue will have a time period
// to which they can acquire the lock. In this time period, the lock id will be locked to
// ensure that no other client can acquire the lock. If the client has not acquired the lock
// by the end of the time period, the lock id will be unlocked and the next client in the queue
func enterNextClientTimeoutPeriod(ctx context.Context, lockId, namespace string, instance int, timeoutPeriod time.Duration) pb.Errors {
	instanceStr := strconv.Itoa(instance)
	lockKey := formatKey(clientQueueMapNameKey, instanceStr, namespace, lockId)
	timeKey := formatKey(clientQueueTimeKey, instanceStr, namespace, lockId)
	nextKey := formatKey(clientQueueNextKey, instanceStr, namespace, lockId)
	queueKey := formatKey(clientQueueKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	managementInstance := configuration.Context.Cluster.ClusterManagement.DB
	redisConn, err := handler.RedisConnection(managementInstance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Get mutex for this lock
	var mtx sync.Mutex
	actual, _ := lockMutexes.LoadOrStore(lockKey, &mtx)
	mutex := actual.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	// Check if there's anybody in the queue
	exists, err := redisConn.Exists(queueKey)
	if err != nil {
		log.Errorf("Failed to check if queue exists: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		// No one in queue, nothing to do
		return pb.Errors_UNKNOWN_ERROR
	}

	queueValue, err := redisConn.Get(queueKey)
	if err != nil {
		log.Errorf("Failed to get queue: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	items := strings.Split(queueValue, ",")
	if len(items) == 0 {
		// Empty queue, nothing to do
		return pb.Errors_UNKNOWN_ERROR
	}

	// Get the first item in queue
	item := items[0]

	// Extract queue ID
	parts := strings.Split(item, ":")
	if len(parts) < 1 {
		log.Errorf("Invalid queue item format: %s", item)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}
	queueId := parts[0]

	// Set the next client in queue
	err = redisConn.Set(nextKey, queueId)
	if err != nil {
		log.Errorf("Failed to set next in queue: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Set the timeout timestamp
	expiration := time.Now().Add(timeoutPeriod)
	err = redisConn.Set(timeKey, strconv.FormatInt(expiration.UnixNano(), 10))
	if err != nil {
		log.Errorf("Failed to set timeout: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	return pb.Errors_UNKNOWN_ERROR
}

// Check if client is the next in queue
func isNextInQueue(ctx context.Context, lockId, namespace string, instance int, queueId string) (bool, pb.Errors) {
	instanceStr := strconv.Itoa(instance)
	nextKey := formatKey(clientQueueNextKey, instanceStr, namespace, lockId)
	timeKey := formatKey(clientQueueTimeKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	redisConn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Check if this client is the next in queue
	exists, err := redisConn.Exists(nextKey)
	if err != nil {
		log.Errorf("Failed to check if next key exists: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		return false, pb.Errors_UNKNOWN_ERROR
	}

	nextInQueue, err := redisConn.Get(nextKey)
	if err != nil {
		log.Errorf("Failed to get next in queue: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	if nextInQueue != queueId {
		return false, pb.Errors_UNKNOWN_ERROR
	}

	// Check if the timeout has passed
	exists, err = redisConn.Exists(timeKey)
	if err != nil {
		log.Errorf("Failed to check if timeout key exists: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		// No timeout set, client is still next
		return true, pb.Errors_UNKNOWN_ERROR
	}

	timeoutStr, err := redisConn.Get(timeKey)
	if err != nil {
		log.Errorf("Failed to get timeout: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	timeout, err := strconv.ParseInt(timeoutStr, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse timeout: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	expiration := time.Unix(0, timeout)
	if time.Now().After(expiration) {
		// Timeout passed, move to next client
		if err := moveToNextClient(ctx, lockId, namespace, instance); err != pb.Errors_UNKNOWN_ERROR {
			return false, err
		}
		return false, pb.Errors_UNKNOWN_ERROR
	}

	return true, pb.Errors_UNKNOWN_ERROR
}

// Move to the next client in queue
func moveToNextClient(ctx context.Context, lockId, namespace string, instance int) pb.Errors {
	instanceStr := strconv.Itoa(instance)
	queueKey := formatKey(clientQueueKey, instanceStr, namespace, lockId)
	nextKey := formatKey(clientQueueNextKey, instanceStr, namespace, lockId)
	timeKey := formatKey(clientQueueTimeKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	redisConn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Get the current queue
	exists, err := redisConn.Exists(queueKey)
	if err != nil {
		log.Errorf("Failed to check if queue exists: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		// Clean up any lingering keys
		redisConn.Delete(nextKey)
		redisConn.Delete(timeKey)
		return pb.Errors_UNKNOWN_ERROR
	}

	queueValue, err := redisConn.Get(queueKey)
	if err != nil {
		log.Errorf("Failed to get queue: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	items := strings.Split(queueValue, ",")
	if len(items) == 0 {
		// Clean up any lingering keys
		redisConn.Delete(queueKey)
		redisConn.Delete(nextKey)
		redisConn.Delete(timeKey)
		return pb.Errors_UNKNOWN_ERROR
	}

	// Remove first item from queue
	items = items[1:]

	// Delete timeout
	redisConn.Delete(timeKey)

	if len(items) == 0 {
		// No more clients, clean up
		redisConn.Delete(queueKey)
		redisConn.Delete(nextKey)
		return pb.Errors_UNKNOWN_ERROR
	}

	// Update the queue
	newQueueValue := strings.Join(items, ",")
	err = redisConn.Set(queueKey, newQueueValue)
	if err != nil {
		log.Errorf("Failed to update queue: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Get the next client timeout duration
	timeout, errTimeout := nextInQueueTimeout(namespace, instance)
	if errTimeout != pb.Errors_UNKNOWN_ERROR {
		return errTimeout
	}

	return enterNextClientTimeoutPeriod(ctx, lockId, namespace, instance, timeout)
}

// Remove client from queue
func removeFromQueue(ctx context.Context, lockId, namespace string, instance int, queueId string) pb.Errors {
	instanceStr := strconv.Itoa(instance)
	queueKey := formatKey(clientQueueKey, instanceStr, namespace, lockId)
	nextKey := formatKey(clientQueueNextKey, instanceStr, namespace, lockId)

	// Get Redis connection from pool
	redisConn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Failed to get Redis connection: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Check if queue exists
	exists, err := redisConn.Exists(queueKey)
	if err != nil {
		log.Errorf("Failed to check if queue exists: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	if !exists {
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Get current queue
	queueValue, err := redisConn.Get(queueKey)
	if err != nil {
		log.Errorf("Failed to get queue: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	items := strings.Split(queueValue, ",")

	// Find the item to remove
	var indexToRemove int = -1
	for i, item := range items {
		parts := strings.Split(item, ":")
		if len(parts) > 0 && parts[0] == queueId {
			indexToRemove = i
			break
		}
	}

	if indexToRemove < 0 {
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Check if this client is the next in queue
	exists, err = redisConn.Exists(nextKey)
	nextInQueue := ""
	if err == nil && exists {
		nextInQueue, _ = redisConn.Get(nextKey)
	}

	if nextInQueue == queueId {
		// If this client was next, we need to move to the next client
		if indexToRemove == 0 {
			return moveToNextClient(ctx, lockId, namespace, instance)
		} else {
			// This is unexpected - next in queue should be the first item
			log.Warnf("Next in queue is not the first item in the queue")
		}
	}

	// Remove the item
	if indexToRemove == 0 {
		items = items[1:]
	} else if indexToRemove == len(items)-1 {
		items = items[:indexToRemove]
	} else {
		items = append(items[:indexToRemove], items[indexToRemove+1:]...)
	}

	// Update the queue
	if len(items) == 0 {
		redisConn.Delete(queueKey)
	} else {
		newQueueValue := strings.Join(items, ",")
		err = redisConn.Set(queueKey, newQueueValue)
		if err != nil {
			log.Errorf("Failed to update queue: %v", err)
			return pb.Errors_INTERNAL_SERVER_ERROR
		}
	}

	return pb.Errors_UNKNOWN_ERROR
}

// Schedules a client id to a lock queue
func ScheduleLock(ctx context.Context, lockId, namespace string, clientID uuid.UUID, instance int) (string, pb.Errors) {
	// Ensure that the ID is not empty
	if lockId == "" {
		return "", pb.Errors_LOCK_ID_IS_EMPTY
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return "", pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}

	// Ensure that the client ID is not empty
	if clientID == uuid.Nil {
		return "", pb.Errors_CLIENT_ID_IS_EMPTY
	}

	// Ensure that the instance is valid
	instanceConfig, ok := configuration.Instances[instance]
	if !ok {
		return "", pb.Errors_INSTANCE_OUTSIDE_SERVER_RANGE
	}

	// Ensure that the namespace is valid
	_, ok = instanceConfig.Namespaces[namespace]
	if !ok {
		return "", pb.Errors_NAMESPACE_NOT_FOUND
	}

	// Add client to queue
	queueId, err := addToQueue(ctx, lockId, namespace, instance, clientID)
	if err != pb.Errors_UNKNOWN_ERROR {
		return "", err
	}

	return queueId, pb.Errors_UNKNOWN_ERROR
}

// Acquire lock if client is next in queue
func AcquireLock(ctx context.Context, lockId, namespace string, queueId string, instance int) pb.Errors {
	// Validate inputs
	if lockId == "" {
		return pb.Errors_LOCK_ID_IS_EMPTY
	}
	if namespace == "" {
		return pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}
	if queueId == "" {
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Check if client is next in queue
	isNext, err := isNextInQueue(ctx, lockId, namespace, instance, queueId)
	if err != pb.Errors_UNKNOWN_ERROR {
		return err
	}

	if !isNext {
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	// Remove client from queue
	return removeFromQueue(ctx, lockId, namespace, instance, queueId)
}

// Release lock and set the next client
func ReleaseLock(ctx context.Context, lockId, namespace string, instance int) pb.Errors {
	// Validate inputs
	if lockId == "" {
		return pb.Errors_LOCK_ID_IS_EMPTY
	}
	if namespace == "" {
		return pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}

	// Get timeout duration
	timeout, err := nextInQueueTimeout(namespace, instance)
	if err != pb.Errors_UNKNOWN_ERROR {
		return err
	}

	// Setup next client timeout
	return enterNextClientTimeoutPeriod(ctx, lockId, namespace, instance, timeout)
}

// Get queue position for a client
func GetQueuePosition(ctx context.Context, lockId, namespace string, queueId string, instance int) (int64, pb.Errors) {
	// Validate inputs
	if lockId == "" {
		return -1, pb.Errors_LOCK_ID_IS_EMPTY
	}
	if namespace == "" {
		return -1, pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}
	if queueId == "" {
		return -1, pb.Errors_INTERNAL_SERVER_ERROR
	}

	return getPositionInQueue(ctx, lockId, namespace, instance, queueId)
}
