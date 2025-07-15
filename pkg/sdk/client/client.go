// package: github.com/mitchs-dev/dislo/pkg/client
/*
This is the client API for the Dislo service.
*/
package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Context for the client to use when connecting to the Dislo service
type clientContext struct {
	instance int             // Instance number to use within the Dislo service (0-9)
	procCtx  context.Context // Context for the client
	grpcCtx  pb.DisloClient  // gRPC client clientContext
	clientID uuid.UUID       // Client ID for the client
}

// Creates a new client clientContext
func NewContext(host string, port int, skipTls bool, instance int, clientID uuid.UUID) *clientContext {

	var grpcOpts []grpc.DialOption
	if skipTls {
		insecureCreds := grpc.WithTransportCredentials(insecure.NewCredentials())
		grpcOpts = []grpc.DialOption{
			insecureCreds,
		}
	}

	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", host, port), grpcOpts...)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Dislo service: %v", err))
	}
	// Create a new gRPC client
	client := pb.NewDisloClient(conn)
	if client == nil {
		panic("Failed to create gRPC client")
	}

	// Create a new process context
	ctx := context.Background()

	return &clientContext{
		instance: instance,
		procCtx:  ctx,
		grpcCtx:  client,
		clientID: clientID,
	}
}

// Lock a lock in the Dislo service
func (c *clientContext) Lock(id, namespace, correlationID string) error {

	// Ensure that the ID is not empty
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Ensure that the correlation ID is not empty
	if correlationID == "" {
		return fmt.Errorf("correlation ID cannot be empty")
	}

	response, err := c.grpcCtx.Lock(c.procCtx, &pb.Request{
		CorrelationId: correlationID,
		Locks: []*pb.Lock{
			{
				Id:        id,
				Namespace: namespace,
			},
		},
		ClientId: c.clientID.String(),
		Instance: int32(c.instance),
	})

	if err != nil {
		return fmt.Errorf("failed to create lock: %v", err)
	}

	if response.Error != pb.Errors_UNKNOWN_ERROR {
		return fmt.Errorf("failed to create lock: %s", response.Error)
	}

	return nil

}

// Unlock a lock in the Dislo service
func (c *clientContext) Unlock(id, namespace, correlationID string) error {
	// Ensure that the ID is not empty
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Ensure that the correlation ID is not empty
	if correlationID == "" {
		return fmt.Errorf("correlation ID cannot be empty")
	}

	response, err := c.grpcCtx.Unlock(c.procCtx, &pb.Request{
		CorrelationId: correlationID,
		Locks: []*pb.Lock{
			{
				Id:        id,
				Namespace: namespace,
			},
		},
		ClientId: c.clientID.String(),
		Instance: int32(c.instance),
	})
	if err != nil {
		return fmt.Errorf("failed to create lock: %v", err)
	}
	if response.Error != pb.Errors_UNKNOWN_ERROR {
		return fmt.Errorf("failed to create lock: %s", response.Error)
	}
	return nil
}

// Create a new lock in the Dislo service without labels
func (c *clientContext) Create(id, namespace, correlationID string) error {
	// Ensure that the ID is not empty
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Ensure that the correlation ID is not empty
	if correlationID == "" {
		return fmt.Errorf("correlation ID cannot be empty")
	}

	response, err := c.grpcCtx.Create(c.procCtx, &pb.Request{
		CorrelationId: correlationID,
		Locks: []*pb.Lock{
			{
				Id:        id,
				Namespace: namespace,
			},
		},
		ClientId: c.clientID.String(),
		Instance: int32(c.instance),
	})
	if err != nil {
		return fmt.Errorf("failed to create lock: %v", err)
	}
	if response.Error != pb.Errors_UNKNOWN_ERROR {
		return fmt.Errorf("failed to create lock: %s", response.Error)
	}
	return nil
}

// Create a new lock in the Dislo service with labels
func (c *clientContext) CreateWithLabels(id, namespace string, labels []pb.Label) error {
	// Ensure that the ID is not empty
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	// Ensure that the labels are not empty
	if len(labels) == 0 {
		return fmt.Errorf("labels cannot be empty")
	}
	// Implement create with labels logic here
	return nil
}

// Delete a lock in the Dislo service
func (c *clientContext) Delete(id, namespace string) error {
	// Ensure that the ID is not empty
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	// Implement delete logic here
	return nil
}

// Get the status of a lock in the Dislo service
func (c *clientContext) Status(id, namespace string) ([]pb.Lock, error) {
	// Ensure that the ID is not empty
	if id == "" {
		return nil, fmt.Errorf("ID cannot be empty")
	}
	// Ensure that the namespace is not empty
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	// Implement status logic here

	return nil, nil
}

// Lists statuses via labels instead of ID
func (c *clientContext) StatusByLabels(namespace string, labels []pb.Label) ([]pb.Lock, error) {
	// Ensure that the namespace is not empty
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	// Ensure that the labels are not empty
	if len(labels) == 0 {
		return nil, fmt.Errorf("labels cannot be empty")
	}
	// Implement status by labels logic here
	return nil, nil
}

// Lists all locks in the Dislo service namespace
func (c *clientContext) List(namespace string) ([]string, error) {
	// Ensure that the namespace is not empty
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	// Implement list logic here
	return nil, nil
}

// Lists all locks in the Dislo service namespace with labels
func (c *clientContext) ListWithLabels(namespace string, labels []pb.Label) ([]string, error) {
	// Ensure that the namespace is not empty
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	// Ensure that the labels are not empty
	if len(labels) == 0 {
		return nil, fmt.Errorf("labels cannot be empty")
	}
	// Implement list with labels logic here
	return nil, nil
}
