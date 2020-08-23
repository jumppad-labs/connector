package remote

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type remoteConnection struct {
	id   string
	conn interface{}
}

func (r *remoteConnection) Send(data *shipyard.OpenData) {
	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		c.Send(data)
	case shipyard.RemoteConnection_OpenStreamServer:
		c.Send(data)
	}
}

type service struct {
	id               string
	name             string
	remoteServerAddr string
	localServiceAddr string
	localPort        int
	remotePort       int
	listener         net.Listener
	remoteConnection *remoteConnection
	localConnections map[string]net.Conn
}

func newService(id string) *service {
	return &service{
		id:               id,
		localConnections: map[string]net.Conn{},
	}
}

type Server struct {
	log      hclog.Logger
	services map[string]*service
}

func New(l hclog.Logger) *Server {
	return &Server{
		l,
		map[string]*service{},
	}
}

func (s *Server) findService(id string) *service {
	svc, ok := s.services[id]
	if !ok {
		svc = newService(id)
		s.services[id] = svc
	}

	return svc
}

func (s *Server) OpenStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	s.log.Info("Received new connection from client")

	for {
		msg, err := svr.Recv()
		if err != nil {
			s.log.Error("Error receiving server message", "error", err)
			time.Sleep(1 * time.Second) // backoff for now, we need to handle failure better
			break
		}

		s.log.Debug("Received server message", "type", msg.Type, "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
		switch msg.Type {
		case shipyard.MessageType_HELLO:
			// new connection for service add to the collection
			svc := s.findService(msg.ServiceId)
			svc.remoteConnection = &remoteConnection{id: msg.ServiceId, conn: svr}
			svc.localServiceAddr = msg.Location
		case shipyard.MessageType_DATA:
			s.log.Trace("Message detail", "msg", msg)

			svc := s.findService(msg.ServiceId)
			conn, ok := svc.localConnections[msg.ConnectionId]
			if !ok {
				s.log.Debug("Local connection does not exist, creating", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId, "addr", svc.localServiceAddr)
				// if there is no local connection assume that a remote instance is trying to connect to our local
				// service for the first time
				var err error
				conn, err = net.Dial("tcp", svc.localServiceAddr)
				svc.localConnections[msg.ConnectionId] = conn

				if err != nil {
					s.log.Error("Unable to create local connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId, "error", err)
					continue
				}
			}

			s.log.Debug("Writing data to connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
			conn.Write(msg.Data)
		case shipyard.MessageType_WRITE_DONE:
			// all data has been received, read from the local connection
			s.readData(msg)
		case shipyard.MessageType_CLOSED:
			svc := s.findService(msg.ServiceId)
			conn, ok := svc.localConnections[msg.ConnectionId]
			if !ok {
				s.log.Debug("Closing connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
				conn.Close()
			}
		}

	}

	return nil
}

func (s *Server) ExposeService(ctx context.Context, r *shipyard.ExposeRequest) (*shipyard.ExposeResponse, error) {
	s.log.Info("Expose Service", "req", r)

	// generate a unique id for the service
	id := uuid.New().String()
	svc := s.findService(id)
	svc.name = r.Name
	svc.localServiceAddr = r.ServiceAddr
	svc.localPort = int(r.LocalPort)
	svc.remotePort = int(r.RemotePort)
	svc.remoteServerAddr = r.RemoteServerAddr

	switch r.Type {
	case shipyard.ServiceType_REMOTE: // expose a remote service locally
		s.CreateListener(ctx, &shipyard.ListenerRequest{Id: id, LocalPort: r.LocalPort})
	case shipyard.ServiceType_LOCAL:
		c, err := s.getClient(r.RemoteServerAddr)
		if err != nil {
			s.log.Error("Unable to get remote client", "addr", r.RemoteServerAddr, "error", err)
			return nil, status.Error(codes.Internal, err.Error())
		}

		_, err = c.CreateListener(ctx, &shipyard.ListenerRequest{Id: id, LocalPort: r.RemotePort})
		if err != nil {
			s.log.Error("Unable to create remote listener", "addr", r.RemoteServerAddr, "error", err)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// we need to open a stream to the remote server
	s.log.Debug("Opening Stream to remote server", "addr", r.RemoteServerAddr)
	c, err := s.getClient(r.RemoteServerAddr)
	if err != nil {
		s.log.Error("Unable to get remote client", "addr", r.RemoteServerAddr, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	rc, err := c.OpenStream(context.Background())
	if err != nil {
		s.log.Error("Unable to open remote connection", "addr", r.RemoteServerAddr, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// ping the remote server a hello message
	rc.Send(&shipyard.OpenData{Type: shipyard.MessageType_HELLO, ServiceId: id, Location: svc.localServiceAddr})

	// add to the collection and handle messages
	svc.remoteConnection = &remoteConnection{id: id, conn: rc}
	s.handleRemoteConnection(id, rc)

	s.log.Debug("Done", "addr", r.RemoteServerAddr, "error", err)
	return &shipyard.ExposeResponse{Id: id}, nil
}

func (s *Server) DestroyService(context.Context, *shipyard.DestroyRequest) (*shipyard.NullResponse, error) {
	return nil, nil
}

func (s *Server) CreateListener(ctx context.Context, lr *shipyard.ListenerRequest) (*shipyard.NullResponse, error) {
	s.log.Info("Create Listener", "id", lr.Id, "port", lr.LocalPort)

	svc := s.findService(lr.Id)
	svc.localPort = int(lr.LocalPort)

	// create the listener
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", lr.LocalPort))
	if err != nil {
		s.log.Error("Unable to open TCP Listener", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	svc.listener = l

	s.handleListener(lr.Id, l)

	return &shipyard.NullResponse{}, nil
}

func (s *Server) DestroyListener(context.Context, *shipyard.ListenerRequest) (*shipyard.NullResponse, error) {
	return nil, nil
}

// Shutdown the server, closing all connections and listeners
func (s *Server) Shutdown() {
	s.log.Info("Shutting down")

	// close all listeners
	s.log.Info("Closing all TCPListeners")
	for _, t := range s.services {
		if t.listener != nil {
			t.listener.Close()
		}
	}
}

// get a gRPC client for the given address
func (s *Server) getClient(addr string) (shipyard.RemoteConnectionClient, error) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	return shipyard.NewRemoteConnectionClient(conn), nil
}

func (s *Server) handleRemoteConnection(id string, conn shipyard.RemoteConnection_OpenStreamClient) {
	// wrap in a go func to immediately return
	go func(id string, conn shipyard.RemoteConnection_OpenStreamClient) {
		for {
			msg, err := conn.Recv()
			if err != nil {
				s.log.Error("Error receiving client message", "id", id, "error", err)
				time.Sleep(1 * time.Second) // backoff for now, we need to handle failure better
				break
			}

			s.log.Debug("Received client message", "type", msg.Type, "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
			switch msg.Type {
			case shipyard.MessageType_DATA:
				s.log.Trace("Received client message", "id", id, "msg", msg)
				// if we get data send it to the local service instance
				// do we have a local connection, if not create one
				svc := s.findService(id)
				c, ok := svc.localConnections[msg.ConnectionId]
				if !ok {
					var err error
					c, err = net.Dial("tcp", svc.localServiceAddr)
					if err != nil {
						s.log.Error("Unable to create connection to remote", "addr", svc.localServiceAddr)
						continue
					}

					svc.localConnections[msg.ConnectionId] = c
				}

				s.log.Debug("Writing data to local", "serviceID", id, "connectionID", msg.ConnectionId)
				c.Write(msg.Data)
			case shipyard.MessageType_WRITE_DONE:
				s.readData(msg)
			}
		}
	}(id, conn)
}

// read data from the local and send back to the server
func (s *Server) readData(msg *shipyard.OpenData) {

	svc := s.findService(msg.ServiceId)
	con, ok := svc.localConnections[msg.ConnectionId]
	if !ok {
		s.log.Error("No connection to read from", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
		return
	}

	for {
		s.log.Debug("Reading data from local server", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)

		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		i, err := con.Read(data) // read 4k of data

		// if we had a read error tell the server
		if i == 0 || err != nil {
			// The server has closed the connection
			if err == io.EOF {
				// notify the remote
				svc.remoteConnection.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, ConnectionId: msg.ConnectionId, Type: shipyard.MessageType_CLOSED})
			} else {
				s.log.Error("Error reading from connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId, "error", err)
			}

			// cleanup
			delete(svc.localConnections, msg.ConnectionId)
			break
		}

		// send the data back to the server
		s.log.Debug("Sending data to remote connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
		svc.remoteConnection.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, ConnectionId: msg.ConnectionId, Type: shipyard.MessageType_DATA, Data: data[:i]})

		// all read close the connection
		if i < maxBuffer {
			//t := shipyard.MessageType_READ_DONE
			//s.closeConnection(msg, &t, sc)
			break
		}
	}
}

func (s *Server) handleListener(id string, l net.Listener) {
	// wrap in a go func to immediately return
	go func(id string, l net.Listener) {
		for {
			conn, err := l.Accept()
			if err != nil {
				s.log.Error("Error accepting connection", "id", id, "error", err)
				break
			}

			s.log.Debug("Handle new connection", "id", id)
			s.handleConnection(id, conn)
		}
	}(id, l)
}

func (s *Server) handleConnection(id string, conn net.Conn) {
	// generate a unique id for the connection
	connID := uuid.New().String()
	svc := s.findService(id)
	svc.localConnections[connID] = conn

	// read the data from the connection
	for {
		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		s.log.Debug("Starting read", "serviceID", id, "connID", connID)

		// read 4K of data from the connection
		i, err := conn.Read(data)

		// unable to read the data, kill the connection
		if err != nil || i == 0 {
			if err == io.EOF {
				// the connection has closed
				s.log.Debug("Connection closed", "serviceID", id, "connID", connID, "error", err)
			} else {
				s.log.Error("Unable to read data from the connection", "serviceID", id, "connID", connID, "error", err)
			}

			delete(svc.localConnections, connID)
			break
		}

		s.log.Trace("Read data for connection", "serviceID", id, "connID", connID, "len", i, "data", string(data[:i]))

		// send the read chunk of data over the gRPC stream
		// check there is a remote connection if not just return
		if gconn := s.findService(id).remoteConnection; gconn != nil {
			s.log.Debug("Sending data to stream", "serviceID", id, "connID", connID, "data", string(data[:i]))
			gconn.Send(&shipyard.OpenData{ServiceId: id, ConnectionId: connID, Type: shipyard.MessageType_DATA, Data: data[:i]})

			// we have read all the data send the other end a message so it knows it can now send a response
			if i < maxBuffer {
				s.log.Debug("All data read", "serviceID", id, "connID", connID)
				gconn.Send(&shipyard.OpenData{ServiceId: id, ConnectionId: connID, Type: shipyard.MessageType_WRITE_DONE})
			}
		} else {
			s.log.Error("No stream for connection", "serviceID", id, "connID", connID)
			break
		}

	}

}
