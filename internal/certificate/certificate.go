package certificate

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"time"
)

func generateCertificate(certPEM, certPrivKeyPEM []byte) (*tls.Certificate, error) {
	serverCert, err := tls.X509KeyPair(certPEM, certPrivKeyPEM)
	if err != nil {
		return nil, err
	}
	return &serverCert, nil
}

func generateKeyPair(randrdr io.Reader, ca *x509.Certificate, caPrivKey interface{}, subjectKeyID []byte) (certPEM []byte, certPrivKeyPEM []byte, err error) {
	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"goMarkableStream"},
			Country:       []string{"FR"},
			Province:      []string{""},
			Locality:      []string{"Lille"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: subjectKeyID,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(randrdr, 2048)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(randrdr, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPEMBuf := new(bytes.Buffer)
	err = pem.Encode(certPEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	certPrivKeyPEMBuf := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEMBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, nil, err
	}
	return certPEMBuf.Bytes(), certPrivKeyPEMBuf.Bytes(), nil
}
