package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratesCA(t *testing.T) {
	k, err := GenerateKeyPair()
	require.NoError(t, err)

	c, err := GenerateCA(k.Private)

	require.True(t, c.IsCA)
}

func TestCASerializeToString(t *testing.T) {
	k, err := GenerateKeyPair()
	require.NoError(t, err)

	c, err := GenerateCA(k.Private)
	p := c.String()

	assert.Greater(t, len(p), 1)

	b, _ := pem.Decode([]byte(p))

	xc, err := x509.ParseCertificate(b.Bytes)
	require.NoError(t, err)
	require.True(t, xc.IsCA)
}

func TestCAReadWriteFile(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	k, err := GenerateKeyPair()
	require.NoError(t, err)

	c, err := GenerateCA(k.Private)

	c.WriteFile(path.Join(tmp, "test.cert"))
	require.NoError(t, err)

	c2 := &X509{}
	c2.ReadFile(path.Join(tmp, "test.cert"))
	require.NoError(t, err)

	require.True(t, c2.IsCA)
}

func TestGeneratesLeaf(t *testing.T) {
	rk, err := GenerateKeyPair()
	require.NoError(t, err)

	ca, err := GenerateCA(rk.Private)
	require.True(t, ca.IsCA)

	// create the leaf key
	lk, err := GenerateKeyPair()
	require.NoError(t, err)

	lc, err := GenerateLeaf([]string{"127.0.0.1"}, []string{"toasties"}, ca, rk.Private, lk.Private)
	require.NoError(t, err)

	require.Equal(t, lc.IPAddresses[0].String(), "127.0.0.1")
	require.Equal(t, lc.DNSNames[0], "toasties")

	err = lc.CheckSignatureFrom(ca.Certificate)
	require.NoError(t, err)
}

func TestCertValidHTTPCert(t *testing.T) {
	rk, err := GenerateKeyPair()
	require.NoError(t, err)

	ca, err := GenerateCA(rk.Private)
	require.True(t, ca.IsCA)

	// create the server key
	sk, err := GenerateKeyPair()
	require.NoError(t, err)

	// generate the server cert
	sc, err := GenerateLeaf([]string{"127.0.0.1"}, nil, ca, rk.Private, sk.Private)
	require.NoError(t, err)

	// create and start a test server
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
	})

	ts := httptest.NewUnstartedServer(http.DefaultServeMux)

	// configure TLS
	keyPair, _ := tls.X509KeyPair(
		sc.PEMBlock(),
		sk.Private.PEMBlock(),
	)

	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{keyPair},
	}

	ts.StartTLS()
	t.Cleanup(func() {
		ts.Close()
	})

	// create the client
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(ca.PEMBlock())

	// configure a client to use trust those certificates
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}

	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
