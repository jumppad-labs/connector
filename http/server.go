package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/http/handlers"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// LocalServer represents a local HTTP server
type LocalServer struct {
	logger      hclog.Logger
	apiAddress  string
	bindAddress string
	server      *gohttp.Server

	tlsCAPath   string
	tlsCertPath string
	tlsKeyPath  string
}

// NewLocalServer creates a new local HTTP server which can be used
// to expose gRPC server methods with JSON
func NewLocalServer(tlsCAPath, tlsCertPath, tlsKeyPath, apiAddress, bindAddr string, l hclog.Logger) *LocalServer {
	return &LocalServer{apiAddress: apiAddress, bindAddress: bindAddr, logger: l, tlsCAPath: tlsCAPath, tlsCertPath: tlsCertPath, tlsKeyPath: tlsKeyPath}
}

// Serve starts serving traffic
// Does not block
func (l *LocalServer) Serve() error {

	// add the handlers
	mux := l.createHandlers()

	// create the server and add handlers
	l.server = &gohttp.Server{Handler: mux}
	l.server.Addr = l.bindAddress

	// are we using TLS?
	if l.tlsCertPath != "" && l.tlsKeyPath != "" {
		l.logger.Info("Loading TLS Key", "path", l.tlsKeyPath)

		go func() {
			err := l.server.ListenAndServeTLS(l.tlsCertPath, l.tlsKeyPath)
			if err != nil {
				l.logger.Error("Unable to start server", "error", err)
			}
		}()

		return nil
	}

	go func() {
		err := l.server.ListenAndServe()
		if err != nil {
			l.logger.Error("Unable to start server", "error", err)
		}
	}()

	return nil
}

// Close all connections and shutdown the server
func (l *LocalServer) Close() error {
	return l.server.Close()
}

func (l *LocalServer) createHandlers() *mux.Router {
	r := mux.NewRouter()
	cli, _ := getRemoteClient(l.tlsCAPath, l.tlsCertPath, l.tlsKeyPath, l.apiAddress)

	// health handler
	hh := handlers.NewHealth(l.logger.Named("health_handler"))
	r.Handle("/health", hh).Methods(gohttp.MethodGet)

	eh := handlers.NewExpose(cli, l.logger.Named("expose_handler"))
	r.Handle("/expose", eh).Methods(gohttp.MethodPost)

	dh := handlers.NewRemove(cli, l.logger.Named("remove_handler"))
	r.Handle("/expose/{id}", dh).Methods(gohttp.MethodDelete)

	lh := handlers.NewList(cli, l.logger.Named("list_handler"))
	r.Handle("/list", lh).Methods(gohttp.MethodGet)

	return r
}

func getRemoteClient(tlsCAPath, tlsCertPath, tlsKeyPath, uri string) (shipyard.RemoteConnectionClient, error) {
	if tlsCAPath != "" && tlsCertPath != "" && tlsKeyPath != "" {
		// if we are using TLS create a TLS client
		certificate, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err != nil {
			return nil, err
		}

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
			ServerName:   uri,
			Certificates: []tls.Certificate{certificate},
			RootCAs:      certPool,
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
