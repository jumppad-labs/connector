package connectors


type Match {
	// Change these...
	HTTP string
	GRPC string
}
// Connector defines the base interface which implementations like Consul or Istio implement
type Connector interface {
	Register(id string, name string, port int, matches []Match) error
	Deregister(id string) error
}
