package certificate

import (
	"bytes"
	"crypto/rand"
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

// CertWrapper is a top level structure for certificates
type CertWrapper struct {
	randreader    io.Reader
	ServerTLSConf *tls.Config
	ClientTLSConf *tls.Config
}

// NewCertWrapper creates new certificate wrapper
func NewCertWrapper(r io.Reader) *CertWrapper {
	return &CertWrapper{
		randreader: r,
	}
}

// CreateCertificate creates a server certificate and the corresponding client configuration
// The server certificate is signed with a ad-hoc creation of a CA certificate.
// The private key is generated
func (c *CertWrapper) CreateCertificate() error {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	// set up our server certificate
	serverCert, err := generateCert(c.randreader, ca, caPrivKey, []byte(`goMarkableStreamServer`))
	if err != nil {
		return err
	}
	clientCert, err := generateCert(c.randreader, ca, caPrivKey, []byte(`goMarkableStreamClient`))
	if err != nil {
		return err
	}
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())
	m := &myca{certpool}

	c.ServerTLSConf = &tls.Config{
		Certificates:          []tls.Certificate{*serverCert},
		ClientAuth:            tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:    true,
		RootCAs:               certpool,
		VerifyPeerCertificate: m.customVerify,
	}

	c.ClientTLSConf = &tls.Config{
		Certificates:          []tls.Certificate{*clientCert},
		InsecureSkipVerify:    true,
		RootCAs:               certpool,
		VerifyPeerCertificate: m.customVerify,
	}

	return nil
}

func generateCert(randrdr io.Reader, ca *x509.Certificate, caPrivKey interface{}, subjectKeyID []byte) (*tls.Certificate, error) {
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

	certPrivKey, err := rsa.GenerateKey(randrdr, 4096)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(randrdr, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}
	return &serverCert, nil

}

type myca struct {
	*x509.CertPool
}

func (m *myca) customVerify(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	for _, rawCert := range rawCerts {
		cert, _ := x509.ParseCertificate(rawCert)
		_, err := cert.Verify(x509.VerifyOptions{
			Roots: m.CertPool,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
