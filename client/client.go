package client

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
)

type Client struct {
	rc          shipyard.RemoteConnectionClient
	log         hclog.Logger
	remoteIDs   map[int]string
	localIDs    map[int]string
	listeners   map[string]net.Listener
	tcpConn     map[string]net.Conn
	connections map[string]shipyard.RemoteConnection_OpenRemoteClient
}

// New creates a new client
func New(addr string, log hclog.Logger) (*Client, error) {
	var c shipyard.RemoteConnectionClient

	if addr != "" {
		conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
		if err != nil {
			return nil, fmt.Errorf("Unable to open connection to server %s", err)
		}

		c = shipyard.NewRemoteConnectionClient(conn)
	}

	return &Client{
		c,
		log,
		map[int]string{},
		map[int]string{},
		map[string]net.Listener{},
		map[string]net.Conn{},
		map[string]shipyard.RemoteConnection_OpenRemoteClient{},
	}, nil
}

// OpenLocalConnection opens a connection between a remote port and a local service
func (c *Client) OpenLocalConnection(remotePort int, name string, localAddr string) error {
	er, err := c.rc.ExposeLocalService(context.Background(), &shipyard.ExposeRequest{Name: name, Port: int32(remotePort)})

	c.remoteIDs[remotePort] = er.Id

	// establish a stream with the server
	sc, err := c.rc.OpenLocal(context.Background())
	if err != nil {
		c.log.Error("Unable to handle traffic for remote service", "error", err)
		return err
	}

	for {
		msg, err := sc.Recv()
		if err != nil {
			c.log.Error("Error receiving message", "error", err)
			return err
		}

		c.log.Debug("Received message", "message", msg)

		switch msg.Type {
		case shipyard.MessageType_NEW_CONNECTION:
			c.newConnection(msg, localAddr)
		case shipyard.MessageType_DATA:
			c.writeData(msg, sc) // write data from the remote connection to the local endpoint
		case shipyard.MessageType_WRITE_DONE:
			c.readData(msg, sc) // read the response and send back to the server
		case shipyard.MessageType_ERROR:
			c.closeConnection(msg, nil, sc) // read the response and send back to the server
		}
	}
}

func (c *Client) CloseLocalConnection(remotePort int) {
	c.rc.DestroyLocalService(context.Background(), &shipyard.DestroyRequest{Id: c.remoteIDs[remotePort]})
}

// OpenRemoteConnection opens a connection on the local host which connects to a service on a remote machine
func (c *Client) OpenRemoteConnection(localPort int, name string, remoteAddr string) error {
	id := uuid.New().String()
	c.localIDs[localPort] = id

	c.log.Info("Exposing Remote Service", "name", name, "id", id)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		c.log.Error("Unable to create listener", "error", err)
		return err
	}

	c.listeners[id] = l

	go c.tcpListen(id, remoteAddr)

	con, err := c.rc.OpenRemote(context.Background())
	if err != nil {
		return err
	}

	c.connections["a"] = con

	for {
		msg, err := con.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			c.log.Error("Error receiving data", "error", err)
			// clean up the connection
			delete(c.connections, "a")

			break
		}

		c.log.Debug("Got message from stream", "id", msg.ServiceId, "rid", msg.RequestId, "type", msg.Type)

		// attempt top get a local listener for the message
		switch msg.Type {
		case shipyard.MessageType_DATA:
			if conn, ok := c.tcpConn[msg.RequestId]; ok {
				conn.Write(msg.Data)
			}
		case shipyard.MessageType_READ_DONE:
			if conn, ok := c.tcpConn[msg.RequestId]; ok {
				conn.Close()
			}
			delete(c.tcpConn, msg.RequestId)
		case shipyard.MessageType_ERROR:
			c.log.Error("Error from remote endpoint", "message", msg)

			if conn, ok := c.tcpConn[msg.RequestId]; ok {
				conn.Close()
			}
			delete(c.tcpConn, msg.RequestId)
		}
	}

	return nil
}

func (c *Client) CloseRemoteConnection(localPort int) {
	id := c.localIDs[localPort]
	if l, ok := c.listeners[id]; ok {
		l.Close()
	}
}

func (c *Client) newConnection(msg *shipyard.OpenData, localAddr string) error {
	var err error
	con, err := net.Dial("tcp", localAddr)
	if err != nil {
		c.log.Error("Unable to open connection to remote server", "addr", localAddr, "error", err)
		return err
	}

	c.tcpConn[msg.RequestId] = con
	return nil
}

func (c *Client) closeConnection(msg *shipyard.OpenData, t *shipyard.MessageType, sc shipyard.RemoteConnection_OpenLocalClient) {
	if conn, ok := c.tcpConn[msg.RequestId]; ok {
		conn.Close()
		delete(c.tcpConn, msg.RequestId)

		if t != nil {
			// send the close message back to the server
			sc.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, RequestId: msg.RequestId, Type: *t})
		}
	}
}

func (c *Client) readData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenLocalClient) {
	con, ok := c.tcpConn[msg.RequestId]
	if ok {
		for {
			c.log.Debug("Reading data from local server", "rid", msg.GetRequestId)

			maxBuffer := 4096
			data := make([]byte, maxBuffer)

			i, err := con.Read(data) // read 4k of data

			// if we had a read error tell the server
			if i == 0 || err != nil {
				t := shipyard.MessageType_ERROR
				c.closeConnection(msg, &t, sc)
				break
			}

			// send the data back to the server
			c.log.Debug("Sending data to remote connection", "rid", msg.GetRequestId)
			sc.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, RequestId: msg.RequestId, Type: shipyard.MessageType_DATA, Data: data[:i]})

			// all read close the connection
			if i < maxBuffer {
				t := shipyard.MessageType_READ_DONE
				c.closeConnection(msg, &t, sc)
				break
			}
		}
	}
}

func (c *Client) writeData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenLocalClient) {
	con, ok := c.tcpConn[msg.RequestId]
	if ok {
		_, err := con.Write(msg.Data)
		if err != nil {
			c.log.Error("Unable to write data to connection", "error", err)
		}
	}
}

func (c *Client) tcpListen(id string, remoteAddr string) {
	l := c.listeners[id]
	for {
		// accept the next connection in the queue
		conn, err := l.Accept()
		if err != nil {
			c.log.Error("Unable to accept connection", "error", err)
			break
		}

		// work on the connection in the background to enable the next connection to be handled concurrently
		c.log.Debug("Handle new connection", "id", id)
		rid := uuid.New().String() // generate a new request id
		c.tcpConn[rid] = conn

		// send the new connection message
		c.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Location: remoteAddr, Type: shipyard.MessageType_NEW_CONNECTION})

		go func(conn net.Conn, id string, rid string) {
			for {
				maxBuffer := 4096
				data := make([]byte, maxBuffer)

				// read 4K of data from the connection
				// if no data left to read break
				c.log.Debug("Starting read", "service", id, "rid", rid)

				i, err := conn.Read(data)
				if err != nil || i == 0 {
					c.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_ERROR})
					break
				}

				// send the read chunk of data over the gRPC stream
				c.log.Debug("Read data for connection", "service", id, "rid", rid, "len", i, "data", string(data[:i]))

				// check there is a connection if not just return
				if gconn := c.connections["a"]; gconn != nil {
					c.log.Debug("Sending data to stream", "service", id, "rid", rid, "data", string(data[:i]))

					gconn.Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_DATA, Data: data[:i]})
				}

				if i < maxBuffer {
					c.log.Debug("All data read", "service", id, "rid", rid)
					c.connections["a"].Send(&shipyard.OpenData{ServiceId: id, RequestId: rid, Type: shipyard.MessageType_WRITE_DONE})
					break
				}
			}
		}(conn, id, rid)
	}
}
