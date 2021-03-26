package certs

//go:generate go run cmd/main.go

import (
	// Embedding the certificate
	_ "embed"
	"errors"

	"github.com/owulveryck/goMarkableStream/internal/certificate"
)

//go:embed certs.bin
var certs []byte

// GetCertificateWrapper ...
func GetCertificateWrapper() (*certificate.CertConfigCarrier, error) {

	if certs == nil {
		return nil, errors.New("bad certificate")
	}
	var cw certificate.CertConfigCarrier
	err := cw.GobDecode(certs)
	if err != nil {
		return nil, err
	}
	return &cw, nil

}
