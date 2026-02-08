package main

import (
	"crypto/rand"
	"sync"
)

// FunnelCredentials manages temporary credentials for Tailscale Funnel access.
// These credentials are generated when Funnel is enabled and invalidated when disabled.
type FunnelCredentials struct {
	mu       sync.RWMutex
	username string
	password string
	active   bool
}

// NewFunnelCredentials creates a new FunnelCredentials manager.
func NewFunnelCredentials() *FunnelCredentials {
	return &FunnelCredentials{}
}

// Generate creates new temporary credentials with a fixed username "stream"
// and a random 8-character alphanumeric password.
func (fc *FunnelCredentials) Generate() (username, password string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.username = "stream"
	fc.password = generateRandomPassword(8)
	fc.active = true

	return fc.username, fc.password
}

// Clear invalidates the temporary credentials.
func (fc *FunnelCredentials) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.username = ""
	fc.password = ""
	fc.active = false
}

// Validate checks if the provided credentials match the active temporary credentials.
func (fc *FunnelCredentials) Validate(username, password string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if !fc.active {
		return false
	}

	return fc.username == username && fc.password == password
}

// IsActive returns whether temporary credentials are currently active.
func (fc *FunnelCredentials) IsActive() bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return fc.active
}

// GetCredentials returns the current temporary credentials if active.
// Returns empty strings if not active.
func (fc *FunnelCredentials) GetCredentials() (username, password string, active bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return fc.username, fc.password, fc.active
}

// generateRandomPassword generates a random alphanumeric password of the specified length.
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple pattern if crypto/rand fails (should never happen)
		return "temppass"
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
