package crypto

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/url"
	"time"
)

type CertReaderWriter interface {
	fmt.Stringer
	PEMBlock() []byte
	ReadFile(path string) error
	WriteFile(path string) error
}

type X509 struct {
	*x509.Certificate
}

func (x *X509) String() string {
	/// PEM encode the certificate (this is a standard TLS encoding)
	return string(x.PEMBlock())
}

// PEMBlock encodes the certificate to a PEM encoded byte array
func (x *X509) PEMBlock() []byte {
	b := pem.Block{Type: "CERTIFICATE", Bytes: x.Raw}
	return pem.EncodeToMemory(&b)
}

// ReadFile loads the key from a PEM encoded file
func (x *X509) ReadFile(path string) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Unable to read file at path: %s", path)
	}

	b, _ := pem.Decode(d)

	xc, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return fmt.Errorf("Unable to decode file at path: %s", path)
	}

	x.Certificate = xc

	return nil
}

// WriteFile writes the Certificate to the given path
func (x *X509) WriteFile(path string) error {
	err := ioutil.WriteFile(path, x.PEMBlock(), 0400)
	if err != nil {
		return fmt.Errorf("Unable to write cert to path %s: %s", path, err)
	}

	return nil
}

// GenerateLeaf creates an X509 leaf certificate
func GenerateLeaf(name string, ipAddresses []string, dnsNames []string, rootCert *X509, rootKey *PrivateKey, leafKey *PrivateKey) (*X509, error) {
	leafCertTmpl, err := certTemplate()
	if err != nil {
		return nil, fmt.Errorf("Unable to generate root certificate template: %s", err)
	}
	leafCertTmpl.Subject.CommonName = name

	leafCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
	leafCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	ips := []net.IP{}

	for _, i := range ipAddresses {
		ips = append(ips, net.ParseIP(i))
	}

	leafCertTmpl.IPAddresses = ips
	leafCertTmpl.DNSNames = dnsNames

	// generate a random spiffe id
	spiffe, _ := url.Parse(fmt.Sprintf("spiffe://jumppad.dev/private/%d", time.Now().UnixNano()))
	leafCertTmpl.URIs = []*url.URL{
		spiffe,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, leafCertTmpl, rootCert.Certificate, leafKey.Public(), rootKey)
	if err != nil {
		return nil, err
	}

	// parse the resulting certificate so we can use it again
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &X509{cert}, nil
}

// GenerateCA creates an X509 CA certificate
func GenerateCA(name string, pk *PrivateKey) (*X509, error) {
	rootCertTmpl, err := certTemplate()
	if err != nil {
		return nil, fmt.Errorf("Unable to generate root certificate template: %s", err)
	}

	rootCertTmpl.IsCA = true
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.Subject.CommonName = name

	certDER, err := x509.CreateCertificate(rand.Reader, rootCertTmpl, rootCertTmpl, pk.Public(), pk)
	if err != nil {
		return nil, err
	}
	// parse the resulting certificate so we can use it again
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &X509{cert}, nil
}

func certTemplate() (*x509.Certificate, error) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"Shipyard"}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add((24 * 265) * time.Hour), // valid for a year
		BasicConstraintsValid: true,
	}

	return &tmpl, nil
}
