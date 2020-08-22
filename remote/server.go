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
	connections map[string]shipyard.RemoteConnection_OpenLocalServer
	logger      hclog.Logger
}

func NewServer(log hclog.Logger) *Server {
	return &Server{
		make(map[string]net.Listener),
		make(map[string]net.Conn),
		make(map[string]shipyard.RemoteConnection_OpenLocalServer),
		log,
	}
}

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

func (s *Server) OpenRemote(rc shipyard.RemoteConnection_OpenRemoteServer) error {
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

		switch msg.Type {
		case shipyard.MessageType_NEW_CONNECTION:
			s.newConnection(msg, msg.Location)
		case shipyard.MessageType_DATA:
			s.writeData(msg, rc) // write data from the remote connection to the local endpoint
		case shipyard.MessageType_WRITE_DONE:
			s.readData(msg, rc) // read the response and send back to the server
		case shipyard.MessageType_ERROR:
			s.closeConnection(msg, nil, rc) // read the response and send back to the server
		}
	}

	return nil
}

func (s *Server) OpenLocal(rc shipyard.RemoteConnection_OpenLocalServer) error {
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

func (s *Server) newConnection(msg *shipyard.OpenData, localAddr string) error {
	var err error
	con, err := net.Dial("tcp", localAddr)
	if err != nil {
		s.logger.Error("Unable to open connection to remote server", "addr", localAddr, "error", err)
		return err
	}

	s.tcpConn[msg.RequestId] = con
	return nil
}

func (s *Server) closeConnection(msg *shipyard.OpenData, t *shipyard.MessageType, sc shipyard.RemoteConnection_OpenLocalServer) {
	if conn, ok := s.tcpConn[msg.RequestId]; ok {
		conn.Close()
		delete(s.tcpConn, msg.RequestId)

		if t != nil {
			// send the close message back to the server
			sc.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, RequestId: msg.RequestId, Type: *t})
		}
	}
}

func (s *Server) readData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenLocalServer) {
	con, ok := s.tcpConn[msg.RequestId]
	if ok {
		for {
			s.logger.Debug("Reading data from local server", "rid", msg.RequestId)

			maxBuffer := 4096
			data := make([]byte, maxBuffer)

			i, err := con.Read(data) // read 4k of data

			// if we had a read error tell the server
			if i == 0 || err != nil {
				t := shipyard.MessageType_ERROR
				s.closeConnection(msg, &t, sc)
				break
			}

			// send the data back to the server
			s.logger.Debug("Sending data to remote connection", "rid", msg.RequestId)
			sc.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, RequestId: msg.RequestId, Type: shipyard.MessageType_DATA, Data: data[:i]})

			// all read close the connection
			if i < maxBuffer {
				t := shipyard.MessageType_READ_DONE
				s.closeConnection(msg, &t, sc)
				break
			}
		}
	}
}

func (s *Server) writeData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenLocalServer) {
	con, ok := s.tcpConn[msg.RequestId]
	if ok {
		_, err := con.Write(msg.Data)
		if err != nil {
			s.logger.Error("Unable to write data to connection", "error", err)
		}
	}
}

// Shutdown the server and cleanup resources
func (s *Server) Shutdown() {
	for _, v := range s.listeners {
		v.Close()
	}
}
