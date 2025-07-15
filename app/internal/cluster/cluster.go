package cluster

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/mitchs-dev/dislo/internal/configuration"
)

var nodeID uuid.UUID

func init() {

	// Set the cluster node UUID
	path := configuration.Context.Cluster.IDPath
	if path == "" {
		log.Fatalf("Cluster ID path is not set")
	}
	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create the file if it doesn't exist
		file, err := os.Create(path)
		if err != nil {
			log.Fatalf("Failed to create cluster ID file: %v", err)
		}
		defer file.Close()
		// Generate a new UUID
		newUUID := uuid.New().String()
		// Write the UUID to the file
		if _, err := file.WriteString(newUUID); err != nil {
			log.Fatalf("Failed to write cluster ID to file: %v", err)
		}
		// Parse the UUID
		nodeID, err = uuid.Parse(newUUID)
		if err != nil {
			log.Fatalf("Failed to parse cluster ID: %v", err)
		}
	} else {
		// Read the UUID from the file
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read cluster ID file: %v", err)
		}
		nodeID, err = uuid.ParseBytes(data)
		if err != nil {
			log.Fatalf("Failed to parse cluster ID from file: %v", err)
		}
	}

	if !configuration.Context.Cluster.Enabled {
		log.Infof("Node running in single-node mode")
	} else {
		log.Fatalf("Cluster mode not yet implemented")
	}

}

// GetNodeID returns the node ID
func GetNodeID() uuid.UUID {
	return nodeID
}
