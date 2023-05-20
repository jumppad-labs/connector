package integrations

import (
	"log"
	"regexp"
	"strings"
)

// Integration defines the base interface which implementations like Consul or Istio implement
type Integration interface {
	// Register a new service with the integration, this is used when exposing a local
	// application to a remote cluster
	//
	// serviceType is the type of service being exposed LOCAL, REMOTE
	// component is the LOCAL, or REMOTE part of the connector
	// config is a simple map, the fields for this map vary based on the integration
	//
	// serviceType and component have a baring on the integration
	// For example, registering a LOCAL service on the LOCAL component requires only the
	// address of the upstream application to be set
	// however, registering a LOCAL service on the REMOTE component requires a listener to be
	// created that will route traffic over the stream to the local component
	Register(id, serviceType, component string, config map[string]string) (*ServiceDetails, error)
	// Deregister a new service with the integration, this is used when exposing a local
	// application to a remote cluster
	Deregister(id string) error
	// LookupAddress, allows a service name to be resolved to a physical address
	// where the service name is already addressable (i.e. kubernetes, or local)
	// this method should just return the original service
	LookupAddress(id string) (string, error)
	// GetDetails related to the configured integration, this is specific for each
	// integration plugin
	GetDetails(id string) (map[string]string, error)
}

type ServiceDetails struct {
	Address string
	Port    int
}

// SanitizeName takes a string and returns a URI acceptable name
// e.g. Test Service would become test-service
func SanitizeName(original string) string {
	// first convert to lower case
	original = strings.ToLower(original)

	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	return reg.ReplaceAllString(original, "-")
}
