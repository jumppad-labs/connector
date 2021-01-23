package remote

import (
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

func createServer(t *testing.T, addr, name string) (*Server, *integrations.Mock, func()) {
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

	output := ioutil.Discard

	if os.Getenv("DEBUG") == "true" {
		output = os.Stdout
	}

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

	cleanup := func() {
		s.Shutdown()
		grpcServer.Stop()
		lis.Close()
	}

	t.Cleanup(func() {
		cleanup()
	})

	return s, mi, cleanup
}

func startLocalServer(t *testing.T) (string, *string) {
	bodyData := ""

	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Trace, Name: "http_server"})

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		l.Debug("Got request")

		data, _ := ioutil.ReadAll(r.Body)
		bodyData = string(data)

		// write response
		rw.Write([]byte(SevenKResponse))

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
	Cleanup     func()
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
	s1, m1, c1 := createServer(t, a1, "server_local_1")
	s2, m2, c2 := createServer(t, a2, "server_local_2")
	s3, m3, c3 := createServer(t, a3, "server_local_3")

	servers = []serverStruct{
		{s1, p1, a1, m1, c1},
		{s2, p2, a2, m2, c2},
		{s3, p3, a3, m3, c3},
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

func TestReconfigureRemoteServiceUpdatesListener(t *testing.T) {
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

	require.Eventually(t,
		func() bool {
			_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
			return err == nil
		},
		1*time.Second,
		50*time.Millisecond,
	)

	// remove the service
	c.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: resp.Id})

	require.Eventually(t,
		func() bool {
			s, _ := c.ListServices(context.Background(), &shipyard.NullMessage{})
			if len(s.Services) == 0 {
				return true
			}

			return false
		},
		1*time.Second,
		50*time.Millisecond,
	)

	// reconfigure
	resp, err = c.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                "Test Service",
			RemoteConnectorAddr: servers[1].Address,
			SourcePort:          p + 1,
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

	require.Eventually(t,
		func() bool {
			_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p+1))
			return err == nil
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

func TestDisconnectRemovesRemoteListenerThenReconnects(t *testing.T) {
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

	// shutdown the remote and remove listeners
	servers[1].Cleanup()

	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	require.Error(t, err)

	// restart the service
	createServer(t, servers[1].Address, "server_local_2")

	// connection should be re-established when the remote restarts
	require.Eventually(t, func() bool {
		// check the listener exists
		_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
		return err == nil
	},
		3*time.Second,
		50*time.Millisecond,
	)
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

var SevenKResponse = `
He was an old man who fished alone in a skiff in the Gulf Stream and he had gone eighty-four days now without taking a fish. In the first forty days a boy had been with him. But after forty days without a fish the boy's parents had told him that the old man was now definitely and finally salao, which is the worst form of unlucky, and the boy had gone at their orders in another boat which caught three good fish the first week. It made the boy sad to see the old man come in each day with his skiff empty and he always went down to help him carry either the coiled lines or the gaff and harpoon and the sail that was furled around the mast. The sail was patched with flour sacks and, furled, it looked like the flag of permanent defeat.

The old man was thin and gaunt with deep wrinkles in the back of his neck. The brown blotches of the benevolent skin cancer the sun brings from its reflection on the tropic sea were on his cheeks. The blotches ran well down the sides of his face and his hands had the deep-creased scars from handling heavy fish on the cords. But none of these scars were fresh. They were as old as erosions in a fishless desert.

Everything about him was old except his eyes and they were the same color as the sea and were cheerful and undefeated.

"Santiago," the boy said to him as they climbed the bank from where the skiff was hauled up. "I could go with you again. We've made some money."

The old man had taught the boy to fish and the boy loved him.

"No," the old man said. "You're with a lucky boat. Stay with them."

"But remember how you went eighty-seven days without fish and then we caught big ones every day for three weeks."

"I remember," the old man said. "I know you did not leave me because you doubted."

"It was papa made me leave. I am a boy and I must obey him."

"I know," the old man said. "It is quite normal."

"He hasn't much faith."

"No," the old man said. "But we have. Haven't we?"

"Yes," the boy said. "Can I offer you a beer on the Terrace and then we'll take the stuff home."

"Why not?" the old man said. "Between fishermen."

They sat on the Terrace and many of the fishermen made fun of the old man and he was not angry. Others, of the older fishermen, looked at him and were sad. But they did not show it and they spoke politely about the current and the depths they had drifted their lines at and the steady good weather and of what they had seen. The successful fishermen of that day were already in and had butchered their marlin out and carried them laid full length across two planks, with two men staggering at the end of each plank, to the fish house where they waited for the ice truck to carry them to the market in Havana. Those who had caught sharks had taken them to the shark factory on the other side of the cove where they were hoisted on a block and tackle, their livers removed, their fins cut off and their hides skinned out and their flesh cut into strips for salting.

When the wind was in the east a smell came across the harbour from the shark factory; but today there was only the faint edge of the odour because the wind had backed into the north and then dropped off and it was pleasant and sunny on the Terrace.

"Santiago," the boy said.

"Yes," the old man said. He was holding his glass and thinking of many years ago.

"Can I go out to get sardines for you for tomorrow?"

"No. Go and play baseball. I can still row and Rogelio will throw the net."

"I would like to go. If I cannot fish with you, I would like to serve in some way."

"You bought me a beer," the old man said. "You are already a man."

"How old was I when you first took me in a boat?"

"Five and you nearly were killed when I brought the fish in too green and he nearly tore the boat to pieces. Can you remember?"

"I can remember the tail slapping and banging and the thwart breaking and the noise of the clubbing. I can remember you throwing me into the bow where the wet coiled lines were and feeling the whole boat shiver and the noise of you clubbing him like chopping a tree down and the sweet blood smell all over me."

"Can you really remember that or did I just tell it to you?"

"I remember everything from when we first went together."

The old man looked at him with his sun-burned, confident loving eyes.

"If you were my boy I'd take you out and gamble," he said. "But you are your father's and your mother's and you are in a lucky boat."

"May I get the sardines? I know where I can get four baits too."

"I have mine left from today. I put them in salt in the box."

"Let me get four fresh ones."

"One," the old man said. His hope and his confidence had never gone. But now they were freshening as when the breeze rises.

"Two," the boy said.

"Two," the old man agreed. "You didn't steal them?"

"I would," the boy said. "But I bought these."

"Thank you," the old man said. He was too simple to wonder when he had attained humility. But he knew he had attained it and he knew it was not disgraceful and it carried no loss of true pride.

"Tomorrow is going to be a good day with this current," he said.

"Where are you going?" the boy asked.

"Far out to come in when the wind shifts. I want to be out before it is light."

"I'll try to get him to work far out," the boy said. "Then if you hook something truly big we can come to your aid."

"He does not like to work too far out."

"No," the boy said. "But I will see something that he cannot see such as a bird working and get him to come out after dolphin."

"Are his eyes that bad?"

"He is almost blind."

"It is strange," the old man said. "He never went turtle-ing. That is what kills the eyes."

"But you went turtle-ing for years off the Mosquito Coast and your eyes are good."

"I am a strange old man."

"But are you strong enough now for a truly big fish?"

"I think so. And there are many tricks."

"Let us take the stuff home," the boy said. "So I can get the cast net and go after the sardines."

They picked up the gear from the boat. The old man carried the mast on his shoulder and the boy carried the wooden box with the coiled, hard-braided brown lines, the gaff and the harpoon with its shaft. The box with the baits was under the stern of the skiff along with the club that was used to subdue the big fish when they were brought alongside. No one would steal from the old man but it was better to take the sail and the heavy lines home as the dew was bad for them and, though he was quite sure no local people would steal from him, the old man thought that a gaff and a harpoon were needless temptations to leave in a boat.

They walked up the road together to the old man's shack and went in through its open door. The old man leaned the mast with its wrapped sail against the wall and the boy put the box and the other gear beside it. The mast was nearly as long as the one room of the shack. The shack was made of the tough bud-shields of the royal palm which are called guano and in it there was a bed, a table, one chair, and a place on the dirt floor to cook with charcoal. On the brown walls of the flattened, overlapping leaves of the sturdy fibered guano there was a picture in color of the Sacred Heart of Jesus and another of the Virgin of Cobre. These were relics of his wife. Once there had been a tinted photograph of his wife on the wall but he had taken it down because it made him too lonely to see it and it was on the shelf in the corner under his clean shirt.
`
