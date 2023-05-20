package nomad

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/integrations"
	"golang.org/x/xerrors"
)

type Integration struct {
	log   hclog.Logger
	cache map[string]cacheItem
}

type cacheItem struct {
	ServiceType string
	Component   string
	Job         string
	Group       string
	Task        string
	TaskPort    string
	Address     string
	Port        int
}

func New(log hclog.Logger) *Integration {
	return &Integration{log, map[string]cacheItem{}}
}

// Register handles events when new services are exposed
//
// if we are creating a local service on the remote component, we need
// to create an address that points at a tcp port which will
// route traffic over the stream to the local connector

// if we are creating a local service on the local component, we need to
// just set the address of the local service being exposed

// if we are creating a remote service on the remote component, we need to
// set the address of the local service being exposed

// if we are creating a remote service on the local component, we need to
// set the address to the location of the tcp listener
func (i *Integration) Register(id, serviceType, component string, c map[string]string) (*integrations.ServiceDetails, error) {
	port, err := getPortFromConfig(c)
	if err != nil {
		return nil, err
	}

	ci := cacheItem{
		Port:        port,
		ServiceType: serviceType,
		Component:   component,
	}

	if serviceType == "REMOTE" {
		ci2, err := getDetailsFromConfig(c)
		if err != nil {
			return nil, err
		}

		ci2.Port = ci.Port
		ci2.ServiceType = ci.ServiceType
		ci2.Component = ci.Component

		i.cache[id] = *ci2

	}

	addr, p, err := getAddressFromConfig(c)
	if err != nil {
		return nil, err
	}

	ci.Address = fmt.Sprintf("%s:%d", addr, p)

	i.cache[id] = ci

	addr, err = i.LookupAddress(id)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(addr, ":")
	p, _ = strconv.Atoi(parts[1])

	return &integrations.ServiceDetails{
		Address: parts[0],
		Port:    p,
	}, nil
}

func (i *Integration) Deregister(id string) error {
	delete(i.cache, id)
	return nil
}

func (i *Integration) LookupAddress(id string) (string, error) {
	i.log.Debug("Attempting to resolve host address for", "service", id)

	ci := i.cache[id]

	// return the address of the local service
	if ci.Component == "LOCAL" && ci.ServiceType == "LOCAL" {
		return ci.Address, nil
	}

	// return the address of the local listener
	if ci.Component == "LOCAL" && ci.ServiceType == "REMOTE" {
		return fmt.Sprintf("localhost:%d", ci.Port), nil
	}

	// if a remote service get the address of the connector
	if ci.Component == "REMOTE" && ci.ServiceType == "LOCAL" {
		ci.Job = "connector"
		ci.Group = "connector"
		ci.Task = "connector"
		ci.TaskPort = "http"
	}

	// return the address of the nomad task
	eps, err := i.jobEndpoints(ci.Job, ci.Group, ci.Task)
	if err != nil {
		return "", fmt.Errorf("unable to find endpoint for service %s, error: %s", id, err)
	}

	if len(eps) < 1 {
		return "", fmt.Errorf("unable to find endpoint for service %s", id)
	}

	// choose a random endpoint
	ei := rand.Intn(len(eps))
	ep := eps[ei]

	// get the endpoint for the port
	p, ok := ep[ci.TaskPort]
	if !ok {
		return "", fmt.Errorf("unable to find port %s in endpoints for service %s", ci.TaskPort, id)
	}

	return p, nil
}

func (i *Integration) GetDetails(id string) (map[string]string, error) {
	addr, err := i.LookupAddress(id)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"address": addr,
	}, nil
}

func (i *Integration) jobEndpoints(job, group, task string) ([]map[string]string, error) {
	// check we have a valid Nomad server address
	httpAddr := os.Getenv("NOMAD_ADDR")
	if httpAddr == "" {
		return nil, fmt.Errorf("unable to create nomad client NOMAD_ADD environment not set")
	}

	jobs, err := i.getJobAllocations(job)
	if err != nil {
		return nil, err
	}

	i.log.Trace("got job allocations", "allocs", jobs)

	endpoints := []map[string]string{}

	// get the allocation details for each endpoint
	for _, j := range jobs {
		// only find running jobs
		if j["ClientStatus"].(string) != "running" {
			continue
		}

		r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/allocation/%s", httpAddr, j["ID"]), nil)
		if err != nil {
			return nil, xerrors.Errorf("unable to create http request: %w", err)
		}

		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			return nil, xerrors.Errorf("unable to get allocation: %w", err)
		}

		if resp.Body == nil {
			return nil, xerrors.Errorf("no body returned from Nomad API")
		}

		defer resp.Body.Close()

		allocDetail := allocation{}
		err = json.NewDecoder(resp.Body).Decode(&allocDetail)
		if err != nil {
			return nil, fmt.Errorf("error getting endpoints from server: %s: err: %s", httpAddr, err)
		}

		ports := []string{}

		// find the ports used by the task
		for _, tg := range allocDetail.Job.TaskGroups {
			if tg.Name == group {
				// non connect services will have their ports
				// coded in the driver config block
				for _, t := range tg.Tasks {
					if t.Name == task {
						ports = append(ports, t.Config.Ports...)
					}
				}

				// connect services will have this coded
				// in the groups network block
				for _, n := range tg.Networks {
					for _, dp := range n.DynamicPorts {
						ports = append(ports, dp.Label)
					}

					for _, dp := range n.ReservedPorts {
						ports = append(ports, dp.Label)
					}
				}
			}
		}

		ep := map[string]string{}
		epc := 0
		for _, p := range ports {
			// lookup the resources for the ports
			for _, n := range allocDetail.Resources.Networks {
				for _, dp := range n.DynamicPorts {
					if dp.Label == p {
						ep[p] = fmt.Sprintf("%s:%d", n.IP, dp.Value)
						epc++
					}
				}

				for _, dp := range n.ReservedPorts {
					if dp.Label == p {
						ep[p] = fmt.Sprintf("%s:%d", n.IP, dp.Value)
						epc++
					}
				}
			}
		}

		if epc > 0 {
			endpoints = append(endpoints, ep)
		}
	}

	i.log.Trace("found endpoints", "endpoints", endpoints)

	return endpoints, nil
}

func (i *Integration) getJobAllocations(job string) ([]map[string]interface{}, error) {
	httpAddr := os.Getenv("NOMAD_ADDR")

	// get the allocations for the job
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/job/%s/allocations", httpAddr, job), nil)
	if err != nil {
		return nil, xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("Unable to query job: %w", err)
	}

	if resp.Body == nil {
		return nil, xerrors.Errorf("No body returned from Nomad API")
	}

	defer resp.Body.Close()

	jobDetail := make([]map[string]interface{}, 0)
	err = json.NewDecoder(resp.Body).Decode(&jobDetail)
	if err != nil {
		return nil, fmt.Errorf("Unable to query jobs in Nomad server: %s: %s", httpAddr, err)
	}

	return jobDetail, err
}

type allocation struct {
	ID        string
	Job       job
	Resources resource
}

type job struct {
	Name       string
	TaskGroups []taskGroup
}

type taskGroup struct {
	Name     string
	Tasks    []task
	Networks []allocNetwork
}

type task struct {
	Name   string
	Config taskConfig
}

type taskConfig struct {
	Ports []string
}

type resource struct {
	Networks []allocNetwork
}

type allocNetwork struct {
	IP            string
	DynamicPorts  []port
	ReservedPorts []port
}

type port struct {
	Label string
	Value int
}

func getPortFromConfig(c map[string]string) (int, error) {
	if c == nil || c["port"] == "" {
		return -1, fmt.Errorf(`"port", missing from configuration`)
	}

	port, err := strconv.Atoi(c["port"])
	if err != nil {
		return -1, fmt.Errorf("port is not a number, %s", err)
	}

	return port, nil
}

func getAddressFromConfig(c map[string]string) (string, int, error) {
	if c == nil || c["address"] == "" {
		return "", -1, fmt.Errorf(`"address", missing from configuration`)
	}

	parts := strings.Split(c["address"], ":")
	port, _ := strconv.Atoi(parts[1])

	return parts[0], port, nil
}

func getDetailsFromConfig(c map[string]string) (*cacheItem, error) {
	if c == nil || c["task_port"] == "" {
		return nil, fmt.Errorf(`"task_port", missing from configuration`)
	}

	if c == nil || c["job"] == "" {
		return nil, fmt.Errorf(`"job", missing from configuration`)
	}

	if c == nil || c["group"] == "" {
		return nil, fmt.Errorf(`"group", missing from configuration`)
	}

	if c == nil || c["task"] == "" {
		return nil, fmt.Errorf(`"task", missing from configuration`)
	}

	return &cacheItem{
		Job:      c["job"],
		Group:    c["group"],
		Task:     c["task"],
		TaskPort: c["task_port"],
	}, nil
}
