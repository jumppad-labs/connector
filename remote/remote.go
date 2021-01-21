package remote

import (
	"net"
	"reflect"

	"github.com/shipyard-run/connector/protos/shipyard"
)

func (s *Server) newRemoteStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	s.log.Info("Received new stream connection from client")

	gc := newGRPCConn(svr)
	streamInfo := newStreamInfo()
	streamInfo.addr = "localhost" // this is an inbound connection
	streamInfo.grpcConn = gc

	s.streams.add(streamInfo)

	for {
		s.log.Trace("Waiting for remote client message")

		msg, err := svr.Recv()
		s.log.Trace("Remote Stream server message", "msg", msg, "error", err)

		if err != nil {
			s.log.Error("Error receiving server message", "error", err)

			// We need to tear down any listeners related to this request and clean up resources
			// the downstream should attempt to re-establish the connection and resend the expose requests
			s.teardownConnection(streamInfo)
			return nil
		}

		s.log.Debug("Received server message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "type", reflect.TypeOf(msg.Message))

		switch m := msg.Message.(type) {
		case *shipyard.OpenData_Expose:
			// Does this already exist? If so it will be a repeat send so ignore
			_, ok := streamInfo.services.get(msg.ServiceId)
			if ok {
				continue
			}

			s.log.Info("Expose new service", "service_id", msg.ServiceId, "type", m.Expose.Service.Type)

			// The connection is exposing a local service to us
			// we need to open a TCP Listener for the service
			var listener net.Listener
			if m.Expose.Service.Type == shipyard.ServiceType_LOCAL {
				var err error
				listener, err = s.createListenerAndListen(msg.ServiceId, int(m.Expose.Service.SourcePort))
				if err != nil {
					s.log.Error("Error creating listener", "service_id", msg.ServiceId, "type", m.Expose.Service.Type, "error", err)
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
					s.log.Error("Error creating integration", "service_id", msg.ServiceId, "type", m.Expose.Service.Type, "error", err)
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
			}

			// set the listener on our collection
			svc := newService()
			streamInfo.services.add(msg.ServiceId, svc)

			svc.tcpListener = listener
			svc.detail = m.Expose.Service
			svc.detail.Status = shipyard.ServiceStatus_COMPLETE

			svr.Send(&shipyard.OpenData{
				ServiceId: msg.ServiceId,
				Message: &shipyard.OpenData_StatusUpdate{
					StatusUpdate: &shipyard.StatusUpdate{
						Status: shipyard.ServiceStatus_COMPLETE,
					},
				},
			})

		case *shipyard.OpenData_Destroy:
			s.log.Info("Destroy service", "service_id", msg.ServiceId)

			// Does this already exist? If so it will be a repeat send so ignore
			si, ok := streamInfo.services.get(msg.ServiceId)
			if !ok {
				continue
			}

			s.teardownService(si)

			streamInfo.services.delete(msg.ServiceId)

		case *shipyard.OpenData_Data:
			s.log.Trace("Message detail", "msg", m.Data.Data)

			// get the service for this data
			svc, ok := streamInfo.services.get(msg.ServiceId)

			// if there is no service for this message ignore it
			if !ok {
				s.log.Error("Service does not exist for message", "service_id", msg.ServiceId)
				continue
			}

			// get the connection
			tcpConn, ok := svc.tcpConnections.Load(msg.ConnectionId)

			// no connection exists, if this is a remote service try to establish a new connection to the upstream service
			// otherwise ignore as the connection should have been created by the local listener
			if !ok {
				if svc.detail.Type == shipyard.ServiceType_LOCAL {
					s.log.Error("Local connection does not exist", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
					continue
				}

				// open a new connection
				s.log.Debug("Local connection does not exist, creating", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "addr", svc.detail.DestinationAddr)
				tcpConn, err = net.Dial("tcp", svc.detail.DestinationAddr)
				if err != nil {
					s.log.Error("Unable to create local connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "error", err)
					svr.Send(
						&shipyard.OpenData{
							ServiceId:    msg.ServiceId,
							ConnectionId: msg.ConnectionId,
							Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
						},
					)
					continue
				}

				svc.tcpConnections.Store(msg.ConnectionId, tcpConn)
			}

			s.log.Debug("Writing data to connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
			tcpConn.(net.Conn).Write(m.Data.Data)

		case *shipyard.OpenData_WriteDone:
			// all data has been received, read from the local connection
			s.readData(msg)

		case *shipyard.OpenData_Closed:
			// remote end of the connection has been closed, close this end
			svc, ok := streamInfo.services.get(msg.ServiceId)

			// if there is no service for this message ignore it
			if !ok {
				s.log.Error("Service does not exist for message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
				continue
			}

			conn, ok := svc.tcpConnections.Load(msg.ConnectionId)

			// no connection exists
			if !ok {
				s.log.Error("Connection does not exist for message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
				continue
			}

			s.log.Debug("Closing connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
			conn.(net.Conn).Close()

			svc.tcpConnections.Delete(msg.ConnectionId)
		}
	}
}
