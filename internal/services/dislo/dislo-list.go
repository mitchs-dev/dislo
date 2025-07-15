package dislo

import (
	"context"

	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
)

func (s *DisloServiceServer) Status(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	// Implement status logic here
	return &pb.Response{}, nil
}
