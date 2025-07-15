package dislo

import (
	"context"

	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
)

func (s *DisloServiceServer) List(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	// Implement list logic here
	return &pb.Response{}, nil
}
