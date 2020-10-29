package remote

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// get a gRPC client for the given address
func (s *Server) getClient(addr string) (shipyard.RemoteConnectionClient, error) {
	// are we using TLS?
	if s.certPool != nil && s.cert != nil {
		s.log.Debug("Creating TLS connection", "addr", addr)
		creds := credentials.NewTLS(&tls.Config{
			ServerName:   addr,
			Certificates: []tls.Certificate{*s.cert},
			RootCAs:      s.certPool,
		})

		// Create a connection with the TLS credentials
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
		if err != nil {
			return nil, fmt.Errorf("could not dial %s: %s", addr, err)
		}

		return shipyard.NewRemoteConnectionClient(conn), nil
	}

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
		s.log.Info("Connection attempt already in process", "addr", conn.addr)
		return nil
	}

	// mark we are attempting to connect
	conn.setConnecting(true)
	defer func() {
		conn.setConnecting(false)
	}()

	for s.ctx.Err() == nil {
		// if we do not have a connection create one
		if conn.grpcConn == nil || conn.grpcConn.Closed {
			// connect to the service
			s.log.Info("Connecting to server", "addr", conn.addr)
			gc, err := s.openRemoteConnection(conn.addr)
			if err != nil {
				s.log.Error("Unable to open remote connection", "error", err)

				// back off and try again
				time.Sleep(connectionBackoff)
				continue
			}

			// set the connection
			conn.setGRPCConn(gc)

			// send a ping message
			s.log.Debug("Ping connection", "addr", conn.addr)
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

			// set up all the local listeners if the type is remote and the listener does not already
			// exist
			if svc.detail.Type == shipyard.ServiceType_REMOTE && svc.tcpListener == nil {
				// open the listener locally
				l, err := s.createListenerAndListen(id, int(svc.detail.SourcePort))
				if err != nil {
					s.log.Error("Unable to create listener for service", "service_id", id, "error", err)
					return true
				}

				// create the integration such as a kubernetes service
				err = s.createIntegration(id, svc.detail.Name, int(svc.detail.SourcePort))
				if err != nil {
					s.log.Error("Unable to create integration for service", "service_id", id, "error", err)
					return true
				}

				// add the listener to the service
				svc.tcpListener = l
			}

			// send the expose message to the remote so it can open
			s.log.Debug("Sending expose message to remote component", "addr", svc.detail.RemoteConnectorAddr)
			req := &shipyard.OpenData{ServiceId: id}
			req.Message = &shipyard.OpenData_Expose{Expose: &shipyard.ExposeRequest{Service: svc.detail}}

			conn.grpcConn.Send(req)

			return true
		})

		break
	}

	return nil
}

func (s *Server) openRemoteConnection(addr string) (*grpcConn, error) {
	// we need to open a stream to the remote server
	s.log.Debug("Opening Stream to remote server", "addr", addr)

	c, err := s.getClient(addr)
	if err != nil {
		s.log.Error("Unable to get remote client", "addr", addr, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	rc, err := c.OpenStream(context.Background())
	if err != nil {
		s.log.Error("Unable to open remote connection", "addr", addr, "error", err)
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
			select {
			case msg := <-newMessage:
				s.handleRemoteMessage(si, msg)
			case err := <-newError:
				s.log.Error("Error receiving message from remote connection", "addr", si.addr, "error", err)

				// if the connection has not been closed reconnect
				// and if the server is not shutting down
				if !si.grpcConn.Closed && !s.Closed() {
					// We need to tear down any listeners related to this request and clean up resources
					// the downstream should attempt to re-establish the connection and resend the expose requests
					s.teardownConnection(si)

					s.handleReconnection(si)
				}

				return // exit this loop as handleReconnection will recall ths function when a connection is established
			case <-si.grpcConn.Done():
				s.log.Debug("Connection cancelled")
				return
			}

		}
	}(si)
}

func (s *Server) handleRemoteMessage(si *streamInfo, msg *shipyard.OpenData) {
	s.log.Debug("Received client message", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)

	switch m := msg.Message.(type) {
	case *shipyard.OpenData_Data:
		s.log.Trace("Received client message", "service_id", msg.ServiceId, "msg", msg)

		// if we get data send it to the local service instance
		// do we have a local connection, if not create one
		svc, _ := si.services.get(msg.ServiceId)
		c, ok := svc.tcpConnections.Load(msg.ConnectionId)
		if !ok {
			// is this an message reply to a local listener, if there is no connection
			// assume it has gone away so ignore
			if svc.detail.Type == shipyard.ServiceType_REMOTE {
				s.log.Error("Connection does not exist for local listener", "port", svc.detail.SourcePort)
				return
			}

			// otherwise create a new upstream connection
			var err error
			c, err = net.Dial("tcp", svc.detail.DestinationAddr)
			if err != nil {
				s.log.Error("Unable to create connection to remote", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "addr", svc.detail.DestinationAddr)
				return
			}

			// set the connection
			svc.tcpConnections.Store(msg.ConnectionId, c)
		}

		s.log.Debug("Writing data to local", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
		c.(net.Conn).Write(m.Data.Data)

	case *shipyard.OpenData_WriteDone:
		// all writing has been completed for the connection switch to read mode
		s.readData(msg)
	case *shipyard.OpenData_Closed:
		svc, _ := si.services.get(msg.ServiceId)
		c, ok := svc.tcpConnections.Load(msg.ConnectionId)
		if ok {
			s.log.Debug("Closing connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
			// we have a connection close it
			c.(net.Conn).Close()
			svc.tcpConnections.Delete(msg.ConnectionId)
		}
	case *shipyard.OpenData_StatusUpdate:
		s.log.Debug("Received status message", "service_id", msg.ServiceId, "status", m.StatusUpdate.Status)
		svc, _ := si.services.get(msg.ServiceId)
		svc.detail.Status = m.StatusUpdate.Status
	}
}
