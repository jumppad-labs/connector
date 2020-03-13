package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func setupServerTests() (shipyard.RemoteConnectionClient, func()) {
	s := NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 12344))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)
	//... // determine whether to use TLS
	go grpcServer.Serve(lis)

	// generate a client
	conn, err := grpc.Dial(":12344", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		panic(err)
	}

	c := shipyard.NewRemoteConnectionClient(conn)

	return c, func() {
		conn.Close()
		grpcServer.Stop()
		lis.Close()
	}
}

func TestOpenOpensTCPConnection(t *testing.T) {
	c, td := setupServerTests()
	defer td()

	co, err := c.Create(context.Background(), &shipyard.CreateRequest{})
	assert.NoError(t, err)

	// Test Id
	assert.NotEmpty(t, co.GetId())

	// test we can connect to the socket
	_, err = net.Dial("tcp", "localhost:19090")
	assert.NoError(t, err)
}
