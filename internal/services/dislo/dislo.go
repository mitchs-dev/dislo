package dislo

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchs-dev/dislo/internal/configuration"
	"github.com/mitchs-dev/dislo/internal/handler"
	"github.com/mitchs-dev/dislo/internal/queueManager"
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	log "github.com/sirupsen/logrus"
)

var (
	lockKey                   = "$INSTANCE:$NAMESPACE:$ID:lock" // The key for the lock (in redis)
	defaultNextInQueueTimeout = "15s"
)

type lockActions int

const (
	lockActionCreate lockActions = iota
	lockActionLock
	lockActionUnlock
	lockActionDelete
)

func fmtLockKey(id, namespace string, instance int) string {
	instanceStr := strconv.Itoa(instance)
	key := strings.ReplaceAll(lockKey, "$INSTANCE", instanceStr)
	key = strings.ReplaceAll(key, "$NAMESPACE", namespace)
	key = strings.ReplaceAll(key, "$ID", id)
	return key
}

func checkForLockInInstance(id, namespace string, instance int) (bool, pb.Errors) {
	// Ensure that the ID is not empty
	if id == "" {
		return false, pb.Errors_LOCK_ID_IS_EMPTY
	}
	// Ensure that the instance is not the management db
	if instance == configuration.Context.Cluster.ClusterManagement.DB {
		return false, pb.Errors_INSTANCE_IS_RESERVED
	}

	// Ensure that the namespace is not empty
	if namespace == "" {
		return false, pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}

	// Ensure that the instance is valid
	instanceConfig, ok := configuration.Instances[instance]
	if !ok {
		return false, pb.Errors_INSTANCE_OUTSIDE_SERVER_RANGE
	}

	// Ensure that the namespace is valid
	_, ok = instanceConfig.Namespaces[namespace]
	if !ok {
		return false, pb.Errors_NAMESPACE_NOT_FOUND
	}

	conn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	exists, err := conn.Exists(fmtLockKey(id, namespace, instance))
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return false, pb.Errors_INTERNAL_SERVER_ERROR
	}

	// For debugging purposes, we don't directly return the results of the Redis call
	log.Debugf("Lock check for ID %s in namespace %s on instance %d: %v", id, namespace, instance, exists)
	return exists, pb.Errors_UNKNOWN_ERROR
}

// get the lock status for a given lock ID
func getLockStatus(lockId, namespace string, instance int) (pb.LockStatus, pb.Errors) {
	// Ensure that the ID is not empty
	if lockId == "" {
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_LOCK_ID_IS_EMPTY
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}

	// Ensure that the instance is valid
	instanceConfig, ok := configuration.Instances[instance]
	if !ok {
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_INSTANCE_OUTSIDE_SERVER_RANGE
	}

	// Ensure that the namespace is valid
	_, ok = instanceConfig.Namespaces[namespace]
	if !ok {
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_NAMESPACE_NOT_FOUND
	}

	conn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_INTERNAL_SERVER_ERROR
	}

	status, err := conn.Get(fmtLockKey(lockId, namespace, instance))
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_INTERNAL_SERVER_ERROR
	}

	if status == "" {
		log.Errorf("Lock status is empty for ID %s in namespace %s on instance %d", lockId, namespace, instance)
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_INTERNAL_SERVER_ERROR
	}
	// Convert the status to a LockStatus enum
	var lockStatus pb.LockStatus
	switch status {
	case pb.LockStatus_PENDING_CREATION.String():
		lockStatus = pb.LockStatus_PENDING_CREATION
	case pb.LockStatus_LOCKED.String():
		lockStatus = pb.LockStatus_LOCKED
	case pb.LockStatus_UNLOCKED.String():
		lockStatus = pb.LockStatus_UNLOCKED
	case pb.LockStatus_PENDING_DELETION.String():
		lockStatus = pb.LockStatus_PENDING_DELETION
	default:
		log.Errorf("Unknown lock status: %s", status)
		return pb.LockStatus_UNKNOWN_STATUS, pb.Errors_INTERNAL_SERVER_ERROR
	}
	log.Debugf("Lock status for ID %s in namespace %s on instance %d: %s", lockId, namespace, instance, lockStatus.String())
	return lockStatus, pb.Errors_UNKNOWN_ERROR
}

func updateLockInInstance(id, namespace string, instance int, action lockActions) pb.Errors {
	// Ensure that the ID is not empty
	if id == "" {
		return pb.Errors_LOCK_ID_IS_EMPTY
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return pb.Errors_LOCK_NAMESPACE_IS_EMPTY
	}

	// Ensure that the instance is valid
	instanceConfig, ok := configuration.Instances[instance]
	if !ok {
		return pb.Errors_INSTANCE_OUTSIDE_SERVER_RANGE
	}

	// Ensure that the namespace is valid
	_, ok = instanceConfig.Namespaces[namespace]
	if !ok {
		return pb.Errors_NAMESPACE_NOT_FOUND
	}

	conn, err := handler.RedisConnection(instance)
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	var status string
	switch action {
	case lockActionCreate:
		status = pb.LockStatus_PENDING_CREATION.String()
	case lockActionLock:
		status = pb.LockStatus_LOCKED.String()
	case lockActionUnlock:
		status = pb.LockStatus_UNLOCKED.String()
	case lockActionDelete:
		status = pb.LockStatus_PENDING_DELETION.String()
	}

	log.Debugf("updateLockInInstance status: %s", status)
	// Set the lock status so that if we are creating/deleting, the status is set to pending
	err = conn.Set(fmtLockKey(id, namespace, instance), status)
	if err != nil {
		log.Errorf("Error during request: %v", err)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	var actionErr error
	var createOrDelete string
	switch action {
	case lockActionCreate:
		setUnlocked := pb.LockStatus_UNLOCKED.String()
		actionErr = conn.Set(fmtLockKey(id, namespace, instance), setUnlocked)
		createOrDelete = "created"
	case lockActionDelete:
		actionErr = conn.Delete(fmtLockKey(id, namespace, instance))
		createOrDelete = "deleted"
	case lockActionUnlock:
		// When unlocking, notify the queue manager to allow the next client in queue
		ctx := context.Background()
		queueErr := queueManager.ReleaseLock(ctx, id, namespace, instance)
		if queueErr != pb.Errors_UNKNOWN_ERROR {
			log.Errorf("Error notifying queue manager after unlock: %v", queueErr)
			// We continue since the lock is already unlocked
		}
		log.Infof("Lock %s set to %s and queue notified", fmtLockKey(id, namespace, instance), status)
		return pb.Errors_UNKNOWN_ERROR
	default:
		log.Infof("Lock %s set to %s", fmtLockKey(id, namespace, instance), status)
		return pb.Errors_UNKNOWN_ERROR
	}
	if actionErr != nil {
		log.Errorf("Error during request: %v", actionErr)
		return pb.Errors_INTERNAL_SERVER_ERROR
	}

	log.Debugf("updateLockInInstance 2nd status: %s", status)

	log.Infof("Lock %s was %s", fmtLockKey(id, namespace, instance), createOrDelete)

	return pb.Errors_UNKNOWN_ERROR
}

// LockWithQueue tries to acquire a lock, and if failed, adds the client to the queue
func LockWithQueue(ctx context.Context, lockId, namespace string, clientID uuid.UUID, instance int) (string, int64, pb.Errors) {
	// First check if the lock exists
	exists, err := checkForLockInInstance(lockId, namespace, instance)
	if err != pb.Errors_UNKNOWN_ERROR {
		return "", -1, err
	}

	if !exists {
		return "", -1, pb.Errors_LOCK_NOT_FOUND
	}

	// Try to get the lock status
	status, err := getLockStatus(lockId, namespace, instance)
	if err != pb.Errors_UNKNOWN_ERROR {
		return "", -1, err
	}

	// If the lock is already locked, add client to queue
	if status == pb.LockStatus_LOCKED {
		// Add client to queue
		queueId, err := queueManager.ScheduleLock(ctx, lockId, namespace, clientID, instance)
		if err != pb.Errors_UNKNOWN_ERROR {
			return "", -1, err
		}

		// Get the position in queue
		position, err := queueManager.GetQueuePosition(ctx, lockId, namespace, queueId, instance)
		if err != pb.Errors_UNKNOWN_ERROR {
			return queueId, -1, err
		}

		return queueId, position, pb.Errors_UNKNOWN_ERROR
	}

	// If the lock is unlocked, try to acquire it
	if status == pb.LockStatus_UNLOCKED {
		// Set the lock status to locked
		err = updateLockInInstance(lockId, namespace, instance, lockActionLock)
		if err != pb.Errors_UNKNOWN_ERROR {
			return "", -1, err
		}
		return "", 0, pb.Errors_UNKNOWN_ERROR
	}

	// Lock is in an unexpected state
	return "", -1, pb.Errors_INTERNAL_SERVER_ERROR
}

// TryAcquireLock tries to acquire a lock if the client is next in queue
func TryAcquireLock(ctx context.Context, lockId, namespace string, queueId string, instance int) pb.Errors {
	// First check if the client is next in queue
	err := queueManager.AcquireLock(ctx, lockId, namespace, queueId, instance)
	if err != pb.Errors_UNKNOWN_ERROR {
		return err
	}

	// Set lock to locked state
	return updateLockInInstance(lockId, namespace, instance, lockActionLock)
}

// GetQueuePosition returns the client's position in the queue
func GetQueuePosition(ctx context.Context, lockId, namespace string, queueId string, instance int) (int64, pb.Errors) {
	return queueManager.GetQueuePosition(ctx, lockId, namespace, queueId, instance)
}
