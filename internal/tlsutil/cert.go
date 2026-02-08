// Package tlsutil provides utilities for TLS certificate management.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

// CertificateInfo holds generated certificate data.
type CertificateInfo struct {
	CertPEM    []byte
	KeyPEM     []byte
	NotBefore  time.Time
	NotAfter   time.Time
	DNSNames   []string
	IPAddresses []net.IP
}

// GenerateOptions configures certificate generation.
type GenerateOptions struct {
	// ValidDays is the number of days the certificate is valid.
	ValidDays int
	// Hostnames is a list of additional DNS names to include in SANs.
	Hostnames []string
	// Organization is the certificate organization name.
	Organization string
	// CommonName is the certificate common name.
	CommonName string
}

// DefaultGenerateOptions returns sensible defaults for certificate generation.
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		ValidDays:    365,
		Organization: "goMarkableStream",
		CommonName:   "goMarkableStream",
	}
}

// GenerateCertificate creates a new self-signed certificate with ECDSA P-256.
// It automatically includes local IP addresses and common hostnames in SANs.
func GenerateCertificate(opts GenerateOptions) (*CertificateInfo, error) {
	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, opts.ValidDays)

	// Collect IP addresses
	ips, err := GetLocalAddresses()
	if err != nil {
		// Non-fatal: continue with localhost only
		ips = []net.IP{}
	}

	// Add localhost IPs
	ips = append(ips, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))

	// Collect DNS names
	dnsNames := []string{"localhost"}

	// Add device hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		dnsNames = appendUnique(dnsNames, hostname)
	}

	// Add common reMarkable hostnames
	dnsNames = appendUnique(dnsNames, "remarkable")
	dnsNames = appendUnique(dnsNames, "remarkable.local")

	// Add user-specified hostnames
	for _, h := range opts.Hostnames {
		h = strings.TrimSpace(h)
		if h != "" {
			// Check if it's an IP address
			if ip := net.ParseIP(h); ip != nil {
				ips = appendUniqueIP(ips, ip)
			} else {
				dnsNames = appendUnique(dnsNames, h)
			}
		}
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{opts.Organization},
			CommonName:   opts.CommonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return &CertificateInfo{
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		DNSNames:    dnsNames,
		IPAddresses: ips,
	}, nil
}

// GetLocalAddresses returns all local IP addresses from network interfaces.
// It filters out loopback and non-up interfaces.
func GetLocalAddresses() ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var ips []net.IP

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ips = append(ips, ipnet.IP)
			}
		}
	}

	return ips, nil
}

// appendUnique appends s to slice if not already present.
func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

// appendUniqueIP appends ip to slice if not already present.
func appendUniqueIP(slice []net.IP, ip net.IP) []net.IP {
	for _, existing := range slice {
		if existing.Equal(ip) {
			return slice
		}
	}
	return append(slice, ip)
}
