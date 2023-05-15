package local

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestRegisterValidatesConfig(t *testing.T) {
	i := New(hclog.NewNullLogger())

	err := i.Register("abc123", map[string]string{"port": "1234"})
	require.NoError(t, err)
}

func TestRegisterReturnsErrorWhenNoPort(t *testing.T) {
	i := New(hclog.NewNullLogger())

	err := i.Register("abc123", map[string]string{})
	require.Error(t, err)
}

func TestRegisterReturnsErrorPortNoNumber(t *testing.T) {
	i := New(hclog.NewNullLogger())

	err := i.Register("abc123", map[string]string{"port": "abc"})
	require.Error(t, err)
}

func TestLookupAddressReturnsCorrectAddress(t *testing.T) {
	i := New(hclog.NewNullLogger())

	i.Register("abc123", map[string]string{"port": "1900"})
	addr, err := i.LookupAddress("abc123")
	require.NoError(t, err)

	require.Equal(t, "localhost:1900", addr)
}

func TestGetDetailsReturnsCorrectConfig(t *testing.T) {
	i := New(hclog.NewNullLogger())

	i.Register("abc123", map[string]string{"port": "1900"})
	config, err := i.GetDetails("abc123")
	require.NoError(t, err)

	require.Equal(t, "localhost:1900", config["address"])
}
