package k8s

import "github.com/shipyard-run/connector/connectors"

type K8s struct {
	// k8s api endpoint
}

func (c *K8s) Register(id string, name string, port int, matches []connectors.Match) error {
	return nil
}

func (c *K8s) Deregister(id string) error {
	return nil
}
