package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizesName(t *testing.T) {
	s := SanitizeName("Test Import")

	assert.Equal(t, "test-import", s)
}
