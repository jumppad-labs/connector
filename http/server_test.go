package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestServerStartsCorrectly(t *testing.T) {
	s := NewLocalServer("", "", "", ":8082", ":8081", hclog.Default())

	t.Cleanup(func() {
		s.Close()
	})

	s.Serve()

	time.Sleep(100 * time.Millisecond)

	// check the health
	r, err := http.DefaultClient.Get("http://localhost:8081/health")

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}
