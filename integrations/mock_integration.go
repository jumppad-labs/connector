package integrations

import "github.com/stretchr/testify/mock"

// Mock defines a mock integration which can be used in tests
type Mock struct {
	mock.Mock
}

// Register satisfies the Integration interface
func (m *Mock) Register(id string, name string, srcPort, dstPort int) error {
	args := m.Called(id, name, srcPort, dstPort)

	return args.Error(0)
}

// Deregister satisfies the Integration interface
func (m *Mock) Deregister(id string) error {
	args := m.Called(id)

	return args.Error(0)
}

func (m *Mock) LookupAddress(service string) (string, error) {
	args := m.Called(service)

	if args.String(0) == "" && args.Error(1) == nil {
		return service, nil
	}

	return args.String(0), args.Error(1)
}
