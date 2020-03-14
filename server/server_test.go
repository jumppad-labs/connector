package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

var messageData []byte

func setupServerTests(t *testing.T) (shipyard.RemoteConnectionClient, string, func()) {

	// start the gRPC server
	s := NewServer()
	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)

	// create a listener for the server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 12344))
	assert.NoError(t, err)

	// start the server in the background
	go grpcServer.Serve(lis)

	// generate a client
	conn, err := grpc.Dial(":12344", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	assert.NoError(t, err)

	c := shipyard.NewRemoteConnectionClient(conn)

	var connID string
	// establish a stream with the server
	sc, err := c.Open(context.Background())
	assert.NoError(t, err)

	go func(sc shipyard.RemoteConnection_OpenClient) {
		for {
			msg, err := sc.Recv()
			if err != nil {
				fmt.Println("Got error message", err)
				return
			}

			if msg.Type == "hello" {
				connID = msg.Id
				fmt.Println("Got hello message")
				continue
			}

			fmt.Println("Got message", string(msg.GetData()))
			messageData = msg.GetData()
		}
	}(sc)

	// when opening a new stream ensure that we receive an id
	assert.Eventually(
		t,
		func() bool {
			return connID != ""
		},
		1*time.Second,
		10*time.Millisecond,
	)

	return c, connID, func() {
		conn.Close()
		grpcServer.Stop()
		lis.Close()
	}
}

func TestCreateOpensTCPConnectionAndStreamsData(t *testing.T) {
	c, id, td := setupServerTests(t)
	defer td()

	_, err := c.Create(context.Background(), &shipyard.CreateRequest{Id: id})
	assert.NoError(t, err)

	// test we can connect to the socket
	conn, err := net.Dial("tcp", "localhost:19090")
	assert.NoError(t, err)

	var inMessage = "abc123"

	conn.Write([]byte(inMessage))

	assert.Eventually(t,
		func() bool {
			return string(messageData) == inMessage
		},
		1*time.Second,
		10*time.Millisecond,
	)
}
