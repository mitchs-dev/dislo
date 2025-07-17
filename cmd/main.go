package main

import (
	"fmt"
	"net"

	"github.com/mitchs-dev/dislo/internal/configuration"
	"github.com/mitchs-dev/dislo/internal/version"
	log "github.com/sirupsen/logrus"

	"github.com/mitchs-dev/dislo/internal/services/dislo"
	pb "github.com/mitchs-dev/dislo/pkg/generated/dislo"

	"google.golang.org/grpc"
)

func main() {

	log.Infof("Dislo version: %s", version.Version)

	c := configuration.Context
	// Set up a listener
	listener := c.Server.Host + ":" + fmt.Sprint(c.Server.Port)
	lis, err := net.Listen("tcp", listener)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()

	// Register the Dislo service
	disloService := dislo.NewDisloService()
	pb.RegisterDisloServer(s, disloService)

	log.Infof("Listening on %s", listener)

	// Start the server
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
