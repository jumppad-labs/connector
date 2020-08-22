package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
	"google.golang.org/grpc"
)

// Create handler is responsible for exposing new endpoints
type Create struct {
	logger hclog.Logger
}

// NewCreate creates a new Create handler
func NewCreate(l hclog.Logger) *Create {
	return &Create{l}
}

// CreateRequest is the JSON request for the Create handler
type CreateRequest struct {
	Port           int    `json:"port" validate:"required"`
	Service        string `json:"service" validate:"required"`
	RemoteLocation string `json:"remote_location" validate:"required,url"`
}

// Validate the struct and return an error if invalid
func (c *CreateRequest) Validate() error {
	validate := validator.New()

	return validate.Struct(c)
}

func decodeJSON(r io.Reader, dest interface{}) error {
	dec := json.NewDecoder(r)

	return dec.Decode(dest)
}

// ServeHTTP implements the http.Handler interface
func (c *Create) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	cr := &CreateRequest{}

	// get the request
	err := decodeJSON(r.Body, cr)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	err = cr.Validate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// create a new connection

	// first get a client
	c.logger.Debug("Connecting to remote server", "remote", cr.RemoteLocation)
	_, err = c.getRemoteClient(cr.RemoteLocation)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// create the

	// start a new server on this port
	mr := mux.NewRouter()
	mr.HandleFunc(
		"/",
		func(rw http.ResponseWriter, r *http.Request) {
			c.logger.Debug("Got Downstream")
		},
	)

	// start the server
	go http.ListenAndServe(fmt.Sprintf(":%d", cr.Port), mr)
}

func (c *Create) getRemoteClient(uri string) (shipyard.RemoteConnectionClient, error) {
	conn, err := grpc.Dial(uri, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions())
	if err != nil {
		return nil, err
	}

	rc := shipyard.NewRemoteConnectionClient(conn)

	return rc, nil
}
