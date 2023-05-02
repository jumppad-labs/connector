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
	Register(id string, name string, srcPort, dstPort int) error
	// Deregister a new service with the integration, this is used when exposing a local
	// application to a remote cluster
	Deregister(id string) error
	// LookupAddress, allows a service name to be resolved to a physical address
	// where the service name is already addressable (i.e. kubernetes, or local)
	// this method should just return the original service
	LookupAddress(service string) (string, error)
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
