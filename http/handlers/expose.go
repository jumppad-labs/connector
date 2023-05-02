package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/protos/shipyard"
)

// Expose handler is responsible for exposing new endpoints
type Expose struct {
	client shipyard.RemoteConnectionClient
	logger hclog.Logger
}

// NewExpose creates a new Expose handler
func NewExpose(client shipyard.RemoteConnectionClient, l hclog.Logger) *Expose {
	return &Expose{client, l}
}

// ExposeRequest is the JSON request for the Create handler
type ExposeRequest struct {
	Name                string `json:"name" validate:"required"`
	SourcePort          int    `json:"source_port" validate:"required"`
	RemoteConnectorAddr string `json:"remote_connector_addr" validate:"required"`
	DestinationAddr     string `json:"destination_addr" validate:"required"`
	Type                string `json:"type" validate:"oneof=local remote"`
}

// Validate the struct and return an error if invalid
func (c *ExposeRequest) Validate() error {
	validate := validator.New()

	return validate.Struct(c)
}

func decodeJSON(r io.Reader, dest interface{}) error {
	dec := json.NewDecoder(r)

	return dec.Decode(dest)
}

// ServeHTTP implements the http.Handler interface
func (c *Expose) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	c.logger.Info("Handle Expose")

	cr := &ExposeRequest{}

	// get the request
	err := decodeJSON(r.Body, cr)
	if err != nil {
		c.logger.Error("Unable to decode JSON", "error", err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	err = cr.Validate()
	if err != nil {
		c.logger.Error("Failed validation", "error", err)
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// first get a client
	c.logger.Info("Sending request to the local gRPC server")
	t := shipyard.ServiceType_LOCAL
	if cr.Type == "remote" {
		t = shipyard.ServiceType_REMOTE
	}

	// Call the grpc upstream
	resp, err := c.client.ExposeService(context.Background(), &shipyard.ExposeRequest{
		Service: &shipyard.Service{
			Name:                cr.Name,
			RemoteConnectorAddr: cr.RemoteConnectorAddr,
			DestinationAddr:     cr.DestinationAddr,
			SourcePort:          int32(cr.SourcePort),
			Type:                t,
		},
	})

	if err != nil {
		c.logger.Error("Unable to expose service", "error", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// write the expose id
	fmt.Fprint(rw, resp.Id)
}
