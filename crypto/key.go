package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

type KeyReaderWriter interface {
	fmt.Stringer
	PEMBlock() []byte
	ReadFile(path string) error
	WriteFile(path string) error
}

type KeyPair struct {
	Private *PrivateKey
	Public  KeyReaderWriter
}

func NewKeyPair() *KeyPair {
	return &KeyPair{Private: &PrivateKey{}}
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
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(k.PrivateKey),
	}

	return pem.EncodeToMemory(pkb)
}

// String returns a PEM encoded version of the Key
func (k *PrivateKey) String() string {
	return string(k.PEMBlock())
}

// ReadFile loads the key from a PEM encoded file
func (k *PrivateKey) ReadFile(path string) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Unable to read file at path: %s", path)
	}

	pb, _ := pem.Decode(d)

	pk, err := x509.ParsePKCS1PrivateKey(pb.Bytes)
	if err != nil {
		return fmt.Errorf("Unable to decode file at path: %s", path)
	}

	k.PrivateKey = pk

	return nil
}

func (k *PrivateKey) WriteFile(path string) error {
	err := ioutil.WriteFile(path, k.PEMBlock(), 0400)
	if err != nil {
		return fmt.Errorf("Unable to write CA to path %s: %s", path, err)
	}

	return nil
}
