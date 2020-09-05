package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

type KeyReaderWriter interface {
	fmt.Stringer
	PEMBlock() []byte
}

type KeyPair struct {
	Private *PrivateKey
	Public  KeyReaderWriter
}

// PrivateKey is a Golang structure which represents a Cryptographic key
type PrivateKey struct {
	*rsa.PrivateKey
}

// GenerateKeyPair creates a new RSA key pair
func GenerateKeyPair() (*KeyPair, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generating random key: %v", err)
	}

	return &KeyPair{Private: &PrivateKey{privKey}}, nil
}

func (k *PrivateKey) PEMBlock() []byte {
	pkb := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(k.PrivateKey),
	}

	return pem.EncodeToMemory(pkb)
}

// String returns a PEM encoded version of the Key
func (k *PrivateKey) String() string {
	return string(k.PEMBlock())
}
