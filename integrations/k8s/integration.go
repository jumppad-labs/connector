package k8s

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
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
}

// New creates a new Kubernetes integration
func New(log hclog.Logger, namespace string) *Integration {
	return &Integration{log, namespace}
}

// Register handles events when new services are exposed
func (i *Integration) Register(id string, name string, srcPort, dstPort int) error {
	clientset, err := i.createClient()
	if err != nil {
		i.log.Error("Unable to create Kubernetes client", "error", err)
		return err
	}

	// does the service already exist?
	svc, err := clientset.CoreV1().Services(i.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		i.log.Error("Unable to get services", "error", err)
		return err
	}

	// we should not find a service
	// Get return an empty service struct even when err != nil
	// nil error means a service has been found
	if err == nil {
		return fmt.Errorf("unable to create Kubernetes service, service already exists: %#v", svc)
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
		return fmt.Errorf("unable to create Kubernetes service: %s", err)
	}

	return nil
}

// Deregister a service in Kubernetes
func (i *Integration) Deregister(id string) error {
	clientset, err := i.createClient()
	if err != nil {
		i.log.Error("Unable to create Kubernetes client", "error", err)
		return err
	}

	err = clientset.CoreV1().Services(i.namespace).Delete(context.TODO(), id, metav1.DeleteOptions{})
	if err != nil {
		i.log.Error("Unable to remove Kubernetes service", "error", err)
		return err
	}

	return nil
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

func (i *Integration) LookupAddress(service string) (string, error) {
	return service, nil
}
