package remote

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/jumppad-labs/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// get a gRPC client for the given address
func (s *Server) getClient(addr string) (shipyard.RemoteConnectionClient, error) {
	// are we using TLS?
	if s.certPool != nil && s.cert != nil {
		s.log.Debug(
			"server",
			"message", "Creating TLS client",
			"addr", addr)

		creds := credentials.NewTLS(&tls.Config{
			ServerName:   addr,
			Certificates: []tls.Certificate{*s.cert},
			RootCAs:      s.certPool,
		})

		// Create a connection with the TLS credentials
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
		if err != nil {
			return nil, fmt.Errorf(
				"Unable to dial %s: %s", addr, err)
		}

		return shipyard.NewRemoteConnectionClient(conn), nil
	}

	s.log.Debug(
		"server",
		"message", "Creating Insecure client",
		"addr", addr)

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	return shipyard.NewRemoteConnectionClient(conn), nil
}

func (s *Server) handleReconnection(conn *streamInfo) error {
	// if is possible that this method gets called multiple times
	// ensure there is only one operation in process at once
	if conn.isConnecting() {
		s.log.Info(
			"local_server",
			"message", "Connection attempt already in process",
			"addr", conn.addr)

		return nil
	}

	// mark we are attempting to connect
	conn.setConnecting(true)
	defer func() {
		conn.setConnecting(false)
	}()

	for s.ctx.Err() == nil {
		closed := true
		if conn.grpcConn != nil && !conn.grpcConn.Closed {
			closed = false
		}

		s.log.Trace(
			"local_server",
			"message", "Reconnecting, current connection state",
			"grpcConn_nil", conn.grpcConn == nil,
			"grpcConn_closed", closed,
		)

		// if we do not have a connection create one
		if closed {
			// connect to the service
			s.log.Info(
				"local_server",
				"message", "Connecting to remote server",
				"addr", conn.addr)

			gc, err := s.openRemoteConnection(conn.addr)
			if err != nil {
				s.log.Error(
					"local_server",
					"message",
					"Unable to open remote connection", "error", err)

				// back off and try again
				time.Sleep(connectionBackoff)
				continue
			}

			// set the connection
			conn.setGRPCConn(gc)

			// send a ping message
			s.log.Debug(
				"local_server",
				"message", "Remote connetion estabilished, ping connection",
				"addr", conn.addr)

			conn.grpcConn.Send(&shipyard.OpenData{Message: &shipyard.OpenData_Ping{Ping: &shipyard.NullMessage{}}})

			// handle messages for this stream
			s.handleRemoteConnection(conn)
		}

		// loop all services and try to reconfigure
		conn.services.iterate(func(id string, svc *service) bool {
			// do not attempt when status is Error
			if svc.detail.Status == shipyard.ServiceStatus_ERROR {
				return true
			}

			// register the service with the integration, this returns the location details
			// for the local service
			ssd, err := s.integration.Register(id, svc.detail.Type.String(), svc.detail.Config)
			if err != nil {
				s.log.Error(
					"local_server",
					"message", "Unable to create integration for service",
					"service_id", id, "error", err)

				return true
			}

			// set up all the local listeners if the type is remote and the listener does not already
			// exist
			if svc.detail.Type == shipyard.ServiceType_REMOTE && svc.tcpListener == nil {

				// open the listener locally
				l, err := s.createListenerAndListen(id, ssd.Port)
				if err != nil {
					s.log.Error(
						"local_server",
						"message", "Unable to create listener for service",
						"service_id", id, "error", err)

					return true
				}

				// add the listener to the service
				svc.tcpListener = l
			}

			// send the expose message to the remote so it can open
			s.log.Debug(
				"local_server",
				"message", "Sending expose message to remote side",
				"addr", svc.detail.RemoteConnectorAddr)

			req := &shipyard.OpenData{ServiceId: id}
			req.Message = &shipyard.OpenData_Expose{Expose: &shipyard.ExposeRequest{Service: svc.detail}}

			conn.grpcConn.Send(req)

			return true
		})

		return nil
	}

	s.log.Debug(
		"local_server",
		"message", "Context cancelled while waiting to reconnect",
		"err", s.ctx.Err(),
	)

	return nil
}

func (s *Server) openRemoteConnection(addr string) (*grpcConn, error) {
	// we need to open a stream to the remote server
	s.log.Debug(
		"local_server",
		"message", "Opening grpc bi-directional stream to remote server",
		"addr", addr)

	c, err := s.getClient(addr)
	if err != nil {
		s.log.Error(
			"local_server",
			"message", "Unable to create client",
			"addr", addr, "error", err)

		return nil, status.Error(codes.Internal, err.Error())
	}

	rc, err := c.OpenStream(context.Background())
	if err != nil {
		s.log.Error(
			"local_server",
			"message", "Unable to establish remote connection",
			"addr", addr, "error", err)

		return nil, fmt.Errorf("Unable to open remote connection to server %s: %s", addr, err)
	}

	return newGRPCConn(rc), nil
}

func (s *Server) handleRemoteConnection(si *streamInfo) {
	// wrap in a go func to immediately return
	go func(si *streamInfo) {
		newMessage := make(chan *shipyard.OpenData)
		newError := make(chan error)

		go func(si *streamInfo) {
			for {
				msg, err := si.grpcConn.Recv()
				if err != nil {
					newError <- err
					return
				} else if msg != nil {
					newMessage <- msg
				} else {
					return
				}
			}
		}(si)

		for {
			s.log.Trace(
				"local_server",
				"message", "Waiting for remote client message")

			select {
			case msg := <-newMessage:
				s.handleRemoteMessage(si, msg)
			case err := <-newError:
				s.log.Error(
					"local_server",
					"message", "Error receiving message from remote connection",
					"addr", si.addr, "error", err)

				// if the connection has not been closed reconnect
				// and if the server is not shutting down
				if !si.grpcConn.Closed && !s.Closed() {
					s.log.Debug(
						"local_server",
						"message", "Connection closed, attempt reconection",
						"addr", si.addr)

					// mark the internal structure as closed
					si.grpcConn.Closed = true

					// We need to tear down any listeners related to this request and clean up resources
					// the downstream should attempt to re-establish the connection and resend the expose requests
					s.teardownConnection(si)
					s.handleReconnection(si)
				}

				return // exit this loop as handleReconnection will recall ths function when a connection is established
			case <-si.grpcConn.Done():
				s.log.Debug(
					"local_server",
					"message", "Connection context cancelled",
					"addr", si.addr)
				return
			}

		}
	}(si)
}

func (s *Server) handleRemoteMessage(si *streamInfo, msg *shipyard.OpenData) {
	s.log.Debug(
		"local_server",
		"message", "Received message",
		"serviceID", msg.ServiceId,
		"connectionID", msg.ConnectionId)

	switch m := msg.Message.(type) {
	case *shipyard.OpenData_Data:
		s.handleRemoteDataMessage(si, msg, m)
	case *shipyard.OpenData_Closed:
		s.handleRemoteClosedMessage(si, msg, m)
	case *shipyard.OpenData_StatusUpdate:
		s.handleRemoteUpdateMessage(si, msg, m)
	}
}

func (s *Server) handleRemoteDataMessage(si *streamInfo, msg *shipyard.OpenData, m *shipyard.OpenData_Data) {
	s.log.Trace(
		"local_server",
		"message", "Received data from remote",
		"message_id", m.Data.Id,
		"service_id", msg.ServiceId,
		"msg", msg)

	// if we get data find the connection for the message
	// if we do not have a connection create one
	svc, _ := si.services.get(msg.ServiceId)
	c, ok := svc.getTCPConnection(msg.ConnectionId)
	if !ok {
		// is this a message for an upstream and if there is no connection
		// assume the upstream has disconnected and ignore the message
		if svc.detail.Type == shipyard.ServiceType_REMOTE {
			s.log.Error(
				"local_server",
				"message", "No connection for data, ignore message",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId)

			return
		}

		s.log.Trace(
			"local_server",
			"message", "Create new upstream connection for data",
			"service_id", msg.ServiceId,
			"connection_id", msg.ConnectionId)

		// otherwise create a new upstream connection
		var err error

		addr, err := s.integration.LookupAddress(svc.detail.Id)
		if err != nil {
			s.log.Error(
				"local_server",
				"message", "Unable to find address for upstream",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId,
				"error", err,
			)

			si.grpcConn.Send(
				&shipyard.OpenData{
					ServiceId:    msg.ServiceId,
					ConnectionId: msg.ConnectionId,
					Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
				},
			)
			return
		}

		newCon, err := net.Dial("tcp", addr)
		if err != nil {
			s.log.Error(
				"local_server",
				"message", "Unable to create connection to upstream",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId,
				"addr", addr)

			si.grpcConn.Send(
				&shipyard.OpenData{
					ServiceId:    msg.ServiceId,
					ConnectionId: msg.ConnectionId,
					Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
				},
			)
			return
		}

		// set the connection
		c = newBufferedConn(newCon)
		c.id = msg.ConnectionId
		svc.setTCPConnection(msg.ConnectionId, c)

		// start read handler and don't block
		go s.handleConnectionRead(msg.ServiceId, si, svc, c)
	}

	s.log.Trace(
		"local_server",
		"message", "Writing data to local connection",
		"service_id", msg.ServiceId,
		"connection_id", msg.ConnectionId)

	i, err := c.Write(m.Data.Data)
	if err != nil {
		if err == io.EOF {
			s.log.Debug(
				"local_server",
				"message", "Connection closed",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId)
		} else {
			s.log.Error(
				"local_server",
				"message", "Error writing to connection",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId,
				"error", err,
			)
		}

		// send closed message
		si.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    msg.ServiceId,
				ConnectionId: msg.ConnectionId,
				Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
			},
		)
	}

	s.log.Trace(
		"local_server",
		"message", "Data written to local connection",
		"service_id", msg.ServiceId,
		"connection_id", msg.ConnectionId)

	// if the size of the data is less than the max buffer
	// all writing has been completed for the connection switch to read mode
	if i < MessageSize {
		s.log.Trace(
			"local_server",
			"message", "All data written to local connection, start read",
			"data_written", i,
			"message_size", MessageSize,
			"service_id", msg.ServiceId,
			"connection_id", msg.ConnectionId)

	}
}

func (s *Server) handleRemoteClosedMessage(si *streamInfo, msg *shipyard.OpenData, m *shipyard.OpenData_Closed) {
	s.log.Trace(
		"local_server",
		"message", "Received close connection message",
		"service_id", msg.ServiceId,
		"connection_id", msg.ConnectionId)

	svc, _ := si.services.get(msg.ServiceId)
	if svc == nil {
		s.log.Error(
			"local_server",
			"message", "Service does not exist",
			"service_id", msg.ServiceId,
			"connection_id", msg.ConnectionId)

		return
	}

	c, ok := svc.getTCPConnection(msg.ConnectionId)
	if ok {
		s.log.Trace(
			"local_server",
			"message", "Closing connection",
			"service_id", msg.ServiceId,
			"connection_id", msg.ConnectionId)

		// we have a connection close it
		c.Close()
		svc.removeTCPConnection(msg.ConnectionId)
	}
}

func (s *Server) handleRemoteUpdateMessage(si *streamInfo, msg *shipyard.OpenData, m *shipyard.OpenData_StatusUpdate) {
	s.log.Trace(
		"local_server",
		"message", "Received status message",
		"service_id", msg.ServiceId,
		"status", m.StatusUpdate.Status,
		"config", m.StatusUpdate.Config,
	)

	svc, _ := si.services.get(msg.ServiceId)
	svc.detail.Status = m.StatusUpdate.Status
	svc.detail.Details = m.StatusUpdate.Config
}
