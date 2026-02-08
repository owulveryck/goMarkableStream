package jwtutil

import (
	"log"
	"time"
)

// ManagerConfig configures the JWT manager.
type ManagerConfig struct {
	// SecretDir is the directory to store the secret key.
	SecretDir string
	// TokenLifetime is the validity duration of issued tokens.
	TokenLifetime time.Duration
	// AutoGenerate enables automatic secret key generation.
	AutoGenerate bool
}

// Manager handles JWT token creation and validation.
type Manager struct {
	config ManagerConfig
	store  *SecretStore
	secret []byte
}

// NewManager creates a new JWT manager.
func NewManager(config ManagerConfig) *Manager {
	if config.TokenLifetime == 0 {
		config.TokenLifetime = 24 * time.Hour
	}

	return &Manager{
		config: config,
		store:  NewSecretStore(config.SecretDir),
	}
}

// Initialize loads or generates the secret key.
// Returns an error if the secret cannot be loaded or generated.
func (m *Manager) Initialize() error {
	if m.config.AutoGenerate {
		secret, generated, err := m.store.LoadOrGenerate()
		if err != nil {
			return err
		}
		m.secret = secret
		if generated {
			log.Printf("JWT: Generated new secret key in %s", m.store.SecretPath())
		} else {
			log.Printf("JWT: Loaded secret key from %s", m.store.SecretPath())
		}
		return nil
	}

	// Auto-generate is disabled, just try to load
	secret, err := m.store.Load()
	if err != nil {
		return err
	}
	m.secret = secret
	log.Printf("JWT: Loaded secret key from %s", m.store.SecretPath())
	return nil
}

// CreateToken creates a new JWT token for the given subject (username).
func (m *Manager) CreateToken(subject string) (string, error) {
	return CreateToken(subject, m.config.TokenLifetime, m.secret)
}

// ValidateToken validates a JWT token and returns the parsed token.
func (m *Manager) ValidateToken(tokenString string) (*Token, error) {
	return ValidateToken(tokenString, m.secret)
}

// GetTokenLifetime returns the configured token lifetime.
func (m *Manager) GetTokenLifetime() time.Duration {
	return m.config.TokenLifetime
}

// GetStore returns the secret store.
func (m *Manager) GetStore() *SecretStore {
	return m.store
}

// IsInitialized returns true if the manager has been initialized with a secret.
func (m *Manager) IsInitialized() bool {
	return len(m.secret) > 0
}

// ForceRegenerate generates a new secret key, invalidating all existing tokens.
func (m *Manager) ForceRegenerate() error {
	// Delete existing secret
	if err := m.store.Delete(); err != nil {
		log.Printf("JWT: Warning: Failed to delete existing secret: %v", err)
	}

	// Generate new secret
	secret, err := Generate()
	if err != nil {
		return err
	}

	if err := m.store.Save(secret); err != nil {
		return err
	}

	m.secret = secret
	log.Printf("JWT: Regenerated secret key in %s", m.store.SecretPath())
	return nil
}
