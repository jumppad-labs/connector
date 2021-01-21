package remote

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/integrations"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func createServer(t *testing.T, addr, name string) (*Server, *integrations.Mock) {
	//certificate, err := tls.LoadX509KeyPair("/tmp/certs/leaf.cert", "/tmp/certs/leaf.key")
	//require.NoError(t, err)

	//// Create a certificate pool from the certificate authority
	//certPool := x509.NewCertPool()
	//ca, err := ioutil.ReadFile("/tmp/certs/root.cert")
	//require.NoError(t, err)

	//ok := certPool.AppendCertsFromPEM(ca)
	//require.True(t, ok)

	// create the mock
	mi := &integrations.Mock{}
	mi.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mi.On("Deregister", mock.Anything).Return(nil)

	output := bytes.NewBufferString("")

	l := hclog.New(
		&hclog.LoggerOptions{
			Level:  hclog.Trace,
			Name:   name,
			Output: output,
		},
	)

	// start the gRPC server
	//s := New(l, certPool, &certificate, mi)
	s := New(l, nil, nil, mi)

	//creds := credentials.NewTLS(&tls.Config{
	//	ClientAuth:   tls.RequireAndVerifyClientCert,
	//	Certificates: []tls.Certificate{certificate},
	//	ClientCAs:    certPool,
	//})

	//grpcServer := grpc.NewServer(grpc.Creds(creds))
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

		if t.Failed() || os.Getenv("DEBUG") == "true" {
			t.Log(output.String())
		}
	})

	return s, mi
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

type serverStruct struct {
	Server      *Server
	Port        int
	Address     string
	Integration *integrations.Mock
}

var servers []serverStruct

func setupServers(t *testing.T) (string, *string) {
	p1 := rand.Intn(20000) + 40000
	p2 := rand.Intn(20000) + 40000
	p3 := rand.Intn(20000) + 40000

	a1 := fmt.Sprintf("localhost:%d", p1)
	a2 := fmt.Sprintf("localhost:%d", p2)
	a3 := fmt.Sprintf("localhost:%d", p3)

	// local server
	s1, m1 := createServer(t, a1, "server_local_1")
	s2, m2 := createServer(t, a2, "server_local_2")
	s3, m3 := createServer(t, a3, "server_local_3")

	servers = []serverStruct{
		{s1, p1, a1, m1},
		{s2, p2, a2, m2},
		{s3, p3, a3, m3},
	}

	// setup the local endpoint
	return startLocalServer(t)
}

func createClient(t *testing.T, addr string) shipyard.RemoteConnectionClient {
	//certificate, err := tls.LoadX509KeyPair("/tmp/certs/leaf.cert", "/tmp/certs/leaf.key")
	//require.NoError(t, err)

	// Create a certificate pool from the certificate authority
	//certPool := x509.NewCertPool()
	//ca, err := ioutil.ReadFile("/tmp/certs/root.cert")
	//require.NoError(t, err)

	//ok := certPool.AppendCertsFromPEM(ca)
	//require.True(t, ok)

	//creds := credentials.NewTLS(&tls.Config{
	//	ServerName:   addr,
	//	Certificates: []tls.Certificate{certificate},
	//	RootCAs:      certPool,
	//})

	// Create a connection with the TLS credentials
	//conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	require.NoError(t, err)

	return shipyard.NewRemoteConnectionClient(conn)
}

func setupTests(t *testing.T) (shipyard.RemoteConnectionClient, string, *string) {
	tsAddr, tsData := setupServers(t)
	return createClient(t, servers[0].Address), tsAddr, tsData
}

func TestExposeRemoteServiceCreatesLocalListener(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)
}

func TestExposeRemoteServiceCallsIntegration(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	servers[0].Integration.AssertCalled(t, "Register", mock.Anything, "test-service", int(p), int(p))
}

func TestShutdownRemovesLocalListener(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// shutdown
	for _, s := range servers {
		s.Server.Shutdown()
	}

	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.Error(t, err)
}

func TestShutdownRemovesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// shutdown
	for _, s := range servers {
		s.Server.Shutdown()
	}

	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.Error(t, err)
}

func TestExposeRemoteServiceCreatesLocalListener2(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)
}

func TestExposeRemoteServiceUpdatesStatus(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
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

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 2",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.Error(t, err)
}

func TestExposeLocalDuplicateReturnsError(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 2",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.Error(t, err)
}
func TestExposeLocalDifferentServersReturnsOK(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	_, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test2",
			RemoteConnectorAddr: servers[2].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
}

func TestExposeLocalDifferentConnectionsReturnsError(t *testing.T) {
	c, _, _ := setupTests(t)
	c2 := createClient(t, servers[1].Address)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[2].Address,
			SourcePort:          p,
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

	resp, err = c2.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 2",
			RemoteConnectorAddr: servers[2].Address,
			SourcePort:          p,
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

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
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
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener is not accessible
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.Error(t, err)
}

func TestDestroyRemoteServiceRemovesLocalIntegration(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
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
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	servers[1].Integration.AssertCalled(t, "Deregister", "test-1")
}

func TestExposeLocalServiceCreatesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)
}

func TestExposeLocalServiceCallsRemoteIntegration(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond) // wait for setup

	servers[1].Integration.AssertCalled(t, "Register", mock.Anything, "test-1", int(p), int(p))
}

func TestDestroyLocalServiceRemovesRemoteListener(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond)

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	// check the listener is not accessible
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.Error(t, err)
}

func TestDestroyLocalServiceRemovesRemoteIntegration(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     "localhost:19001",
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	time.Sleep(100 * time.Millisecond)

	// check the listener exists
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.NoError(t, err)

	// remove the listener
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})
	time.Sleep(100 * time.Millisecond) // wait for setup

	servers[1].Integration.AssertCalled(t, "Deregister", "test-1")
}

func TestExposeLocalServiceCreatesRemoteConnection(t *testing.T) {
	t.Skip()
}

func TestExposeRemoteServiceCreatesRemoteConnection(t *testing.T) {
	t.Skip()
}

func TestMessageToRemoteEndpointCallsLocalService(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     tsAddr,
			Type:                shipyard.ServiceType_LOCAL,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	// call the remote endpoint
	httpResp, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d", p))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	httpResp, err = http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d", p))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)
}

func TestMessageToLocalEndpointCallsRemoteService(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     tsAddr,
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	// call the remote endpoint
	httpResp, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d", p))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	httpResp, err = http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d", p))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode)
}

func TestMessageToNonExistantEndpointRetrurnsError(t *testing.T) {
	c, _, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)
	p2 := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     fmt.Sprintf("localhost:%d", p2),
			Type:                shipyard.ServiceType_REMOTE,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)

	// wait while to ensure all setup
	time.Sleep(100 * time.Millisecond)

	// call the remote endpoint, should return an error
	_, err = http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d", p))
	require.Error(t, err)
}

func TestListServices(t *testing.T) {
	c, tsAddr, _ := setupTests(t)

	p := int32(rand.Intn(10000) + 30000)

	resp, err := c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test 1",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p,
			DestinationAddr:     tsAddr,
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
