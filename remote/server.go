package remote

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/integrations"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const MessageSize = 4096 // 4k data payload

type Server struct {
	log hclog.Logger
	// Collection which listeners and tcp connections for a Server stream
	streams streams

	certPool *x509.CertPool
	cert     *tls.Certificate

	ctx context.Context
	cf  context.CancelFunc

	integration integrations.Integration
}

// New creates a new gRPC remote connector server
func New(l hclog.Logger, certPool *x509.CertPool, cert *tls.Certificate, integr integrations.Integration) *Server {
	if certPool != nil && cert != nil {
		l.Info("Creating new Server with mTLS")
	} else {
		l.Info("Creating new Server")
	}

	ctx, cf := context.WithCancel(context.Background())

	return &Server{
		l,
		streams{},
		certPool,
		cert,
		ctx, cf,
		integr,
	}
}

// OpenStream is a called by a remote server to open a bidirectional stream between two
// Connectors
func (s *Server) OpenStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	return s.newRemoteStream(svr)
}

// ExposeService is the public gRPC API method for creating a service connection
func (s *Server) ExposeService(ctx context.Context, r *shipyard.ExposeRequest) (*shipyard.ExposeResponse, error) {
	// generate a unique id for the service
	id := uuid.New().String()
	s.log.Info("Expose Service", "req", r, "service_id", id)

	svc := newService()
	svc.detail = r.Service
	svc.detail.Status = shipyard.ServiceStatus_PENDING

	// validate that there is not already a service
	for _, s := range s.streams {
		if s.services.contains(svc) {
			return nil, status.Errorf(codes.InvalidArgument, "Unable to expose remote service: %s, already exists", s.addr)
		}
	}

	// find a remote connection
	si, ok := s.streams.findByRemoteAddr(r.Service.RemoteConnectorAddr)
	if !ok {
		si = newStreamInfo()
		si.addr = r.Service.RemoteConnectorAddr

		// add the new stream to the collection
		s.streams.add(si)
	}

	// add the service to the connection
	svc.detail.Id = id
	si.services.add(id, svc)

	// establish a connection to the remote endpoint and setup listeners
	go s.handleReconnection(si)

	return &shipyard.ExposeResponse{Id: id}, nil
}

// DestroyService is the public gRPC API method to remove a service
func (s *Server) DestroyService(ctx context.Context, dr *shipyard.DestroyRequest) (*shipyard.NullMessage, error) {
	s.log.Info("Destroy service", "id", dr.Id)

	// find the remoteConnection for the service
	si, ok := s.streams.findByServiceID(dr.Id)
	if !ok {
		s.log.Error("Connection does not exist", "id", dr.Id)
		return nil, status.Errorf(codes.NotFound, "Service with ID: %s, does not exist", dr.Id)
	}

	svc, _ := si.services.get(dr.Id)
	s.teardownService(svc)

	// send a message to the remote end that the service has been removed
	if si.grpcConn != nil {
		si.grpcConn.Send(&shipyard.OpenData{ServiceId: dr.Id, Message: &shipyard.OpenData_Destroy{Destroy: &shipyard.DestroyRequest{Id: dr.Id}}})
	}

	// delete the service
	si.services.delete(dr.Id)

	return &shipyard.NullMessage{}, nil
}

// ListServices returns a list of active services along with their state
func (s *Server) ListServices(ctx context.Context, m *shipyard.NullMessage) (*shipyard.ListResponse, error) {
	s.log.Info("Listing services")

	services := []*shipyard.Service{}

	for _, stream := range s.streams {
		stream.services.iterate(func(id string, svc *service) bool {
			services = append(services, svc.detail)

			// return true to continue iterating
			return true
		})
	}

	return &shipyard.ListResponse{Services: services}, nil
}

// Closed returns true or false if server shutdown has been called
func (s *Server) Closed() bool {
	return s.ctx.Err() != nil
}

// Shutdown the server, closing all connections and listeners
func (s *Server) Shutdown() {
	s.log.Info("Shutting down")
	s.cf() // cancel the running context

	//defer func() {
	//	if r := recover(); r != nil {
	//		s.log.Error("Error when shutting down service", "error", r)
	//	}
	//}()

	// close all listeners
	s.log.Info("Closing all TCPListeners and Connections")
	for _, t := range s.streams {
		if t != nil {
			s.teardownConnection(t)
			t.closeGRPCConn()
		}
	}
}

var connectionBackoff = 10 * time.Second

func (s *Server) teardownConnection(si *streamInfo) {
	si.services.iterate(func(id string, svc *service) bool {
		// close any open connections
		s.teardownService(svc)
		svc.detail.Status = shipyard.ServiceStatus_PENDING

		return true
	})
}

var teardownSync = sync.Mutex{}

func (s *Server) teardownService(svc *service) {
	teardownSync.Lock()
	defer teardownSync.Unlock()

	// close any open TCP connections
	svc.tcpConnections.Range(func(k interface{}, v interface{}) bool {
		conn := v.(net.Conn)
		conn.Close()

		svc.removeTCPConnection(k.(string))

		return true
	})

	// close the listener
	if svc.tcpListener != nil {
		s.log.Debug("Closing TCP Listener", "addr", svc.tcpListener.Addr())
		svc.tcpListener.Close()
		svc.tcpListener = nil
	}

	// are there any integrations to remove
	err := s.integration.Deregister(svc.detail.Id)
	if err != nil {
		s.log.Error("Unable to create integration for service", "service_id", svc.detail.Id, "error", err)
	}
}
