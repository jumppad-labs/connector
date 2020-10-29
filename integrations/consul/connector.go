package consul

type Consul struct {
	// consul api endpoint
}

func (c *Consul) Register(id string, name string, port int) error {
	return nil
}

func (c *Consul) Deregister(id string) error {
	return nil
}
