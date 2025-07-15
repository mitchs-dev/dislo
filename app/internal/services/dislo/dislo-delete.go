package dislo

import (
	"context"

	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
)

func (s *DisloServiceServer) Delete(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	// Implement create logic here
	return &pb.Response{}, nil
}
