package dislo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
)

func (s *DisloServiceServer) Lock(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	if req.CorrelationId == "" {
		log.Warnf("Correlation ID is required")
		return &pb.Response{
			Error: pb.Errors_NO_CORRELATION_ID_PROVIDED,
		}, nil
	}
	correlationId := req.CorrelationId

	// Get the client's IP address
	clientIP := "Unknown"
	p, ok := peer.FromContext(ctx)
	if ok {
		clientIP = p.Addr.String()
	}

	if req.Locks == nil || len(req.Locks) == 0 {
		log.Warnf("No locks provided in request (C: %s | IP: %s)", correlationId, clientIP)
		return &pb.Response{
			Error:         pb.Errors_NO_LOCKS_PROVIDED,
			CorrelationId: correlationId,
		}, nil
	}

	instance := int(req.Instance)

	// Parse client ID
	var clientID uuid.UUID
	var err error
	if req.ClientId != "" {
		clientID, err = uuid.Parse(req.ClientId)
		if err != nil {
			log.Warnf("Invalid client ID format (C: %s | IP: %s): %v", correlationId, clientIP, err)
			return &pb.Response{
				Error:         pb.Errors_CLIENT_ID_NOT_IN_UUID_FORMAT,
				CorrelationId: correlationId,
			}, nil
		}
	} else {
		log.Warnf("Client ID is required (C: %s | IP: %s)", correlationId, clientIP)
		return &pb.Response{
			Error:         pb.Errors_CLIENT_ID_IS_EMPTY,
			CorrelationId: correlationId,
		}, nil
	}

	response := &pb.Response{
		Error:         pb.Errors_UNKNOWN_ERROR, // Default to success
		CorrelationId: correlationId,
		Locks:         make([]*pb.Lock, 0, len(req.Locks)),
	}

	// Track the highest queue position for reporting
	highestQueuePosition := int32(-1)

	// Process each lock in the request
	for _, lock := range req.Locks {
		formattedLockId := fmt.Sprintf("%s:%s", lock.Namespace, lock.Id) // Simple format for logging
		responseLock := &pb.Lock{
			Id:        lock.Id,
			Namespace: lock.Namespace,
		}

		// Try to acquire the lock or get queued
		queueId, position, queueErr := LockWithQueue(ctx, lock.Id, lock.Namespace, clientID, instance)

		// Handle errors
		if queueErr != pb.Errors_UNKNOWN_ERROR {
			if queueErr == pb.Errors_LOCK_NOT_FOUND {
				log.Warnf("Lock does not exist (C: %s | IP: %s | Lock: %s)",
					correlationId, clientIP, formattedLockId)
			} else {
				log.Errorf("Error acquiring lock (C: %s | IP: %s | Lock: %s): %v",
					correlationId, clientIP, formattedLockId, queueErr)
			}

			response.Error = queueErr
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with UNKNOWN status
			responseLock.Status = pb.LockStatus_UNKNOWN_STATUS
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		// If queueId is empty, the lock was acquired immediately
		if queueId == "" {
			log.Infof("Lock acquired (C: %s | IP: %s | Lock: %s)",
				correlationId, clientIP, formattedLockId)

			responseLock.Status = pb.LockStatus_LOCKED
		} else {
			// Client was added to the queue
			log.Infof("Client queued (C: %s | IP: %s | Lock: %s | Position: %d | QueueID: %s)",
				correlationId, clientIP, formattedLockId, position, queueId)

			// Add the queue ID to the lock labels
			queueLabel := &pb.Label{
				Key:   "queue_id",
				Value: queueId,
			}

			responseLock.Status = pb.LockStatus_LOCKED // From client's perspective
			responseLock.Labels = []*pb.Label{queueLabel}

			// Track highest queue position
			if int32(position) > highestQueuePosition {
				highestQueuePosition = int32(position)
			}
		}

		response.Locks = append(response.Locks, responseLock)
	}

	// Set the highest queue position in the response
	if highestQueuePosition >= 0 {
		response.QueuePosition = highestQueuePosition
	}

	return response, nil
}
