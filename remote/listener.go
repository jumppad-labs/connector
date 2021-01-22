package remote

import (
	"fmt"
	"net"

	"github.com/google/uuid"
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

			si, ok := s.streams.findByServiceID(serviceID)
			if !ok {
				// no service exists for this connection, close and return, this should never happen
				s.log.Error(
					"listener",
					"message", "Unable to find bi-directional stream for connection",
					"service_id", serviceID)

				continue
			}

			// set the new connection
			svc, ok := si.services.get(serviceID)
			if !ok {
				s.log.Error(
					"listener",
					"message", "Unable to find service for connection",
					"service_id", serviceID)

				continue
			}

			// generate a unique id for the connection
			connID := uuid.New().String()

			c := newBufferedConn(conn)
			c.id = connID
			svc.tcpConnections.Store(connID, c)

			s.handleConnectionRead(serviceID, si, svc, c)
		}
	}(serviceID, l)
}
