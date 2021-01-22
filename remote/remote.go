package remote

import (
	"io"
	"net"

	"github.com/shipyard-run/connector/protos/shipyard"
)

func (s *Server) newRemoteStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	s.log.Info(
		"server",
		"message", "Received new grpc bi-directional stream from client")

	gc := newGRPCConn(svr)
	si := newStreamInfo()
	si.addr = "localhost" // this is an inbound connection
	si.grpcConn = gc

	s.streams.add(si)

	for {
		s.log.Trace(
			"remote_server",
			"message", "Waiting for remote client message")

		msg, err := svr.Recv()

		if err != nil {
			s.log.Error(
				"remote_server",
				"message", "Error receiving message from remote connection, assume connection problem. Tearing down connection",
				"addr", si.addr, "error", err)

			// We need to tear down any listeners related to this request and clean up resources
			// the downstream should attempt to re-establish the connection and resend the expose requests
			s.teardownConnection(si)
			return nil
		}

		s.log.Debug(
			"remote_server",
			"message", "Received message",
			"serviceID", msg.ServiceId,
			"connectionID", msg.ConnectionId)

		switch m := msg.Message.(type) {
		case *shipyard.OpenData_Expose:
			// Does this already exist? If so it will be a repeat send so ignore
			_, ok := si.services.get(msg.ServiceId)
			if ok {
				s.log.Trace(
					"remote_server",
					"message", "Service already exists, ignoring message",
					"service_id", msg.ServiceId,
					"connection_id", msg.ConnectionId,
					"port", m.Expose.Service.SourcePort)

				continue
			}

			s.log.Info(
				"remote_server",
				"message", "Expose new service",
				"service_id", msg.ServiceId,
				"type", m.Expose.Service.Type)

			svc := newService()

			// The connection is exposing a local service to us
			// we need to open a TCP Listener for the service
			if m.Expose.Service.Type == shipyard.ServiceType_LOCAL {
				s.log.Trace(
					"remote_server",
					"message", "Create new listener for inbound data",
					"service_id", msg.ServiceId,
					"connection_id", msg.ConnectionId,
					"port", m.Expose.Service.SourcePort)

				var listener net.Listener
				var err error
				listener, err = s.createListenerAndListen(msg.ServiceId, int(m.Expose.Service.SourcePort))
				if err != nil {
					s.log.Error(
						"remote_server",
						"message", "Error creating listener, send notification to remote",
						"service_id", msg.ServiceId,
						"type", m.Expose.Service.Type,
						"error", err)

					// we need to send an error back to the connection
					svr.Send(&shipyard.OpenData{
						ServiceId: msg.ServiceId,
						Message: &shipyard.OpenData_StatusUpdate{
							StatusUpdate: &shipyard.StatusUpdate{
								Status:  shipyard.ServiceStatus_ERROR,
								Message: err.Error(),
							},
						},
					})

					continue
				}

				// create the integration such as a kubernetes service
				err = s.createIntegration(msg.ServiceId, m.Expose.Service.Name, int(m.Expose.Service.SourcePort))
				if err != nil {
					s.log.Error(
						"remote_local",
						"message", "Unable to create integration for service",
						"service_id", msg.ServiceId, "error", err)

					// we need to send an error back to the connection
					svr.Send(&shipyard.OpenData{
						ServiceId: msg.ServiceId,
						Message: &shipyard.OpenData_StatusUpdate{
							StatusUpdate: &shipyard.StatusUpdate{
								Status:  shipyard.ServiceStatus_ERROR,
								Message: err.Error(),
							},
						},
					})

					continue
				}

				svc.tcpListener = listener
			}

			svc.detail = m.Expose.Service
			svc.detail.Status = shipyard.ServiceStatus_COMPLETE
			si.services.add(msg.ServiceId, svc)

			s.log.Trace(
				"remote_server",
				"message", "Exposing service complete, notify remote",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId,
				"port", m.Expose.Service.SourcePort)

			svr.Send(&shipyard.OpenData{
				ServiceId: msg.ServiceId,
				Message: &shipyard.OpenData_StatusUpdate{
					StatusUpdate: &shipyard.StatusUpdate{
						Status: shipyard.ServiceStatus_COMPLETE,
					},
				},
			})

		case *shipyard.OpenData_Destroy:
			s.log.Trace(
				"remote_server",
				"message", "Received destroy service message",
				"service_id", msg.ServiceId,
				"msg", msg)

			// Does this already exist? If so it will be a repeat send so ignore
			svc, ok := si.services.get(msg.ServiceId)
			if !ok {
				continue
			}

			s.teardownService(svc)
			si.services.delete(msg.ServiceId)

		case *shipyard.OpenData_Data:
			s.log.Trace(
				"remote_server",
				"message", "Received data message",
				"message_id", m.Data.Id,
				"service_id", msg.ServiceId,
				"msg", msg)

			// get the service for this data
			svc, ok := si.services.get(msg.ServiceId)
			if !ok {
				// if there is no service for this message ignore it
				s.log.Error(
					"remote_server",
					"message", "Service does not exist for message, ignoring",
					"service_id", msg.ServiceId)

				continue
			}

			// get the connection
			c, ok := svc.getTCPConnection(msg.ConnectionId)

			// no connection exists, if this is a remote service try to establish a new connection to the upstream service
			// otherwise ignore as the connection should have been created by the local listener
			if !ok {
				if svc.detail.Type == shipyard.ServiceType_LOCAL {
					s.log.Error(
						"remote_server",
						"message", "No connection for data, ignore message",
						"port", svc.detail.SourcePort,
						"serviceID", msg.ServiceId,
						"connectionID", msg.ConnectionId)

					continue
				}

				// open a new connection
				s.log.Trace(
					"remote_server",
					"message", "Create new upstream connection for data",
					"service_id", msg.ServiceId,
					"connection_id", msg.ConnectionId,
					"addr", svc.detail.DestinationAddr)

				newConn, err := net.Dial("tcp", svc.detail.DestinationAddr)
				if err != nil {
					s.log.Error(
						"remote_server",
						"message", "Unable to create connection to upstream",
						"service_id", msg.ServiceId,
						"connection_id", msg.ConnectionId,
						"addr", svc.detail.DestinationAddr)

					svr.Send(
						&shipyard.OpenData{
							ServiceId:    msg.ServiceId,
							ConnectionId: msg.ConnectionId,
							Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
						},
					)
					continue
				}

				c = newBufferedConn(newConn)
				c.id = msg.ConnectionId
				svc.setTCPConnection(msg.ConnectionId, c)

				// start read handler and don't block
				go s.handleConnectionRead(msg.ServiceId, si, svc, c)
			}

			s.log.Trace(
				"remote_server",
				"message", "Writing data to local connection",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId)

			i, err := c.Write(m.Data.Data)
			if err != nil {
				if err == io.EOF {
					s.log.Debug(
						"remote_server",
						"message", "Connection closed",
						"service_id", msg.ServiceId,
						"connection_id", msg.ConnectionId)
				} else {
					s.log.Error(
						"remote_server",
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
				"remote_server",
				"message", "Data written to local connection",
				"service_id", msg.ServiceId,
				"connection_id", msg.ConnectionId)

			// if the size of the data is less than the max buffer
			// all writing has been completed for the connection switch to read mode
			if i < MessageSize {
				s.log.Trace(
					"remote_server",
					"message", "All data written to local connection, start read",
					"data_written", i,
					"message_size", MessageSize,
					"service_id", msg.ServiceId,
					"connection_id", msg.ConnectionId)
			}

		case *shipyard.OpenData_Closed:
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

				continue
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
	}
}
