package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/integrations"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Integration handles the integration between the connector and the local platform
type Integration struct {
	// k8s api endpoint
	log       hclog.Logger
	namespace string
	cache     map[string]cacheItem
}

type cacheItem struct {
	ServiceType string
	Component   string
	Name        string
	Address     string
	Port        int
}

// New creates a new Kubernetes integration
func New(log hclog.Logger, namespace string) *Integration {
	return &Integration{log, namespace, map[string]cacheItem{}}
}

// Register handles events when new services are exposed
//
// if we are creating a local service on the remote component, we need
// to create a kubernetes service that points at a tcp port which will
// route traffic over the stream to the local connector

// if we are creating a local service on the local component, we need to
// just set the address of the local service being exposed

// if we are creating a remote service on the remote component, we need to
// set the address of the local service being exposed

// if we are creating a remote service on the local component, we need to
// set the address to the location of the tcp listener
func (i *Integration) Register(id, serviceType, component string, config map[string]string) (*integrations.ServiceDetails, error) {
	if component == "REMOTE" && serviceType == "LOCAL" {
		return i.createLocalIntegration(id, config)
	}

	if (component == "REMOTE" && serviceType == "REMOTE") || (component == "LOCAL" && serviceType == "LOCAL") {
		addr, port, err := getAddressFromConfig(config)
		if err != nil {
			return nil, err
		}

		ci := cacheItem{
			Component:   component,
			ServiceType: serviceType,
			Address:     addr,
			Port:        port,
		}

		i.cache[id] = ci

		return &integrations.ServiceDetails{Address: ci.Address, Port: ci.Port}, nil
	}

	if component == "LOCAL" && serviceType == "REMOTE" {
		p, err := getPortFromConfig(config)
		if err != nil {
			return nil, err
		}

		ci := cacheItem{
			Component:   component,
			ServiceType: serviceType,
			Address:     "localhost",
			Port:        p,
		}

		i.cache[id] = ci

		return &integrations.ServiceDetails{Address: ci.Address, Port: ci.Port}, nil
	}

	return nil, fmt.Errorf("invalid serviceType %s and component %s", serviceType, component)
}

// Deregister a service in Kubernetes
func (i *Integration) Deregister(id string) error {
	name := i.cache[id].Name

	clientset, err := i.createClient()
	if err != nil {
		i.log.Error("Unable to create Kubernetes client", "error", err)
		return err
	}

	err = clientset.CoreV1().Services(i.namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		i.log.Error("Unable to remove Kubernetes service", "error", err)
		return err
	}

	delete(i.cache, id)

	return nil
}

func (i *Integration) LookupAddress(id string) (string, error) {
	addr := i.cache[id].Address
	port := i.cache[id].Port

	return fmt.Sprintf("%s:%d", addr, port), nil
}

func (i *Integration) GetDetails(id string) (map[string]string, error) {
	addr, err := i.LookupAddress(id)
	if err != nil {
		return nil, err
	}

	return map[string]string{"address": addr}, nil
}

func (i *Integration) createLocalIntegration(id string, config map[string]string) (*integrations.ServiceDetails, error) {
	clientset, err := i.createClient()
	if err != nil {
		i.log.Error("Unable to create Kubernetes client", "error", err)
		return nil, err
	}

	name := config["name"]
	dstPort, _ := getPortFromConfig(config)

	ci := cacheItem{
		Component:   "REMOTE",
		ServiceType: "LOCAL",
		Name:        name,
		Address:     fmt.Sprintf("%s.%s.svc", name, i.namespace),
		Port:        dstPort,
	}

	i.cache[id] = ci

	// does the service already exist?
	svc, err := clientset.CoreV1().Services(i.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		i.log.Error("Unable to get services", "error", err)
		return nil, err
	}

	// we should not find a service
	// Get return an empty service struct even when err != nil
	// nil error means a service has been found
	if err == nil {
		return nil, fmt.Errorf("unable to create Kubernetes service, service already exists: %#v", svc)
	}

	svc = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": "connector"},
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Protocol: "TCP",
					Port:     int32(dstPort),
				},
			},
		},
	}

	// create the service
	_, err = clientset.CoreV1().Services(i.namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes service: %s", err)
	}

	return &integrations.ServiceDetails{
		Address: ci.Address,
		Port:    dstPort,
	}, nil
}

func (i *Integration) createClient() (*kubernetes.Clientset, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		i.log.Error("Unable to read kubernetes cluster config", "error", err)
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		i.log.Error("Unable to kubernetes clientset", "error", err)
		return nil, err
	}

	return clientset, nil
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

func getAddressFromConfig(config map[string]string) (string, int, error) {
	if config == nil || config["address"] == "" {
		return "", -1, fmt.Errorf(`"address", missing from configuration`)
	}

	parts := strings.Split(config["address"], ":")

	p := parts[1]
	ip, err := strconv.Atoi(p)
	if err != nil {
		return "", -1, fmt.Errorf(`"port" must be a number`)
	}

	return parts[0], ip, nil
}
