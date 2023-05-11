package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jumppad-labs/connector/crypto"

	"github.com/hashicorp/go-hclog"
)

type GenerateCertificate struct {
	logger hclog.Logger
	caKey  *crypto.PrivateKey
	caCert *crypto.X509
}

type certificateRequest struct {
	Name        string   `json:"name"`
	IPAddresses []string `json:"ip_addresses"`
	DNSNames    []string `json:"dns_names"`
}

type certificateResponse struct {
	CA          string `json:"ca"`
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

func NewGenerateCertificate(l hclog.Logger, pathCACert, pathCAKey string) *GenerateCertificate {
	// try to load the ca key
	key := &crypto.PrivateKey{}
	err := key.ReadFile(pathCAKey)
	if err != nil {
		l.Error("Unable to load Private Key, endpoint disabled")
		return &GenerateCertificate{l, nil, nil}
	}

	ca := &crypto.X509{}
	err = ca.ReadFile(pathCACert)
	if err != nil {
		l.Error("Unable to load Certificate, endpoint disabled")
		return &GenerateCertificate{l, nil, nil}
	}

	return &GenerateCertificate{l, key, ca}
}

func (gc *GenerateCertificate) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	// parse the request
	cr := &certificateRequest{}
	err := json.NewDecoder(r.Body).Decode(cr)
	if err != nil {
		gc.logger.Error("Unable to parse request", "error", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	gc.logger.Info("Generate new certificate", "payload", cr)

	// generate the cert
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		gc.logger.Error("Unable to generate leaf key", "error", err)
		http.Error(rw, "unable to generate leaf key", http.StatusInternalServerError)
		return
	}

	cert, err := crypto.GenerateLeaf(cr.Name, cr.IPAddresses, cr.DNSNames, gc.caCert, gc.caKey, kp.Private)
	if err != nil {
		gc.logger.Error("Unable to generate certificate", "error", err)
		http.Error(rw, "unable to generate certificate", http.StatusInternalServerError)
		return
	}

	cw := certificateResponse{
		CA:          gc.caCert.String(),
		Certificate: cert.String(),
		PrivateKey:  kp.Private.String(),
	}

	json.NewEncoder(rw).Encode(cw)
}
