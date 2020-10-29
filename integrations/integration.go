package integrations

import (
	"log"
	"regexp"
	"strings"
)

// Integration defines the base interface which implementations like Consul or Istio implement
type Integration interface {
	Register(id string, name string, srcPort, dstPort int) error
	Deregister(id string) error
}

// SanitizeName takes a string and returns a URI acceptble name
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
