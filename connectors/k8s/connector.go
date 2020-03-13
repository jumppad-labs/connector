package k8s

type K8s struct {
	// k8s api endpoint
}

func (c *K8s) Register(id string, name string, port int, matches []Match) error {}

func (c *K8s) Deregister(id string) error {}
