package nomad

import "github.com/hashicorp/go-hclog"

type Integration struct {
	log hclog.Logger
}

func New(log hclog.Logger) *Integration {
	return &Integration{log}
}

func (i *Integration) Register(id string, name string, srcPort, dstPort int) error {
	return nil
}

func (i *Integration) Deregister(id string) error {
	return nil
}

func (i *Integration) LookupAddress(service string) (string, error) {
	return service, nil
}
