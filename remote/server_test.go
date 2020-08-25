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
	createServer(t, ":1235", "server_remote")

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
		Name:                "Test Service",
		RemoteConnectorAddr: "localhost:1235",
		SourcePort:          19000,
		DestinationAddr:     "localhost:19001",
		Type:                shipyard.ServiceType_REMOTE,
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19000")
	require.NoError(t, err)
}

func TestExposeLocalServiceCreatesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Name:                "Test Service",
		RemoteConnectorAddr: "localhost:1235",
		SourcePort:          19001,
		DestinationAddr:     "localhost:19000",
		Type:                shipyard.ServiceType_LOCAL,
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond)

	// check the listener exists
	_, err = net.Dial("tcp", "localhost:19001")
	require.NoError(t, err)
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
		Name:                "Test Service",
		RemoteConnectorAddr: "localhost:1235",
		DestinationAddr:     tsAddr,
		SourcePort:          19001,
		Type:                shipyard.ServiceType_LOCAL,
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
		Name:                "Test Service",
		RemoteConnectorAddr: "localhost:1235",
		DestinationAddr:     tsAddr,
		SourcePort:          19001,
		Type:                shipyard.ServiceType_REMOTE,
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
