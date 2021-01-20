package remote

import (
	"net"
	"sync"

	"github.com/shipyard-run/connector/protos/shipyard"
)

type services struct {
	lock sync.Mutex
	svcs map[string]*service
}

func newServices() *services {
	return &services{
		lock: sync.Mutex{},
		svcs: map[string]*service{},
	}
}

func (s *services) get(key string) (*service, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	svc, ok := s.svcs[key]

	return svc, ok
}

func (s *services) add(key string, value *service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.svcs[key] = value
}

func (s *services) delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.svcs, key)
}

func (s *services) contains(svc *service) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, r := range s.svcs {
		// check to see if we already have a listener defined for this port
		if svc.detail.SourcePort == r.detail.SourcePort &&
			r.detail.Type == shipyard.ServiceType_REMOTE {
			return true
		}

		// check to see if there is a listener defined on the remote server for this port
		if svc.detail.Type == shipyard.ServiceType_LOCAL &&
			svc.detail.RemoteConnectorAddr == r.detail.RemoteConnectorAddr &&
			svc.detail.SourcePort == r.detail.SourcePort {
			return true
		}
	}

	return false
}

func (s *services) iterate(c func(k string, svc *service) bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for k, v := range s.svcs {
		// if true is not returned from the callback
		// the caller does not want the next item
		if !c(k, v) {
			break
		}
	}
}

type service struct {
	detail         *shipyard.Service
	tcpListener    net.Listener
	tcpConnections sync.Map
}

func newService() *service {
	return &service{tcpConnections: sync.Map{}}
}
