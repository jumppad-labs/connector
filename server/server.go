package main

import (
	"context"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/shipyard-run/connector/protos/shipyard"
)

var startPort = 19090
var currentPort = startPort

type Server struct {
	listeners   map[string]net.Listener
	connections map[string]shipyard.RemoteConnection_OpenServer
}

func NewServer() *Server {
	return &Server{make(map[string]net.Listener), make(map[string]shipyard.RemoteConnection_OpenServer)}
}

func (s *Server) Open(or *shipyard.OpenRequest, rc shipyard.RemoteConnection_OpenServer) error {
	// listen on TCP connection
	// when data
	// send over rc
	s.connections[or.GetId()] = rc
	return nil
}

// create a new connection for a given service
func (s *Server) Create(ctx context.Context, cr *shipyard.CreateRequest) (*shipyard.CreateResponse, error) {
	// open up inbound TCP connection
	// return an id
	id := uuid.New().String()
	// generate a random TCP connection
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", currentPort))
	if err != nil {
		panic(err)
	}

	s.listeners[id] = l

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				// handle error
				panic(err)
			}

			go func(conn net.Conn) {
				for {
					data := make([]byte, 4096)
					i, err := conn.Read(data)
					if err != nil || i == 0 {
						break
					}

					// s.connections[id].Send(data[:i])
					fmt.Printf("handle conn: %#v\n", string(data[:i]))
				}
			}(conn)
		}
	}()

	currentPort++

	return &shipyard.CreateResponse{
		Id: id,
	}, nil
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
