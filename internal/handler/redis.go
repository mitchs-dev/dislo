package handler

import (
	"fmt"
	"sync"

	"github.com/mitchs-dev/dislo/internal/configuration"
	"github.com/mitchs-dev/library-go/redisTools"
	log "github.com/sirupsen/logrus"
)

// RedisConnectionPool is a pool of Redis connections
//
// We opt to pool connections instead of creating a new one for each request to improve performance.
var redisConnectionPool map[int]*redisDisloConfig

type redisDisloConfig struct {
	client *redisTools.RedisClient
	hold   *sync.Mutex
}

// RedisConnection returns a Redis connection for the given instance
func RedisConnection(instance int) (*redisDisloConfig, error) {

	_, ok := configuration.Instances[instance]
	if !ok {
		return nil, fmt.Errorf("instance %d not found in configuration", instance)
	}

	if redisConnectionPool == nil {
		redisConnectionPool = make(map[int]*redisDisloConfig)
	}
	context, ok := redisConnectionPool[instance]

	if !ok {
		log.Debugf("Creating new Redis connection for instance %d", instance)
		var err error
		context, err = generateNewRedisConnection(instance)
		if err != nil {
			return nil, err
		}
		redisConnectionPool[instance] = context
		log.Debugf("New Redis connection created for instance %d: %v", instance, context)
	} else {
		log.Debugf("Re-using existing Redis connection for instance %d: %v", instance, context)
	}

	return context, nil
}

// generateNewRedisConnection creates a new Redis connection for the given instance
func generateNewRedisConnection(instance int) (*redisDisloConfig, error) {
	log.Debugf("Creating new Redis connection for instance %d", instance)
	redisConfig := redisTools.RedisConfiguration{}
	redisHostConfig := redisTools.RedisConfigHost{
		Addr:     configuration.Context.Redis.Host + ":" + fmt.Sprint(configuration.Context.Redis.Port),
		Password: configuration.Context.Redis.Password,
		DB:       instance,
	}
	redisConfig.Host = redisHostConfig

	client := redisTools.NewRedisClient(redisConfig)
	if client == nil {
		log.Fatalf("Error creating Redis client for instance %d", instance)
		return nil, fmt.Errorf("error creating Redis client for instance %d", instance)
	}
	context := &redisDisloConfig{
		client: client,
		hold:   &sync.Mutex{},
	}
	// Test Connection
	err := redisTools.TestConnection(redisConfig)
	if err != nil {
		log.Errorf("Error creating Redis connection for instance %d: %v", instance, err)
		return nil, err
	}
	log.Debugf("Redis connection created: %v", context)
	return context, nil
}

// CloseSingleRedisConnection closes a single Redis connection for the given instance
func CloseSingleRedisConnection(instance int) {
	log.Debugf("Closing Redis connection for instance %d", instance)
	context, ok := redisConnectionPool[instance]
	if ok {
		err := context.client.Close()
		if err != nil {
			log.Errorf("Error closing Redis connection for instance %d: %v", instance, err)
		}
		delete(redisConnectionPool, instance)
		log.Debugf("Redis connection for instance %d closed", instance)
	} else {
		log.Debugf("No Redis connection found for instance %d", instance)
	}
}

// CloseRedisConnections closes all Redis connections in the pool
func CloseRedisConnections() {
	log.Debugf("Closing all Redis connections in the pool")
	for instance, context := range redisConnectionPool {
		log.Debugf("Closing Redis connection for instance %d: %v", instance, context)
		err := context.client.Close()
		if err != nil {
			log.Errorf("Error closing Redis connection for instance %d: %v", instance, err)
		}
		delete(redisConnectionPool, instance)
	}
	log.Debugf("All Redis connections closed")
}

func (c *redisDisloConfig) Get(key string) (string, error) {
	c.hold.Lock()
	defer c.hold.Unlock()
	return c.client.Get(key)
}

func (c *redisDisloConfig) Set(key string, value string) error {
	c.hold.Lock()
	defer c.hold.Unlock()
	log.Debugf("Setting key %s to value %s", key, value)
	return c.client.Set(key, value, 0)
}

func (c *redisDisloConfig) Exists(key string) (bool, error) {
	c.hold.Lock()
	defer c.hold.Unlock()
	return c.client.Exists(key)
}

func (c *redisDisloConfig) Delete(key string) error {
	c.hold.Lock()
	defer c.hold.Unlock()
	return c.client.Del(key)
}
