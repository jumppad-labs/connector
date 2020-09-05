package crypto

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratesKeys(t *testing.T) {
	k, err := GenerateKeyPair()

	require.NoError(t, err)
	require.NotNil(t, k)
}

func TestKeyReadWriteFile(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	k, err := GenerateKeyPair()
	require.NoError(t, err)

	k.Private.WriteFile(path.Join(tmp, "test.key"))

	k2 := NewKeyPair()
	err = k2.Private.ReadFile(path.Join(tmp, "test.key"))
	require.NoError(t, err)
}

func TestPrivateKeyToString(t *testing.T) {
	k, err := GenerateKeyPair()
	require.NoError(t, err)

	pks := k.Private.String()
	require.Greater(t, len(pks), 1)

	// attempt to reassemble from PEM to key
	b, _ := pem.Decode([]byte(pks))
	require.NoError(t, err)

	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	require.NoError(t, err)
	require.NotNil(t, pk)
}
