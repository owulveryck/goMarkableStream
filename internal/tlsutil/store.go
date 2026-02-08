package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// CertFilename is the default certificate filename.
	CertFilename = "server.crt"
	// KeyFilename is the default key filename.
	KeyFilename = "server.key"
	// DirPermissions for the certificate directory.
	DirPermissions = 0700
	// CertPermissions for the certificate file.
	CertPermissions = 0644
	// KeyPermissions for the private key file.
	KeyPermissions = 0600
)

// Store handles certificate persistence to the filesystem.
type Store struct {
	dir string
}

// NewStore creates a new certificate store at the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// CertPath returns the full path to the certificate file.
func (s *Store) CertPath() string {
	return filepath.Join(s.dir, CertFilename)
}

// KeyPath returns the full path to the key file.
func (s *Store) KeyPath() string {
	return filepath.Join(s.dir, KeyFilename)
}

// Exists checks if both certificate and key files exist.
func (s *Store) Exists() bool {
	_, certErr := os.Stat(s.CertPath())
	_, keyErr := os.Stat(s.KeyPath())
	return certErr == nil && keyErr == nil
}

// Save writes the certificate and key to the store directory.
func (s *Store) Save(certPEM, keyPEM []byte) error {
	// Create directory with proper permissions
	if err := os.MkdirAll(s.dir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Write certificate file
	if err := os.WriteFile(s.CertPath(), certPEM, CertPermissions); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write key file
	if err := os.WriteFile(s.KeyPath(), keyPEM, KeyPermissions); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

// Load reads the certificate and key from the store.
func (s *Store) Load() (certPEM, keyPEM []byte, err error) {
	certPEM, err = os.ReadFile(s.CertPath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	keyPEM, err = os.ReadFile(s.KeyPath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key: %w", err)
	}

	return certPEM, keyPEM, nil
}

// LoadCertificate loads the certificate and returns a tls.Certificate.
func (s *Store) LoadCertificate() (tls.Certificate, error) {
	certPEM, keyPEM, err := s.Load()
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

// GetExpiry returns the expiration time of the stored certificate.
// Returns zero time if the certificate cannot be parsed.
func (s *Store) GetExpiry() (time.Time, error) {
	certPEM, err := os.ReadFile(s.CertPath())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert.NotAfter, nil
}

// IsExpiringSoon checks if the certificate expires within the given days.
func (s *Store) IsExpiringSoon(days int) (bool, error) {
	expiry, err := s.GetExpiry()
	if err != nil {
		return true, err
	}

	threshold := time.Now().AddDate(0, 0, days)
	return expiry.Before(threshold), nil
}

// Delete removes both certificate and key files.
func (s *Store) Delete() error {
	certErr := os.Remove(s.CertPath())
	keyErr := os.Remove(s.KeyPath())

	if certErr != nil && !os.IsNotExist(certErr) {
		return fmt.Errorf("failed to remove certificate: %w", certErr)
	}
	if keyErr != nil && !os.IsNotExist(keyErr) {
		return fmt.Errorf("failed to remove key: %w", keyErr)
	}

	return nil
}
