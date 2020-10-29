package integrations

import (
	"testing"

	"gotest.tools/assert"
)

func Test_SanitizesName(t *testing.T) {
	s := SanitizeName("Test Import")

	assert.Equal(t, "test-import", s)
}
