package remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/client"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var messageData []byte
var lock = sync.Mutex{}

// create the server and client to connect to the server
func setupServerTests(t *testing.T) *client.Client {
	// start the gRPC server
	s := NewServer(hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Name: "grpc_server"}))
	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)

	// create a listener for the server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 12344))
	assert.NoError(t, err)

	// start the server in the background
	go grpcServer.Serve(lis)

	// generate a client
	c, err := client.New(":12344", hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Name: "grpc_client"}))
	require.NoError(t, err)

	t.Cleanup(func() {
		s.Shutdown()
		grpcServer.Stop()
		lis.Close()
	})

	return c
}

// start a Local HTTP server which can be used for testing
func startLocalServer(t *testing.T, bodyData *string) string {

	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Name: "http_server"})

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		l.Debug("Got request")

		data, _ := ioutil.ReadAll(r.Body)
		*bodyData = string(data)
	}))

	t.Cleanup(func() {
		ts.Close()
	})

	return ts.Listener.Addr().String()
}

func TestCreateOpensTCPConnectionAndStreamsData(t *testing.T) {
	response := ""
	c := setupServerTests(t)
	tsAddr := startLocalServer(t, &response)

	go c.OpenLocalConnection(19090, "test_server", tsAddr)
	time.Sleep(100 * time.Millisecond) // wait for the server to connect

	assertRequest(t, &response, "abc123")
	assertRequest(t, &response, "123abc")
}

func TestCloseClosesTCPConnection(t *testing.T) {
	response := ""
	c := setupServerTests(t)
	tsAddr := startLocalServer(t, &response)

	go c.OpenLocalConnection(19090, "test_server", tsAddr)
	time.Sleep(100 * time.Millisecond) // wait for the server to connect

	assertRequest(t, &response, "abc123")

	c.CloseLocalConnection(19090)

	// this should fail as the connection is now closed
	_, err := http.DefaultClient.Get("http://localhost:19090")
	require.Error(t, err)
}

func TestLocalOpensTCPConnectionAndStreamsData(t *testing.T) {
	response := ""
	c := setupServerTests(t)
	tsAddr := startLocalServer(t, &response)

	go c.OpenRemoteConnection(19090, "test_server", tsAddr)
	defer c.CloseRemoteConnection(19090)

	time.Sleep(100 * time.Millisecond) // wait for the server to connect

	assertRequest(t, &response, "abc123")
	assertRequest(t, &response, "123abc")
}

func assertRequest(t *testing.T, response *string, data string) {
	// test we can connect to the remote server socket
	resp, err := http.DefaultClient.Post("http://localhost:19090", "application/json", bytes.NewBufferString(data))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, data, *response)

	*response = ""
}
