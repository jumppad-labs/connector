package http

import (
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/http/handlers"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
)

// LocalServer represents a local HTTP server
type LocalServer struct {
	logger      hclog.Logger
	apiAddress  string
	bindAddress string
	server      *gohttp.Server
}

// NewLocalServer creates a new local HTTP server which can be used
// to expose gRPC server methods with JSON
func NewLocalServer(apiAddress, bindAddr string, l hclog.Logger) *LocalServer {
	return &LocalServer{apiAddress: apiAddress, bindAddress: bindAddr, logger: l}
}

// Serve starts serving traffic
// Does not block
func (l *LocalServer) Serve() error {

	// add the handlers
	mux := l.createHandlers()

	// create the server and add handlers
	l.server = &gohttp.Server{Handler: mux}
	l.server.Addr = l.bindAddress

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

	// health handler
	hh := handlers.NewHealth(l.logger.Named("health_handler"))
	r.Handle("/health", hh).Methods(gohttp.MethodGet)

	cli, _ := getRemoteClient(l.apiAddress)
	eh := handlers.NewExpose(cli, l.logger.Named("expose_handler"))
	r.Handle("/expose", eh).Methods(gohttp.MethodPost)

	return r
}

func getRemoteClient(uri string) (shipyard.RemoteConnectionClient, error) {
	conn, err := grpc.Dial(uri, grpc.WithInsecure(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	rc := shipyard.NewRemoteConnectionClient(conn)

	return rc, nil
}
