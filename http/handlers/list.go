package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/protos/shipyard"
)

type List struct {
	client shipyard.RemoteConnectionClient
	logger hclog.Logger
}

type Service struct {
	ID                  string            `json:"id" validate:"required"`
	RemoteConnectorAddr string            `json:"remote_connector_addr" validate:"required"`
	Type                string            `json:"type" validate:"oneof=local remote"`
	Status              string            `json:"status"`
	Config              map[string]string `json:"config"`
	Details             map[string]string `json:"details"`
}

// NewExpose creates a new Expose handler
func NewList(client shipyard.RemoteConnectionClient, l hclog.Logger) *List {
	return &List{client, l}
}

func (l *List) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	l.logger.Info("Listing services")

	svcs, err := l.client.ListServices(context.Background(), &shipyard.NullMessage{})
	if err != nil {
		http.Error(rw, fmt.Sprintf("Unable to list services: %s", svcs), http.StatusInternalServerError)
		return
	}

	services := []Service{}
	for _, v := range svcs.Services {
		s := Service{
			ID:                  v.Id,
			RemoteConnectorAddr: v.RemoteConnectorAddr,
			Type:                v.Type.String(),
			Status:              v.Status.String(),
			Config:              v.Config,
			Details:             v.Details,
		}

		services = append(services, s)
	}

	je := json.NewEncoder(rw)
	je.Encode(services)
}
