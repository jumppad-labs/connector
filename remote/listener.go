package remote

import (
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/shipyard-run/connector/protos/shipyard"
)

func (s *Server) createListenerAndListen(serviceID string, port int) (net.Listener, error) {
	s.log.Info("Create Listener", "port", port)

	// create the listener
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.log.Error("Unable to open TCP Listener", "error", err)
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
				s.log.Error("Error accepting connection", "service_id", serviceID, "error", err)
				break
			}

			s.log.Debug("Handle new connection", "service_id", serviceID)
			s.handleConnection(serviceID, conn)
		}
	}(serviceID, l)
}

func (s *Server) handleConnection(serviceID string, conn net.Conn) {
	// generate a unique id for the connection
	connID := uuid.New().String()

	s.log.Info("Received new conection on local listener for", "service_id", serviceID, "connection_id", connID)

	str, ok := s.streams.findByServiceID(serviceID)
	if !ok {
		// no service exists for this connection, close and return, this should never happen
		s.log.Error("No stream exists for ", "service_id", serviceID, "connection_id", connID)
		return
	}

	// set the new connection
	svc, ok := str.services.get(serviceID)
	if ok {
		svc.tcpConnections.Store(connID, conn)
	}

	// read the data from the connection
	for {
		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		s.log.Debug("Starting read", "service_id", serviceID, "connection_id", connID)

		// read 4K of data from the connection
		i, err := conn.Read(data)

		// unable to read the data, kill the connection
		if err != nil || i == 0 {
			if err == io.EOF {
				// the connection has closed
				// notify the remote
				str.grpcConn.Send(
					&shipyard.OpenData{
						ServiceId:    serviceID,
						ConnectionId: connID,
						Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
					},
				)
				s.log.Debug("Connection closed", "service_id", serviceID, "connection_id", connID, "error", err)
			} else {
				s.log.Error("Unable to read data from the connection", "service_id", serviceID, "connection_id", connID, "error", err)
			}

			break
		}

		s.log.Trace("Read data for connection", "service_id", serviceID, "connection_id", connID, "len", i, "data", string(data[:i]))

		// send the read chunk of data over the gRPC stream
		// check there is a remote connection if not just return
		s.log.Debug("Sending data to stream", "service_id", serviceID, "connection_id", connID, "data", string(data[:i]))
		str.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    serviceID,
				ConnectionId: connID,
				Message:      &shipyard.OpenData_Data{Data: &shipyard.Data{Data: data[:i]}},
			},
		)

		// we have read all the data send the other end a message so it knows it can now send a response
		if i < maxBuffer {
			s.log.Debug("All data read", "service_id", serviceID, "connID", connID)
			str.grpcConn.Send(&shipyard.OpenData{ServiceId: serviceID, ConnectionId: connID, Message: &shipyard.OpenData_WriteDone{}})
		}
	}
}

// read data from the local and send back to the server
func (s *Server) readData(msg *shipyard.OpenData) {
	str, _ := s.streams.findByServiceID(msg.ServiceId)
	svc, ok := str.services.get(msg.ServiceId)

	con, ok := svc.tcpConnections.Load(msg.ConnectionId)
	if !ok {
		s.log.Error("No connection to read from", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
		return
	}

	for {
		s.log.Debug("Reading data from local server", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)

		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		i, err := con.(net.Conn).Read(data) // read 4k of data

		// if we had a read error tell the server
		if i == 0 || err != nil {
			// The server has closed the connection
			if err == io.EOF {
				// notify the remote
				str.grpcConn.Send(
					&shipyard.OpenData{
						ServiceId:    msg.ServiceId,
						ConnectionId: msg.ConnectionId,
						Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
					},
				)
			} else {
				s.log.Error("Error reading from connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId, "error", err)
			}

			// cleanup
			svc.tcpConnections.Delete(msg.ConnectionId)
			break
		}

		// send the data back to the server
		s.log.Debug("Sending data to remote connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
		str.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    msg.ServiceId,
				ConnectionId: msg.ConnectionId,
				Message:      &shipyard.OpenData_Data{&shipyard.Data{Data: data[:i]}},
			},
		)

		// all read close the connection
		if i < maxBuffer {
			//t := shipyard.MessageType_READ_DONE
			//s.closeConnection(msg, &t, sc)
			break
		}
	}
}
