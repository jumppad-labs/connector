package remote

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	listeners   map[string]net.Listener
	tcpConn     map[string]net.Conn
	connections map[string]shipyard.RemoteConnection_OpenServer
	logger      hclog.Logger
}

func NewServer(log hclog.Logger) *Server {
	return &Server{
		make(map[string]net.Listener),
		make(map[string]net.Conn),
		make(map[string]shipyard.RemoteConnection_OpenServer),
		log,
	}
}

/*
func (s *Server) CallRemote(svr shipyard.RemoteConnection_CallRemoteServer) error {
	// listen for inbound messages
	for {
		msg, err := svr.Recv()
		if err != nil {
			return err
		}

		log.Debug("New request for", "location", msg.GetLocation())
	}

	log.Debug("Request done")

	return nil
}
*/

// ExposeLocalService opens a TCP port for a local service
func (s *Server) ExposeLocalService(ctx context.Context, cr *shipyard.ExposeRequest) (*shipyard.ExposeResponse, error) {
	id := uuid.New().String()

	s.logger.Info("Exposing Local Service", "name", cr.Name, "id", id)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cr.Port))
	if err != nil {
		s.logger.Error("Unable to create listener", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.listeners[id] = l

	go s.tcpListen(id)

	return &shipyard.ExposeResponse{Id: id}, nil
}

// DestroyLocalService a new connection for a given service
func (s *Server) DestroyLocalService(ctx context.Context, dr *shipyard.DestroyRequest) (*shipyard.NullResponse, error) {
	s.logger.Info("Delete Local Service", "id", dr.Id)

	s.listeners[dr.Id].Close()
	delete(s.listeners, dr.Id)

	return &shipyard.NullResponse{}, nil
}

func (s *Server) Open(rc shipyard.RemoteConnection_OpenServer) error {
	// add the connetion to the collection
	s.connections["a"] = rc

	// handle messages for the connection
	for {
		msg, err := rc.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			s.logger.Error("Error receiving data", "error", err)
			// clean up the connection
			delete(s.connections, "a")

			break
		}

		s.logger.Debug("Got message from stream", "id", msg.ServiceId, "rid", msg.RequestId, "type", msg.Type)

		// attempt top get a local listener for the message
		switch msg.Type {
		case shipyard.MessageType_DATA:
			if conn, ok := s.tcpConn[msg.RequestId]; ok {
				conn.Write(msg.Data)
			}
		case shipyard.MessageType_READ_DONE:
			if conn, ok := s.tcpConn[msg.RequestId]; ok {
				conn.Close()
			}
			delete(s.tcpConn, msg.RequestId)
		case shipyard.MessageType_ERROR:
			s.logger.Error("Error from remote endpoint", "message", msg)

			if conn, ok := s.tcpConn[msg.RequestId]; ok {
				conn.Close()
			}
			delete(s.tcpConn, msg.RequestId)
		}
	}

	return nil
}

// tcpListen listens on a local port and attempts to send the data over a gRPC connection
func (s *Server) tcpListen(id string) {
	l := s.listeners[id]
	for {
		// accept the next connection in the queue
		conn, err := l.Accept()
		if err != nil {
			s.logger.Error("Unable to accept connection", "error", err)
			break
		}

		// work on the connection in the background to enable the next connection to be handled concurrently
		s.logger.Debug("Handle new connection", "id", id)
		rid := uuid.New().String() // generate a new request id
		s.tcpConn[rid] = conn

		// send the new connection message
		s.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_NEW_CONNECTION})

		go func(conn net.Conn, id string, rid string) {
			for {
				maxBuffer := 4096
				data := make([]byte, maxBuffer)

				// read 4K of data from the connection
				// if no data left to read break
				s.logger.Debug("Starting read", "service", id, "rid", rid)

				i, err := conn.Read(data)
				if err != nil || i == 0 {
					s.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_ERROR})
					break
				}

				// send the read chunk of data over the gRPC stream
				s.logger.Debug("Read data for connection", "service", id, "rid", rid, "len", i, "data", string(data[:i]))

				// check there is a connection if not just return
				if gconn := s.connections["a"]; gconn != nil {
					s.logger.Debug("Sending data to stream", "service", id, "rid", rid, "data", string(data[:i]))

					gconn.Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_DATA, Data: data[:i]})
				}

				if i < maxBuffer {
					s.logger.Debug("All data read", "service", id, "rid", rid)
					s.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_WRITE_DONE})
					break
				}
			}
		}(conn, id, rid)
	}
}

// Shutdown the server and cleanup resources
func (s *Server) Shutdown() {
	for _, v := range s.listeners {
		v.Close()
	}
}
