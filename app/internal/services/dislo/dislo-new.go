package dislo

import (
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"
)

func NewDisloService() *DisloServiceServer {
	return &DisloServiceServer{}
}

type DisloServiceServer struct {
	pb.UnimplementedDisloServer
}
