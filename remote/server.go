package remote

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/*
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
*/

type Server struct {
	log hclog.Logger
	// Collection which listeners and tcp connections for a Server stream
	streams streams
}

func New(l hclog.Logger) *Server {
	return &Server{
		l,
		streams{},
	}
}

// streams defines a collection of remote streams
// streams can be either opened outbound or inbound
type streams []*streamInfo

var streamMutex sync.Mutex

func (c *streams) add(si *streamInfo) {
	streamMutex.Lock()
	defer streamMutex.Unlock()

	*c = append(*c, si)
}

func (c *streams) remove(si *streamInfo) {
	streamMutex.Lock()
	defer streamMutex.Unlock()

	newSlice := streams{}
	for _, s := range *c {
		if s != si {
			newSlice = append(newSlice, si)
		}
	}

	*c = newSlice
}

func (c *streams) findByRemoteAddr(addr string) (*streamInfo, bool) {
	for _, v := range *c {
		if v.addr == addr {
			return v, true
		}
	}

	return nil, false
}

func (c *streams) findByServiceID(id string) (*streamInfo, bool) {
	for _, v := range *c {
		for id := range v.services {
			if id == id {
				return v, true
			}
		}
	}

	return nil, false
}

func (c *streams) findByRemoteConnection(rc interface{}) (*streamInfo, bool) {
	for _, v := range *c {
		if rc == v.grpcConn.conn {
			return v, true
		}
	}

	return nil, false
}

type streamInfo struct {
	connecting bool
	addr       string
	grpcConn   *grpcConn
	services   map[string]*service
}

func newStreamInfo() *streamInfo {
	return &streamInfo{
		services: map[string]*service{},
	}
}

type grpcConn struct {
	conn interface{}
}

func (r *grpcConn) Send(data *shipyard.OpenData) {
	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		c.Send(data)
	case shipyard.RemoteConnection_OpenStreamServer:
		c.Send(data)
	}
}

func (r *grpcConn) Recv() (*shipyard.OpenData, error) {
	if r == nil {
		return nil, nil
	}

	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		return c.Recv()
	case shipyard.RemoteConnection_OpenStreamServer:
		return c.Recv()
	}

	return nil, nil
}

func (r *grpcConn) Close() {
	if r != nil {
		switch c := r.conn.(type) {
		case shipyard.RemoteConnection_OpenStreamClient:
			c.CloseSend()
		}
	}
}

type service struct {
	exposeRequest  *shipyard.ExposeRequest
	tcpListener    net.Listener
	tcpConnections map[string]net.Conn
	status         kStatus
}

func newService() *service {
	return &service{tcpConnections: map[string]net.Conn{}}
}

type kStatus string

const kServicePending kStatus = "Pending"
const kServiceCreated kStatus = "Created"
const kServiceError kStatus = "Error"

func (s *Server) OpenStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	s.log.Info("Received new stream connection from client")

	gc := &grpcConn{svr}
	streamInfo := newStreamInfo()
	streamInfo.addr = "localhost" // this is an inbound connection
	streamInfo.grpcConn = gc

	s.streams.add(streamInfo)

	for {
		msg, err := svr.Recv()
		if err != nil {
			s.log.Error("Error receiving server message", "error", err)

			// We need to tear down any listeners related to this request and clean up resources
			// the downstream should attempt to re-establish the connection and resend the expose requests
			s.teardownConnection(streamInfo)
			break
		}

		s.log.Debug("Received server message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "type", reflect.TypeOf(msg.Message))

		switch m := msg.Message.(type) {
		case *shipyard.OpenData_Expose:
			// Does this already exist? If so it will be a repeat send so ignore
			_, ok := streamInfo.services[msg.ServiceId]
			if ok {
				continue
			}

			s.log.Info("Expose new service", "service_id", msg.ServiceId, "type", m.Expose.Type)

			// Otherwise create a new service
			streamInfo.services[msg.ServiceId] = newService()

			// The connection is exposing a local service to us
			// we need to open a TCP Listener for the service
			if m.Expose.Type == shipyard.ServiceType_LOCAL {
				l, err := s.createListenerAndListen(msg.ServiceId, int(m.Expose.SourcePort))
				if err != nil {
					// we need to send an error back to the connection
					continue
				}

				// set the listener on our collection
				streamInfo.services[msg.ServiceId].tcpListener = l
			}

			// set the remainder of the service info
			streamInfo.services[msg.ServiceId].exposeRequest = m.Expose
			streamInfo.services[msg.ServiceId].status = kServiceCreated

		case *shipyard.OpenData_Data:
			s.log.Trace("Message detail", "msg", m.Data.Data)

			// get the service for this data
			svc, ok := streamInfo.services[msg.ServiceId]

			// if there is no service for this message ignore it
			if !ok {
				s.log.Error("Service does not exist for message", "service_id", msg.ServiceId)
				continue
			}

			// get the connection
			tcpConn, ok := svc.tcpConnections[msg.ConnectionId]

			// no connection exists, if this is a remote service try to establish a new connection to the upstream service
			// otherwise ignore as the connection should have been created by the local listener
			if !ok {
				if svc.exposeRequest.Type == shipyard.ServiceType_LOCAL {
					s.log.Error("Local connection does not exist", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
					continue
				}

				// open a new connection
				s.log.Debug("Local connection does not exist, creating", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "addr", svc.exposeRequest.DestinationAddr)
				tcpConn, err = net.Dial("tcp", svc.exposeRequest.DestinationAddr)
				if err != nil {
					s.log.Error("Unable to create local connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "error", err)
					continue
				}

				svc.tcpConnections[msg.ConnectionId] = tcpConn
			}

			s.log.Debug("Writing data to connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
			tcpConn.Write(m.Data.Data)

		case *shipyard.OpenData_WriteDone:
			// all data has been received, read from the local connection
			s.readData(msg)

		case *shipyard.OpenData_Closed:
			// remote end of the connection has been closed, close this end
			svc, ok := streamInfo.services[msg.ServiceId]

			// if there is no service for this message ignore it
			if !ok {
				s.log.Error("Service does not exist for message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
				continue
			}

			conn, ok := svc.tcpConnections[msg.ConnectionId]

			// no connection exists
			if !ok {
				s.log.Error("Connection does not exist for message", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
				continue
			}

			s.log.Debug("Closing connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
			conn.Close()

			delete(svc.tcpConnections, msg.ConnectionId)
		}
	}

	return nil
}

func (s *Server) ExposeService(ctx context.Context, r *shipyard.ExposeRequest) (*shipyard.ExposeResponse, error) {
	// generate a unique id for the service
	id := uuid.New().String()
	s.log.Info("Expose Service", "req", r, "service_id", id)

	svc := newService()
	svc.exposeRequest = r
	svc.status = kServicePending

	// find a remote connection
	si, ok := s.streams.findByRemoteAddr(r.RemoteConnectorAddr)
	if !ok {
		si = newStreamInfo()
		si.addr = r.RemoteConnectorAddr

		// add the new stream to the collection
		s.streams.add(si)
	}

	// add the service to the connection
	si.services[id] = svc

	// establish a connection to the remote endpoint and setup listeners
	go s.handleReconnection(si)

	return &shipyard.ExposeResponse{Id: id}, nil
}

func (s *Server) DestroyService(context.Context, *shipyard.DestroyRequest) (*shipyard.NullResponse, error) {
	return nil, nil
}

// Shutdown the server, closing all connections and listeners
func (s *Server) Shutdown() {
	s.log.Info("Shutting down")

	// close all listeners
	s.log.Info("Closing all TCPListeners and Connections")
	for _, t := range s.streams {
		s.teardownConnection(t)
		t.grpcConn.Close()
	}
}

var connectionBackoff = 10 * time.Second

func (s *Server) handleReconnection(conn *streamInfo) error {
	// if is possible that this method gets called multiple times
	// ensure there is only one operation in process at once
	if conn.connecting {
		s.log.Info("Connection attempt already in process", "addr", conn.addr)
		return nil
	}

	// mark we are attempting to connect
	conn.connecting = true
	defer func() {
		conn.connecting = false
	}()

	for {
		// if we do not have a connection create one
		if conn.grpcConn == nil {
			// connect to the service
			gc, err := s.openRemoteConnection(conn.addr)
			if err != nil {
				s.log.Error("Unable to open remote connection", "error", err)

				// back off and try again
				time.Sleep(connectionBackoff)
				continue
			}

			// set the connection
			conn.grpcConn = gc

			// handle messages for this stream
			s.handleRemoteConnection(conn)
		}

		// loop all services and try to reconfigure
		for id, svc := range conn.services {
			// set up all the local listeners if the type is remote and the listener does not already
			// exist
			if svc.exposeRequest.Type == shipyard.ServiceType_REMOTE && svc.tcpListener == nil {
				// open the listener locally
				l, err := s.createListenerAndListen(id, int(svc.exposeRequest.SourcePort))
				if err != nil {
					s.log.Error("Unable to create listener for service", "service_id", id, "error", err)
					continue
				}

				// add the listener to the service
				svc.tcpListener = l
			}

			// send the expose message to the remote so it can open
			s.log.Debug("Sending expose message to remote component", "addr", svc.exposeRequest.RemoteConnectorAddr)
			req := &shipyard.OpenData{ServiceId: id}
			req.Message = &shipyard.OpenData_Expose{Expose: svc.exposeRequest}

			conn.grpcConn.Send(req)
			svc.status = kServiceCreated
		}

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

	return &grpcConn{rc}, nil
}

func (s *Server) handleRemoteConnection(si *streamInfo) {
	// wrap in a go func to immediately return
	go func(si *streamInfo) {
		for {
			msg, err := si.grpcConn.Recv()
			if err != nil {
				s.log.Error("Error receiving client message", "addr", si.addr, "error", err)

				// We need to tear down any listeners related to this request and clean up resources
				// the downstream should attempt to re-establish the connection and resend the expose requests
				s.teardownConnection(si)

				s.handleReconnection(si)
				break // exit this loop as handleReconnection will recall ths function when a connection is established
			}

			s.log.Debug("Received client message", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
			switch m := msg.Message.(type) {
			case *shipyard.OpenData_Data:
				s.log.Trace("Received client message", "service_id", msg.ServiceId, "msg", msg)

				// if we get data send it to the local service instance
				// do we have a local connection, if not create one
				svc := si.services[msg.ServiceId]
				c, ok := svc.tcpConnections[msg.ConnectionId]
				if !ok {
					// is this an message reply to a local listener, if there is no connection
					// assume it has gone away so ignore
					if svc.exposeRequest.Type == shipyard.ServiceType_REMOTE {
						s.log.Error("Connection does not exist for local listener", "port", svc.exposeRequest.SourcePort)
						continue
					}

					// otherwise create a new upstream connection
					var err error
					c, err = net.Dial("tcp", svc.exposeRequest.DestinationAddr)
					if err != nil {
						s.log.Error("Unable to create connection to remote", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId, "addr", svc.exposeRequest.DestinationAddr)
						continue
					}

					// set the connection
					svc.tcpConnections[msg.ConnectionId] = c
				}

				s.log.Debug("Writing data to local", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
				c.Write(m.Data.Data)

			case *shipyard.OpenData_WriteDone:
				// all writing has been completed for the connection switch to read mode
				s.readData(msg)
			case *shipyard.OpenData_Closed:
				svc := si.services[msg.ServiceId]
				c, ok := svc.tcpConnections[msg.ConnectionId]
				if ok {
					s.log.Debug("Closing connection", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
					// we have a connection close it
					c.Close()
					delete(svc.tcpConnections, msg.ConnectionId)
				}
			}
		}
	}(si)
}

func (s *Server) teardownConnection(conn *streamInfo) {
	for _, s := range conn.services {
		// close any open connections
		for id, c := range s.tcpConnections {
			c.Close()
			delete(s.tcpConnections, id)
		}

		// close the listener
		if s.tcpListener != nil {
			s.tcpListener.Close()
			s.tcpListener = nil
		}

		s.status = kServicePending
	}

	conn.grpcConn = nil

}

// get a gRPC client for the given address
func (s *Server) getClient(addr string) (shipyard.RemoteConnectionClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	return shipyard.NewRemoteConnectionClient(conn), nil
}

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

	svc, ok := s.streams.findByServiceID(serviceID)
	if !ok {
		// no service exists for this connection, close and return, this should never happen
		s.log.Error("No stream exists for ", "service_id", serviceID, "connection_id", connID)
		return
	}

	// set the new connection
	svc.services[serviceID].tcpConnections[connID] = conn

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
				svc.grpcConn.Send(
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

			delete(svc.services[serviceID].tcpConnections, connID)
			break
		}

		s.log.Trace("Read data for connection", "service_id", serviceID, "connection_id", connID, "len", i, "data", string(data[:i]))

		// send the read chunk of data over the gRPC stream
		// check there is a remote connection if not just return
		s.log.Debug("Sending data to stream", "service_id", serviceID, "connection_id", connID, "data", string(data[:i]))
		svc.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    serviceID,
				ConnectionId: connID,
				Message:      &shipyard.OpenData_Data{Data: &shipyard.Data{Data: data[:i]}},
			},
		)

		// we have read all the data send the other end a message so it knows it can now send a response
		if i < maxBuffer {
			s.log.Debug("All data read", "service_id", serviceID, "connID", connID)
			svc.grpcConn.Send(&shipyard.OpenData{ServiceId: serviceID, ConnectionId: connID, Message: &shipyard.OpenData_WriteDone{}})
		}
	}
}

// read data from the local and send back to the server
func (s *Server) readData(msg *shipyard.OpenData) {
	svc, _ := s.streams.findByServiceID(msg.ServiceId)
	con, ok := svc.services[msg.ServiceId].tcpConnections[msg.ConnectionId]
	if !ok {
		s.log.Error("No connection to read from", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)
		return
	}

	for {
		s.log.Debug("Reading data from local server", "service_id", msg.ServiceId, "connection_id", msg.ConnectionId)

		maxBuffer := 4096
		data := make([]byte, maxBuffer)

		i, err := con.Read(data) // read 4k of data

		// if we had a read error tell the server
		if i == 0 || err != nil {
			// The server has closed the connection
			if err == io.EOF {
				// notify the remote
				svc.grpcConn.Send(
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
			delete(svc.services[msg.ServiceId].tcpConnections, msg.ConnectionId)
			break
		}

		// send the data back to the server
		s.log.Debug("Sending data to remote connection", "serviceID", msg.ServiceId, "connectionID", msg.ConnectionId)
		svc.grpcConn.Send(
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
