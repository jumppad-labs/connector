package remote

import (
	"testing"

	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/stretchr/testify/assert"
)

func TestContainsNotFindsRemoteService(t *testing.T) {
	s := newServices()

	svc1 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_LOCAL,
		},
	}

	svc2 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_REMOTE,
		},
	}

	s.add("service1", svc1)

	assert.False(t, s.contains(svc2))
}

func TestContainsFindsRemoteService(t *testing.T) {
	s := newServices()

	svc1 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_REMOTE,
		},
	}

	svc2 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_REMOTE,
		},
	}

	s.add("service1", svc1)

	assert.True(t, s.contains(svc2))
}

func TestContainsNotFindsLocalService(t *testing.T) {
	s := newServices()

	svc1 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_LOCAL,
		},
	}

	svc2 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local2",
			Type:                shipyard.ServiceType_LOCAL,
		},
	}

	s.add("service1", svc1)

	assert.False(t, s.contains(svc2))
}

func TestContainsFindsLocalService(t *testing.T) {
	s := newServices()

	svc1 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_LOCAL,
		},
	}

	svc2 := &service{
		detail: &shipyard.Service{
			SourcePort:          9090,
			RemoteConnectorAddr: "local",
			Type:                shipyard.ServiceType_LOCAL,
		},
	}

	s.add("service1", svc1)

	assert.True(t, s.contains(svc2))
}
