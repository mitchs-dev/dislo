package dislo

import (
	"context"
	"fmt"

	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
)

func (s *DisloServiceServer) Unlock(ctx context.Context, req *pb.Request) (*pb.Response, error) {
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

	response := &pb.Response{
		Error:         pb.Errors_UNKNOWN_ERROR, // Default to success
		CorrelationId: correlationId,
		Locks:         make([]*pb.Lock, 0, len(req.Locks)),
	}

	// Process each lock in the request
	for _, lock := range req.Locks {
		formattedLockId := fmt.Sprintf("%s:%s", lock.Namespace, lock.Id) // Simple format for logging
		responseLock := &pb.Lock{
			Id:        lock.Id,
			Namespace: lock.Namespace,
		}

		// First check if the lock exists
		exists, err := checkForLockInInstance(lock.Id, lock.Namespace, instance)
		if err != pb.Errors_UNKNOWN_ERROR {
			log.Warnf("Error checking lock (C: %s | IP: %s | Lock: %s): %v",
				correlationId, clientIP, formattedLockId, err)

			response.Error = err
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with UNKNOWN status
			responseLock.Status = pb.LockStatus_UNKNOWN_STATUS
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		if !exists {
			log.Warnf("Lock does not exist (C: %s | IP: %s | Lock: %s)",
				correlationId, clientIP, formattedLockId)

			response.Error = pb.Errors_LOCK_NOT_FOUND
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with UNKNOWN status
			responseLock.Status = pb.LockStatus_UNKNOWN_STATUS
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		// Check current lock status
		status, statusErr := getLockStatus(lock.Id, lock.Namespace, instance)
		if statusErr != pb.Errors_UNKNOWN_ERROR {
			log.Warnf("Error getting lock status (C: %s | IP: %s | Lock: %s): %v",
				correlationId, clientIP, formattedLockId, statusErr)

			response.Error = statusErr
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with UNKNOWN status
			responseLock.Status = pb.LockStatus_UNKNOWN_STATUS
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		// Only locked locks can be unlocked
		if status == pb.LockStatus_UNLOCKED {

			log.Infof("Lock already unlocked (C: %s | IP: %s | Lock: %s)",
				correlationId, clientIP, formattedLockId)
			responseLock.Status = pb.LockStatus_UNLOCKED
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		} else if status == pb.LockStatus_UNKNOWN_STATUS {
			log.Warnf("Lock status is unknown (C: %s | IP: %s | Lock: %s)",
				correlationId, clientIP, formattedLockId)

			response.Error = pb.Errors_UNKNOWN_ERROR
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with UNKNOWN status
			responseLock.Status = pb.LockStatus_UNKNOWN_STATUS
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		// Unlock the lock - this will also notify the queue manager
		unlockErr := updateLockInInstance(lock.Id, lock.Namespace, instance, lockActionUnlock)
		if unlockErr != pb.Errors_UNKNOWN_ERROR {
			log.Errorf("Error unlocking lock (C: %s | IP: %s | Lock: %s): %v",
				correlationId, clientIP, formattedLockId, unlockErr)

			response.Error = unlockErr
			response.FailedOnLock = formattedLockId

			// Add the failed lock to response with its current status
			responseLock.Status = status
			response.Locks = append(response.Locks, responseLock)

			return response, nil
		}

		log.Infof("Lock unlocked (C: %s | IP: %s | Lock: %s)", correlationId, clientIP, formattedLockId)

		responseLock.Status = pb.LockStatus_UNLOCKED
		response.Locks = append(response.Locks, responseLock)
	}

	return response, nil
}
