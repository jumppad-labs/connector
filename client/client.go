package client

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
)

type Client struct {
	rc          shipyard.RemoteConnectionClient
	log         hclog.Logger
	connections map[string]net.Conn
	remoteIDs   map[int]string
}

// New creates a new client
func New(addr string, log hclog.Logger) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, fmt.Errorf("Unable to open connection to server %s", err)
	}

	c := shipyard.NewRemoteConnectionClient(conn)

	return &Client{c, log, map[string]net.Conn{}, map[int]string{}}, nil
}

// OpenRemoteConnection opens a connection between a remote port and a local service
func (c *Client) OpenRemoteConnection(remotePort int, name string, localAddr string) error {
	er, err := c.rc.ExposeLocalService(context.Background(), &shipyard.ExposeRequest{Name: name, Port: int32(remotePort)})

	c.remoteIDs[remotePort] = er.Id

	// establish a stream with the server
	sc, err := c.rc.Open(context.Background())
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

func (c *Client) CloseRemoteConnection(remotePort int) {
	c.rc.DestroyLocalService(context.Background(), &shipyard.DestroyRequest{Id: c.remoteIDs[remotePort]})
}

func (c *Client) newConnection(msg *shipyard.OpenData, localAddr string) error {
	var err error
	con, err := net.Dial("tcp", localAddr)
	if err != nil {
		c.log.Error("Unable to open connection to remote server", "addr", localAddr, "error", err)
		return err
	}

	c.connections[msg.RequestId] = con
	return nil
}

func (c *Client) closeConnection(msg *shipyard.OpenData, t *shipyard.MessageType, sc shipyard.RemoteConnection_OpenClient) {
	if conn, ok := c.connections[msg.RequestId]; ok {
		conn.Close()
		delete(c.connections, msg.RequestId)

		if t != nil {
			// send the close message back to the server
			sc.Send(&shipyard.OpenData{ServiceId: msg.ServiceId, RequestId: msg.RequestId, Type: *t})
		}
	}
}

func (c *Client) readData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenClient) {
	con, ok := c.connections[msg.RequestId]
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

func (c *Client) writeData(msg *shipyard.OpenData, sc shipyard.RemoteConnection_OpenClient) {
	con, ok := c.connections[msg.RequestId]
	if ok {
		_, err := con.Write(msg.Data)
		if err != nil {
			c.log.Error("Unable to write data to connection", "error", err)
		}
	}
}
