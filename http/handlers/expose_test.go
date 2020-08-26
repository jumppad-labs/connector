package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testClient struct {
	mock.Mock
}

func (t *testClient) OpenStream(ctx context.Context, opts ...grpc.CallOption) (shipyard.RemoteConnection_OpenStreamClient, error) {
	return nil, nil
}

func (t *testClient) ExposeService(ctx context.Context, in *shipyard.ExposeRequest, opts ...grpc.CallOption) (*shipyard.ExposeResponse, error) {
	return &shipyard.ExposeResponse{Id: "test"}, nil
}

func (t *testClient) DestroyService(ctx context.Context, in *shipyard.DestroyRequest, opts ...grpc.CallOption) (*shipyard.NullMessage, error) {
	return nil, nil
}

func (t *testClient) ListServices(ctx context.Context, in *shipyard.NullMessage, opts ...grpc.CallOption) (*shipyard.ServiceResponse, error) {
	return nil, nil
}

func TestNoBodyBadReqest(t *testing.T) {
	h := NewExpose(&testClient{}, hclog.Default())
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{'dfdf'"))

	h.ServeHTTP(rr, r)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInvalidBodyUnprocessableEntity(t *testing.T) {
	cr := &ExposeRequest{}
	d, _ := json.Marshal(cr)

	h := NewExpose(&testClient{}, hclog.Default())
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(d))

	h.ServeHTTP(rr, r)

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}
