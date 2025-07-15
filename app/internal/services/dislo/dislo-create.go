package dislo

import (
	"context"

	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
)

func (s *DisloServiceServer) Create(ctx context.Context, req *pb.Request) (*pb.Response, error) {

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

	if req.Locks == nil {
		log.Warnf("No locks provided in request (C: %s | IP: %s)", correlationId, clientIP)
		return &pb.Response{
			Error: pb.Errors_NO_LOCKS_PROVIDED,
		}, nil
	}

	instance := int(req.Instance)

	for _, lock := range req.Locks {

		formattedLockId := fmtLockKey(lock.Id, lock.Namespace, int(req.Instance))

		exists, err := checkForLockInInstance(lock.Id, lock.Namespace, instance)
		if err != pb.Errors_UNKNOWN_ERROR {
			return &pb.Response{
				Error:        err,
				FailedOnLock: formattedLockId,
			}, nil
		}

		if exists {
			log.Warnf("Lock already exists (C: %s | IP: %s | Lock: %s)", correlationId, clientIP, formattedLockId)
			return &pb.Response{
				Error:        pb.Errors_LOCK_ALREADY_EXISTS,
				FailedOnLock: formattedLockId,
			}, nil
		}
		log.Infof("Creating lock (C: %s | IP: %s | Lock: %s)", correlationId, clientIP, formattedLockId)
		err = updateLockInInstance(lock.Id, lock.Namespace, instance, lockActionCreate)
		if err != pb.Errors_UNKNOWN_ERROR {
			return &pb.Response{
				Error:        pb.Errors_UNKNOWN_ERROR,
				FailedOnLock: formattedLockId,
			}, nil
		}
		log.Infof("Lock created (C: %s | IP: %s | Lock: %s)", correlationId, clientIP, formattedLockId)
	}

	log.Infof("All locks created successfully (C: %s | IP: %s)", correlationId, clientIP)
	return &pb.Response{
		CorrelationId: correlationId,
		Locks:         req.Locks,
	}, nil
}
