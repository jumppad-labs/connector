package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/local/handlers"
)

// LocalServer represents a local HTTP server
type LocalServer struct {
	logger     hclog.Logger
	apiAddress string
	server     *http.Server
}

// NewLocalServer creates a new local server
func NewLocalServer(apiAddress string, l hclog.Logger) *LocalServer {
	return &LocalServer{apiAddress: apiAddress, logger: l}
}

// Serve starts serving traffic
func (l *LocalServer) Serve() error {

	// add the handlers
	mux := createHandlers(l.logger)

	// create the server and add handlers
	l.server = &http.Server{Handler: mux}
	l.server.Addr = l.apiAddress

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

func createHandlers(l hclog.Logger) *mux.Router {
	r := mux.NewRouter()

	// health handler
	hh := handlers.NewHealth(l.Named("health_handler"))
	r.Handle("/health", hh).Methods(http.MethodGet)

	return r
}
