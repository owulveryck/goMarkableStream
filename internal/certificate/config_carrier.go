package certificate

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"time"
)

// CertConfigCarrier is a top level structure for certificates
type CertConfigCarrier struct {
	randreader    io.Reader
	ServerTLSConf *tls.Config
	ClientTLSConf *tls.Config
	serverKey     []byte
	serverCert    []byte
	caKey         []byte
	caCert        []byte
	clientKey     []byte
	clientCert    []byte
}

// NewCertConfigCarrier creates new certificate wrapper
func NewCertConfigCarrier(r io.Reader) *CertConfigCarrier {
	return &CertConfigCarrier{
		randreader: r,
	}
}

// Make creates a server certificate and the corresponding client configuration
// The server certificate is signed with a ad-hoc creation of a CA certificate.
// The private key is generated
func (c *CertConfigCarrier) Make() error {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"goMarkableStream"},
			Country:       []string{"FR"},
			Province:      []string{""},
			Locality:      []string{"Lille"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(c.randreader, 2048)
	if err != nil {
		return err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(c.randreader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
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
	c.caCert = caPEM.Bytes()
	c.caKey = caPrivKeyPEM.Bytes()

	// set up our server certificate
	serverCertBytes, serverKeyBytes, err := generateKeyPair(c.randreader, ca, caPrivKey, []byte(`goMarkableStreamServer`))
	if err != nil {
		return err
	}
	c.serverCert = serverCertBytes
	c.serverKey = serverKeyBytes
	serverCert, err := generateCertificate(serverCertBytes, serverKeyBytes)
	if err != nil {
		return err
	}
	// set up our client certificate
	clientCertBytes, clientKeyBytes, err := generateKeyPair(c.randreader, ca, caPrivKey, []byte(`goMarkableStreamServer`))
	if err != nil {
		return err
	}
	c.clientCert = clientCertBytes
	c.clientKey = clientKeyBytes
	clientCert, err := generateCertificate(clientCertBytes, clientKeyBytes)
	if err != nil {
		return err
	}
	err = c.generateClientConf(clientCert)
	if err != nil {
		return err
	}
	err = c.generateServerConf(serverCert)
	if err != nil {
		return err
	}
	return nil
}

func (c *CertConfigCarrier) generateClientConf(clientCert *tls.Certificate) error {
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(c.caCert)

	m := &myca{certpool}
	c.ClientTLSConf = &tls.Config{
		Certificates:          []tls.Certificate{*clientCert},
		InsecureSkipVerify:    true,
		RootCAs:               certpool,
		VerifyPeerCertificate: m.customVerify,
	}
	c.ClientTLSConf.BuildNameToCertificate()
	return nil
}

func (c *CertConfigCarrier) generateServerConf(serverCert *tls.Certificate) error {
	if c.caCert == nil || c.clientCert == nil {
		return errors.New("no certificates in the warpper")
	}
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(c.caCert)
	clientCertPool := x509.NewCertPool()
	//clientCertPool.AddCert(clientCert.Leaf)
	block, _ := pem.Decode(c.clientCert)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("failed to decode PEM block containing public key")
	}

	/*
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return err
		}
		clientCertPool.AddCert(pub)
	*/
	clientCRT, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	clientCertPool.AddCert(clientCRT)
	m := &myca{certpool}

	c.ServerTLSConf = &tls.Config{
		Certificates:          []tls.Certificate{*serverCert},
		ClientAuth:            tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:    true,
		RootCAs:               certpool,
		ClientCAs:             clientCertPool,
		VerifyPeerCertificate: m.customVerify,
	}
	c.ServerTLSConf.BuildNameToCertificate()
	return nil
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
