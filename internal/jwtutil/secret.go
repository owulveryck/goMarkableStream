package jwtutil

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// SecretFilename is the default secret key filename.
	SecretFilename = "jwt_secret.key"
	// SecretSize is the size of the secret key in bytes (256 bits).
	SecretSize = 32
	// DirPermissions for the secrets directory.
	DirPermissions = 0700
	// SecretPermissions for the secret key file.
	SecretPermissions = 0600
)

// SecretStore handles JWT secret key persistence to the filesystem.
type SecretStore struct {
	dir string
}

// NewSecretStore creates a new secret store at the given directory.
func NewSecretStore(dir string) *SecretStore {
	return &SecretStore{dir: dir}
}

// SecretPath returns the full path to the secret key file.
func (s *SecretStore) SecretPath() string {
	return filepath.Join(s.dir, SecretFilename)
}

// Exists checks if the secret key file exists.
func (s *SecretStore) Exists() bool {
	_, err := os.Stat(s.SecretPath())
	return err == nil
}

// Generate creates a new cryptographically secure secret key.
func Generate() ([]byte, error) {
	secret := make([]byte, SecretSize)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random secret: %w", err)
	}
	return secret, nil
}

// Save writes the secret key to the store directory.
func (s *SecretStore) Save(secret []byte) error {
	// Create directory with proper permissions
	if err := os.MkdirAll(s.dir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Write secret file
	if err := os.WriteFile(s.SecretPath(), secret, SecretPermissions); err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}

	return nil
}

// Load reads the secret key from the store.
func (s *SecretStore) Load() ([]byte, error) {
	secret, err := os.ReadFile(s.SecretPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	if len(secret) != SecretSize {
		return nil, fmt.Errorf("invalid secret size: got %d, expected %d", len(secret), SecretSize)
	}

	return secret, nil
}

// LoadOrGenerate loads an existing secret or generates and saves a new one.
func (s *SecretStore) LoadOrGenerate() ([]byte, bool, error) {
	if s.Exists() {
		secret, err := s.Load()
		if err != nil {
			return nil, false, err
		}
		return secret, false, nil // false = not newly generated
	}

	secret, err := Generate()
	if err != nil {
		return nil, false, err
	}

	if err := s.Save(secret); err != nil {
		return nil, false, err
	}

	return secret, true, nil // true = newly generated
}

// Delete removes the secret key file.
func (s *SecretStore) Delete() error {
	err := os.Remove(s.SecretPath())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove secret: %w", err)
	}
	return nil
}
