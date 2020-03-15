package consul

import "github.com/shipyard-run/connector/connectors"

type Consul struct {
	// consul api endpoint
}

func (c *Consul) Register(id string, name string, port int, matches []connectors.Match) error {
	return nil
}

func (c *Consul) Deregister(id string) error {
	return nil
}
