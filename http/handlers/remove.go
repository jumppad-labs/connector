package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/protos/shipyard"
)

type Remove struct {
	client shipyard.RemoteConnectionClient
	logger hclog.Logger
}

// NewExpose creates a new Expose handler
func NewRemove(client shipyard.RemoteConnectionClient, l hclog.Logger) *Remove {
	return &Remove{client, l}
}

func (re *Remove) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	re.logger.Info("Delete exposed service", "id", id)

	_, err := re.client.DestroyService(context.Background(), &shipyard.DestroyRequest{Id: id})
	if err != nil {
		re.logger.Error("Unable to remove exposed service", "err", err)
		http.Error(rw, fmt.Sprintf("Unable to remove exposed service: %s", err), http.StatusInternalServerError)
		return
	}

}
