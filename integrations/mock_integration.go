package integrations

import "github.com/stretchr/testify/mock"

// Mock defines a mock integration which can be used in tests
type Mock struct {
	mock.Mock
}

// Register statisfies the Integration interface
func (m *Mock) Register(id string, name string, srcPort, dstPort int) error {
	args := m.Called(id, name, srcPort, dstPort)

	return args.Error(0)
}

// Deregister statisfies the Integration interface
func (m *Mock) Deregister(id string) error {
	args := m.Called(id)

	return args.Error(0)
}
