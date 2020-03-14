package main

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
)

var startPort = 19090
var currentPort = startPort

type Server struct {
	listeners   map[string]net.Listener
	connections map[string]shipyard.RemoteConnection_OpenServer
	logger      hclog.Logger
}

func NewServer() *Server {
	return &Server{make(map[string]net.Listener), make(map[string]shipyard.RemoteConnection_OpenServer), hclog.Default()}
}

func (s *Server) Open(rc shipyard.RemoteConnection_OpenServer) error {
	// create an id for the connection
	id := uuid.New().String()
	s.logger.Info("Open stream for", "id", id)

	// add the connetion to the collection
	s.connections[id] = rc

	// send the id back
	s.logger.Info("Send stream id", "id", id)
	err := rc.Send(&shipyard.OpenData{Id: id, Type: "hello"})
	if err != nil {
		s.logger.Error("Error sending data", "error", err)
		return err
	}

	// handle messages for the connection
	for {
		_, err := rc.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			s.logger.Error("Error receiving data", "error", err)
			// clean up the connection
			delete(s.connections, id)

			break
		}

		s.logger.Info("Got message from stream")
	}

	return nil
}

// create a new connection for a given service
func (s *Server) Create(ctx context.Context, cr *shipyard.CreateRequest) (*shipyard.CreateResponse, error) {
	id := cr.GetId()

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", currentPort))
	if err != nil {
		s.logger.Error("Unable to create listerner", "error", err)
		return &shipyard.CreateResponse{}, err
	}

	s.listeners[id] = l

	go func(id string) {
		for {
			// accept the next connection in the queue
			conn, err := l.Accept()
			if err != nil {
				s.logger.Error("Unable to accept connection", "error", err)
				continue
			}

			// work on the connection in the background to enable the next connection to be handled concurrently
			s.logger.Info("Handle new connection", "id", id)
			go func(conn net.Conn, id string) {
				for {
					data := make([]byte, 4096)

					// read 4K of data from the connnection
					// if no data left to read break
					i, err := conn.Read(data)
					if err != nil || i == 0 {
						break
					}

					// send the read chunk of data over the gRPC stream
					s.logger.Info("Read data for connection", "id", id, "data", string(data[:i]))

					// check there is a connection if not just return
					if gconn := s.connections[id]; gconn != nil {
						s.logger.Info("Sending data to stream", "id", id, "data", string(data[:i]))

						gconn.Send(&shipyard.OpenData{Id: id, Data: data[:i]})
						continue
					}

					s.logger.Info("No stream to handle data", "id", id)
				}
			}(conn, id)
		}
	}(id)

	currentPort++

	return &shipyard.CreateResponse{}, nil
}

// close a new connection for a given service
func (s *Server) Destroy(ctx context.Context, dr *shipyard.DestroyRequest) (*shipyard.NullResponse, error) {
	id := dr.GetId()
	s.listeners[id].Close()
	delete(s.listeners, id)

	return &shipyard.NullResponse{}, nil
}

// Shutdown the server and cleanup resources
func (s *Server) Shutdown() {
	for _, v := range s.listeners {
		v.Close()
	}
}
