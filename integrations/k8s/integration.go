package k8s

import (
	"context"
	"fmt"
	"strconv"

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
	Name string
	Port int
}

// New creates a new Kubernetes integration
func New(log hclog.Logger, namespace string) *Integration {
	return &Integration{log, namespace, map[string]cacheItem{}}
}

// Register handles events when new services are exposed
func (i *Integration) Register(id string, direction string, config map[string]string) (*integrations.ServiceDetails, error) {
	clientset, err := i.createClient()
	if err != nil {
		i.log.Error("Unable to create Kubernetes client", "error", err)
		return nil, err
	}

	name := config["name"]
	dstPort, _ := strconv.Atoi(config["port"])

	i.cache[id] = cacheItem{
		Name: name,
		Port: dstPort,
	}

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
		Address: fmt.Sprintf("%s.%s.svc", name, i.namespace),
		Port:    dstPort,
	}, nil
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
	name := i.cache[id].Name
	port := i.cache[id].Port

	return fmt.Sprintf("%s.%s.svc:%d", name, i.namespace, port), nil
}

func (i *Integration) GetDetails(id string) (map[string]string, error) {
	addr, err := i.LookupAddress(id)
	if err != nil {
		return nil, err
	}

	return map[string]string{"address": addr}, nil
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
