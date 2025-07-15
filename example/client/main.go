package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mitchs-dev/dislo/pkg/sdk/client"
	log "github.com/sirupsen/logrus"
)

// Define an interface for what we need from the client
type lockClient interface {
	Lock(id, namespace, correlationID string) error
	Unlock(id, namespace, correlationID string) error
	Create(id, namespace, correlationID string) error
}

func main() {
	// Set up logging format
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Create multiple clients to demonstrate concurrent locking
	client1, err := createClient("Client-1")
	if err != nil {
		log.Fatalf("Failed to create client 1: %v", err)
	}

	client2, err := createClient("Client-2")
	if err != nil {
		log.Fatalf("Failed to create client 2: %v", err)
	}

	client3, err := createClient("Client-3")
	if err != nil {
		log.Fatalf("Failed to create client 3: %v", err)
	}

	// Define common test parameters
	namespace := "default"
	correlationId := "test-correlation-id"

	// Create locks that will be used in our demonstration
	lockIDs := []string{"shared-resource-1", "shared-resource-2", "database-access"}
	for _, id := range lockIDs {
		// We only need to create the lock once, so use client1
		err = client1.Create(id, namespace, correlationId)
		if err != nil {
			log.Errorf("Error creating lock %s: %v", id, err)
			continue
		}
		log.Infof("Lock %s created successfully", namespace+":"+id)
	}

	// Wait group to coordinate all goroutines
	var wg sync.WaitGroup

	// Scenario 1: Sequential access to a lock
	log.Info("\n--- SCENARIO 1: Sequential Access to Lock ---")
	wg.Add(1)
	go func() {
		defer wg.Done()

		lockID := "shared-resource-1"

		// Client 1 acquires lock first
		log.Infof("[Client-1] Attempting to lock %s", lockID)
		err := client1.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-1] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-1] Successfully acquired lock %s", lockID)

		// Simulate work while holding the lock
		log.Infof("[Client-1] Working with protected resource for 3 seconds...")
		time.Sleep(3 * time.Second)

		// Release the lock
		err = client1.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-1] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-1] Released lock %s", lockID)

		// Let Client 2 know it can try to acquire the lock
		time.Sleep(500 * time.Millisecond)

		// Client 2 acquires the same lock after Client 1 releases
		log.Infof("[Client-2] Attempting to lock %s", lockID)
		err = client2.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-2] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-2] Successfully acquired lock %s", lockID)

		// Shorter work simulation
		log.Infof("[Client-2] Quickly working with protected resource...")
		time.Sleep(1 * time.Second)

		// Release the lock
		err = client2.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-2] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-2] Released lock %s", lockID)
	}()

	// Scenario 2: Concurrent access attempt to the same lock
	log.Info("\n--- SCENARIO 2: Concurrent Lock Contention ---")
	wg.Add(2)

	// Lock contention demonstration
	go func() {
		defer wg.Done()

		lockID := "shared-resource-2"

		// Client 1 acquires lock first
		log.Infof("[Client-1] Attempting to lock %s", lockID)
		err := client1.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-1] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-1] Successfully acquired lock %s", lockID)

		// Long operation - hold the lock for 5 seconds
		log.Infof("[Client-1] Performing long operation with lock held (105 seconds)...")
		log.Infof("[Client-3] Attempting to lock %s (should block until Client-1 releases)", lockID)
		err = client3.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		time.Sleep(10 * time.Second)

		// Release the lock
		err = client1.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-1] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-1] Released lock %s", lockID)
		log.Infof("[Client-3] Successfully acquired lock %s after Client-1 released it", lockID)
		err = client3.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-3] Released lock %s", lockID)
	}()

	// Client 3 will try to acquire the same lock while Client 1 holds it
	go func() {
		defer wg.Done()

		lockID := "shared-resource-2"

		// Short delay to ensure Client 1 gets the lock first
		time.Sleep(1 * time.Second)

		// Client 3 attempts to acquire the same lock while Client 1 holds it
		log.Infof("[Client-3] Attempting to lock %s (should block until Client-1 releases)", lockID)
		err := client3.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-3] Successfully acquired lock %s after Client-1 released it", lockID)

		// Short work
		log.Infof("[Client-3] Performing work with the lock...")
		time.Sleep(1 * time.Second)

		// Release the lock
		err = client3.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-3] Released lock %s", lockID)
	}()

	// Scenario 3: Multiple clients accessing different locks simultaneously
	log.Info("\n--- SCENARIO 3: Parallel Access to Different Locks ---")
	wg.Add(2)

	go func() {
		defer wg.Done()

		lockID := "database-access"

		// Client 2 acquires lock
		log.Infof("[Client-2] Attempting to lock %s", lockID)
		err := client2.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-2] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-2] Successfully acquired lock %s", lockID)

		// Simulate database work
		log.Infof("[Client-2] Performing database operations...")
		time.Sleep(4 * time.Second)

		// Release the lock
		err = client2.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-2] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-2] Released lock %s", lockID)
	}()

	go func() {
		defer wg.Done()

		lockID := "shared-resource-1" // Different lock than the other goroutine

		// Client 3 acquires a different lock
		log.Infof("[Client-3] Attempting to lock %s", lockID)
		err := client3.Lock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to acquire lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-3] Successfully acquired lock %s", lockID)

		// Simulate work
		log.Infof("[Client-3] Working with this resource...")
		time.Sleep(2 * time.Second)

		// Release the lock
		err = client3.Unlock(lockID, namespace, correlationId)
		if err != nil {
			log.Errorf("[Client-3] Failed to release lock %s: %v", lockID, err)
			return
		}
		log.Infof("[Client-3] Released lock %s", lockID)
	}()

	// Wait for all scenarios to complete
	wg.Wait()
	log.Info("All lock demonstration scenarios completed.")
}

// createClient creates a new distributed lock client with a unique name for demonstration
func createClient(name string) (lockClient, error) {
	// Create a deterministic UUID based on the client name for this demo
	// In a real application, you'd typically use random UUIDs for each client
	clientID := uuid.New()
	log.Infof("ID for %s: %s", name, clientID.String())

	// Create a new client context that connects to the lock server
	// Parameters:
	// - host: The lock server host
	// - port: The lock server port
	// - skipTls: Whether to skip TLS (true for insecure connections)
	// - instance: Instance number to use (1-9)
	// - clientID: The unique identifier for this client
	return client.NewContext("localhost", 5900, true, 1, clientID), nil
}
