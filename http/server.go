package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/http/handlers"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// LocalServer represents a local HTTP server
type LocalServer struct {
	listener net.Listener
	logger   hclog.Logger
	server   *http.Server
}

// NewLocalServer creates a new local HTTP server which can be used
// to expose gRPC server methods with JSON
func NewLocalServer(l net.Listener, log hclog.Logger) *LocalServer {
	return &LocalServer{listener: l, logger: log}
}

// Serve starts serving traffic
func (l *LocalServer) Serve() error {

	// add the handlers
	mux := l.createHandlers(l.listener.Addr().String())

	// create the server and add handlers
	l.server = &http.Server{
		Handler:           mux,
		Addr:              l.listener.Addr().String(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 0 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ErrorLog:          l.logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true}),
	}

	return l.server.Serve(l.listener)
}

// Close all connections and shutdown the server
func (l *LocalServer) Close() error {
	return l.server.Close()
}

func (l *LocalServer) createHandlers(addr string) *mux.Router {
	r := mux.NewRouter()
	cli, _ := getRemoteClient("", addr)

	// health handler
	hh := handlers.NewHealth(l.logger.Named("health_handler"))
	r.Handle("/health", hh).Methods(http.MethodGet)

	eh := handlers.NewExpose(cli, l.logger.Named("expose_handler"))
	r.Handle("/expose", eh).Methods(http.MethodPost)

	dh := handlers.NewRemove(cli, l.logger.Named("remove_handler"))
	r.Handle("/expose/{id}", dh).Methods(http.MethodDelete)

	lh := handlers.NewList(cli, l.logger.Named("list_handler"))
	r.Handle("/list", lh).Methods(http.MethodGet)

	//ch := handlers.NewGenerateCertificate(l.logger.Named("certificate_handler"), l.tlsCAPath, l.tlsCAKeyPath)
	//r.Handle("/certificate", ch).Methods(http.MethodPost)

	return r
}

func getRemoteClient(tlsCAPath, uri string) (shipyard.RemoteConnectionClient, error) {
	if tlsCAPath != "" {
		// Create a certificate pool from the certificate authority
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(tlsCAPath)
		if err != nil {
			return nil, err
		}

		ok := certPool.AppendCertsFromPEM(ca)
		if !ok {
			return nil, fmt.Errorf("unable to append certs from ca pem")
		}

		creds := credentials.NewTLS(&tls.Config{
			RootCAs: certPool,
		})

		// Create a connection with the TLS credentials
		conn, err := grpc.Dial(uri, grpc.WithTransportCredentials(creds))
		if err != nil {
			return nil, err
		}
		rc := shipyard.NewRemoteConnectionClient(conn)

		return rc, nil
	}

	conn, err := grpc.Dial(uri, grpc.WithInsecure(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	rc := shipyard.NewRemoteConnectionClient(conn)

	return rc, nil
}
