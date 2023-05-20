package local

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/integrations"
)

type Integration struct {
	log   hclog.Logger
	cache map[string]*integrations.ServiceDetails
}

func New(log hclog.Logger) *Integration {
	return &Integration{log, map[string]*integrations.ServiceDetails{}}
}

func (i *Integration) Register(id, serviceType, component string, config map[string]string) (*integrations.ServiceDetails, error) {
	// if this is the local part we use the address field in the local
	// config as this is the local service we need to send traffic to
	if component == "LOCAL" {
		addr, err := getAddressFromConfig(config)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(addr, ":")
		p, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}

		i.cache[id] = &integrations.ServiceDetails{Address: parts[0], Port: p}
		return i.cache[id], err
	}

	// if this is the remote part, we build the address that will form the
	// local listener that will eventually route data over the remote stream
	p, err := getPortFromConfig(config)
	i.cache[id] = &integrations.ServiceDetails{Address: "localhost", Port: p}

	return i.cache[id], err
}

func (i *Integration) Deregister(id string) error {
	delete(i.cache, id)
	return nil
}

func (i *Integration) LookupAddress(id string) (string, error) {
	cache, ok := i.cache[id]
	if !ok {
		return "", fmt.Errorf("unable to find address for id: %s", id)
	}

	port := cache.Port

	return fmt.Sprintf("localhost:%d", port), nil
}

func (i *Integration) GetDetails(id string) (map[string]string, error) {
	addr, err := i.LookupAddress(id)
	if err != nil {
		return nil, err
	}

	return map[string]string{"address": addr}, nil
}

func getPortFromConfig(config map[string]string) (int, error) {
	if config == nil || config["port"] == "" {
		return -1, fmt.Errorf(`"port", missing from configuration`)
	}

	p := config["port"]
	ip, err := strconv.Atoi(p)
	if err != nil {
		return -1, fmt.Errorf(`"port" must be a number`)
	}

	return ip, nil
}

func getAddressFromConfig(config map[string]string) (string, error) {
	if config != nil && config["address"] != "" {
		return config["address"], nil
	}

	return "", fmt.Errorf(`"address", missing from configuration`)
}
