package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/shipyard-run/connector/remote"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNoBodyBadReqest(t *testing.T) {
	h := NewCreate(hclog.Default())
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{'dfdf'"))

	h.ServeHTTP(rr, r)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInvalidBodyUnprocessableEntity(t *testing.T) {
	cr := &CreateRequest{}
	d, _ := json.Marshal(cr)

	h := NewCreate(hclog.Default())
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(d))

	h.ServeHTTP(rr, r)

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

var upstreamCalled bool

func TestCreateRequestOpensLocalPort(t *testing.T) {
	logger := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug})

	// start a test upstream
	ts := createTestHTTPServer(t)
	l, tg := createTestGrpcServer(t)

	t.Cleanup(func() {
		ts.Close()
		tg.Shutdown()
		l.Close()
	})

	cr := &CreateRequest{
		Port:           30001,
		Service:        ts.URL,
		RemoteLocation: "localhost:30000",
	}

	d, _ := json.Marshal(cr)

	h := NewCreate(logger)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(d))

	h.ServeHTTP(rr, r)

	// call the local endpoint
	time.Sleep(1 * time.Second)
	resp, err := http.DefaultClient.Get("http://localhost:30001")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.True(t, upstreamCalled)
}

func createTestHTTPServer(t *testing.T) *httptest.Server {
	// start a test server
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				upstreamCalled = true
			},
		),
	)

	return ts
}

func createTestGrpcServer(t *testing.T) (net.Listener, *remote.Server) {
	// start the gRPC server
	s := remote.NewServer()
	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)

	// create a listener for the server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 30000))
	require.NoError(t, err)

	// start the server in the background
	go grpcServer.Serve(lis)

	return lis, s
}
