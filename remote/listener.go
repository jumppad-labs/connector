package remote

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/shipyard-run/connector/protos/shipyard"
)

func (s *Server) createListenerAndListen(serviceID string, port int) (net.Listener, error) {
	s.log.Info("listener", "message", "Create Listener", "port", port)

	// create the listener
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.log.Error("listener", "message", "Unable to create TCP listener", "error", err)
		return nil, err
	}

	s.handleListener(serviceID, l)
	return l, nil
}

func (s *Server) handleListener(serviceID string, l net.Listener) {
	// wrap in a go func to immediately return
	go func(serviceID string, l net.Listener) {
		for {
			conn, err := l.Accept()
			if err != nil {
				s.log.Error("listener", "message", "Unable to accept connection", "service_id", serviceID, "error", err)
				break
			}

			s.log.Debug("listener", "message", "Handle new connection", "service_id", serviceID)
			s.handleConnectionRead(serviceID, newBufferedConn(conn))
		}
	}(serviceID, l)
}

func (s *Server) handleConnectionRead(serviceID string, conn *bufferedConn) {
	s.log.Info("listener", "message", "Received new conection for", "service_id", serviceID)

	str, ok := s.streams.findByServiceID(serviceID)
	if !ok {
		// no service exists for this connection, close and return, this should never happen
		s.log.Error(
			"listener",
			"message", "Unable to find bi-directional stream for connection",
			"service_id", serviceID)

		return
	}

	// set the new connection
	svc, ok := str.services.get(serviceID)
	if ok {
		s.log.Error(
			"listener",
			"message", "Unable to find service for connection",
			"service_id", serviceID)

		return
	}

	// generate a unique id for the connection
	connID := uuid.New().String()
	svc.tcpConnections.Store(connID, conn)

	// read the data from the connection
	for {
		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		s.log.Debug("listener", "message", "Reading data from connection", "service_id", serviceID, "connection_id", connID)

		// read 4K of data from the connection
		i, err := conn.Read(data)

		// unable to read the data, kill the connection
		if err != nil || i == 0 {
			if err == io.EOF {
				s.log.Debug(
					"listener",
					"message", "Connection closed",
					"service_id", serviceID,
					"connection_id", connID,
					"error", err)

			} else {
				s.log.Error(
					"listener",
					"message", "Unable to read data from the connection",
					"service_id", serviceID,
					"connection_id", connID,
					"error", err)
			}

			// the connection has closed
			// notify the remote
			str.grpcConn.Send(
				&shipyard.OpenData{
					ServiceId:    serviceID,
					ConnectionId: connID,
					Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
				},
			)

			// exit the for loop
			return
		}

		s.log.Trace(
			"listener",
			"message", "Read data from connection",
			"service_id", serviceID,
			"connection_id", connID,
			"len", i,
			"data", string(data[:i]))

		// send the read chunk of data over the gRPC stream
		// check there is a remote connection if not just return
		s.log.Debug(
			"listener",
			"message", "Sending data to remote server",
			"addr", str.addr,
			"service_id", serviceID,
			"connection_id", connID)

		str.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    serviceID,
				ConnectionId: connID,
				Message:      &shipyard.OpenData_Data{Data: &shipyard.Data{Data: data[:i]}},
			},
		)

		// we have read all the data send the other end a message so it knows it can now send a response
		if i < maxBuffer {
			s.log.Debug(
				"listener",
				"message", "All data read from connection",
				"i", i,
				"service_id", serviceID,
				"connID", connID)

			// check if remote has closed the connection
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			if _, err := conn.Peek(1); err == io.EOF {
				s.log.Debug(
					"listener",
					"message", "Detected connection closed",
					"service_id", serviceID,
					"connection_id", connID,
					"error", err)

				conn.Close()
				svc.removeTCPConnection(connID)

				str.grpcConn.Send(
					&shipyard.OpenData{
						ServiceId:    serviceID,
						ConnectionId: connID,
						Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
					},
				)

			} else {
				s.log.Debug(
					"listener",
					"message", "All data read but connection still open",
					"service_id", serviceID,
					"connection_id", connID,
					"error", err)

				conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			}

			return
		}
	}
}
