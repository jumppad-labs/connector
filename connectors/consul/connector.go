package consul

type Consul struct {
	// consul api endpoint
}

func (c *Consul) Register(id string, name string, port int, matches []Match) error {}

func (c *Consul) Deregister(id string) error {}
