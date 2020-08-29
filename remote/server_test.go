package remote

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func createServer(t *testing.T, addr, name string) {
	// start the gRPC server
	s := New(hclog.New(&hclog.LoggerOptions{Level: hclog.Trace, Name: name}))
	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)

	// create a listener for the server
	lis, err := net.Listen("tcp", addr)
	require.NoError(t, err)

	// start the server in the background
	go grpcServer.Serve(lis)

	t.Cleanup(func() {
		s.Shutdown()
		grpcServer.Stop()
		lis.Close()
	})
}

func startLocalServer(t *testing.T) (string, *string) {
	bodyData := ""

	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Trace, Name: "http_server"})

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		l.Debug("Got request")

		data, _ := ioutil.ReadAll(r.Body)
		bodyData = string(data)
	}))

	t.Cleanup(func() {
		ts.Close()
	})

	return ts.Listener.Addr().String(), &bodyData
}

func setupServers(t *testing.T) (string, *string) {
	// local server
	createServer(t, ":1234", "server_local")
	createServer(t, ":1235", "server_remote_1")
	createServer(t, ":1236", "server_remote_2")

	// setup the local endpoint
	return startLocalServer(t)
}

func createClient(t *testing.T, addr string) shipyard.RemoteConnectionClient {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	require.NoError(t, err)

	return shipyard.NewRemoteConnectionClient(conn)
}

func setupTests(t *testing.T) (shipyard.RemoteConnectionClient, string, *string) {
	tsAddr, tsData := setupServers(t)
	return createClient(t, "localhost:1234"), tsAddr, tsData
}

func TestExposeRemoteServiceCreatesLocalListener(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19000")
	require.NoError(t, err)
}
func TestExposeRemoteServiceCreatesLocalListener2(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19000")
	require.NoError(t, err)
}

func TestExposeRemoteServiceUpdatesStatus(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	require.Eventually(t,
		func() bool {
			s, _ := c.ListServices(context.Background(), &shipyard.NullMessage{})
			if len(s.Services) > 0 {
				if s.Services[0].Status == shipyard.ServiceStatus_COMPLETE {
					return true
				}
			}

			return false
		},
		1*time.Second,
		50*time.Millisecond,
	)
}

func TestExposeRemoteDuplicateReturnsError(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test1",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test2",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.Error(t, err)
}

func TestExposeLocalDuplicateReturnsError(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test1",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test2",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.Error(t, err)
}
func TestExposeLocalDifferentServersReturnsOK(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test1",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test2",
			RemoteConnectorAddr: "localhost:1236",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
}

func TestExposeLocalDifferentConnectionsReturnsError(t *testing.T) {
	c, _, _ := setupTests(t)
	c2 := createClient(t, "localhost:1235")

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test1",
			RemoteConnectorAddr: "localhost:1236",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	require.Eventually(t,
		func() bool {
			s, _ := c.ListServices(context.Background(), &shipyard.NullMessage{})
			if len(s.Services) > 0 {
				if s.Services[0].Status == shipyard.ServiceStatus_COMPLETE {
					return true
				}
			}

			return false
		},
		1*time.Second,
		50*time.Millisecond,
	)

	_, err = c2.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test2",
			RemoteConnectorAddr: "localhost:1236",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// Both connections will return OK as there is no local validation failure
	// however the second connection will fail and the server will return a message
	// as the listener is in use

	require.Eventually(t,
		func() bool {
			s, _ := c2.ListServices(context.Background(), &shipyard.NullMessage{})
			if len(s.Services) > 0 {
				if s.Services[0].Status == shipyard.ServiceStatus_ERROR {
					return true
				}
			}

			return false
		},
		1*time.Second,
		50*time.Millisecond,
	)
}

func TestDestroyRemoteServiceRemovesLocalListener(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19000,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	require.Eventually(t,
		func() bool {
			s, _ := c.ListServices(context.Background(), &shipyard.NullMessage{})
			if len(s.Services) > 0 {
				if s.Services[0].Status == shipyard.ServiceStatus_COMPLETE {
					return true
				}
			}

			return false
		},
		1*time.Second,
		50*time.Millisecond,
	)

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19000")
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener is not accessible
	_, err = net.Dial("tcp", "localhost:19000")
	require.Error(t, err)
}

func TestExposeLocalServiceCreatesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19001,
			DestinationAddr:     "localhost:19000",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19001")
	require.NoError(t, err)
}

func TestDestroyLocalServiceRemovesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			SourcePort:          19001,
			DestinationAddr:     "localhost:19000",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond)

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19001")
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener is not accessible
	_, err = net.Dial("tcp", "localhost:19001")
	require.Error(t, err)
}

func TestExposeLocalServiceCreatesRemoteConnection(t *testing.T) {
	t.Skip()
}

func TestExposeRemoteServiceCreatesRemoteConnection(t *testing.T) {
	t.Skip()
}

func TestMessageToRemoteEndpointCallsLocalService(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			DestinationAddr:     tsAddr,
			SourcePort:          19001,
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	// call the remote endpoint
	httpResp, err := http.DefaultClient.Get("http://localhost:19001")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	httpResp, err = http.DefaultClient.Get("http://localhost:19001")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)
}

func TestMessageToLocalEndpointCallsRemoteService(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			DestinationAddr:     tsAddr,
			SourcePort:          19001,
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	// call the remote endpoint
	httpResp, err := http.DefaultClient.Get("http://localhost:19001")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	httpResp, err = http.DefaultClient.Get("http://localhost:19001")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)
}

func TestListServices(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: "localhost:1235",
			DestinationAddr:     tsAddr,
			SourcePort:          19001,
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	s, err := c.ListServices(context.Background(), &shipyard.NullMessage{})
	require.NoError(t, err)
	require.Len(t, s.Services, 1)
}
