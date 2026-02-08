package jwtutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSecret(t *testing.T) {
	secret, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(secret) != SecretSize {
		t.Errorf("Generate() returned %d bytes, want %d", len(secret), SecretSize)
	}

	// Verify secrets are unique
	secret2, err := Generate()
	if err != nil {
		t.Fatalf("Generate() second call error = %v", err)
	}
	if string(secret) == string(secret2) {
		t.Error("Generate() returned identical secrets")
	}
}

func TestSecretStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSecretStore(tmpDir)

	// Initially should not exist
	if store.Exists() {
		t.Error("Exists() = true, want false for new store")
	}

	// Generate and save
	secret, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if err := store.Save(secret); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Should exist now
	if !store.Exists() {
		t.Error("Exists() = false, want true after save")
	}

	// Load and verify
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(loaded) != string(secret) {
		t.Error("Load() returned different secret than saved")
	}

	// Delete
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Exists() {
		t.Error("Exists() = true, want false after delete")
	}
}

func TestSecretStoreLoadOrGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSecretStore(tmpDir)

	// First call should generate
	secret, generated, err := store.LoadOrGenerate()
	if err != nil {
		t.Fatalf("LoadOrGenerate() error = %v", err)
	}
	if !generated {
		t.Error("LoadOrGenerate() generated = false, want true for new store")
	}
	if len(secret) != SecretSize {
		t.Errorf("LoadOrGenerate() returned %d bytes, want %d", len(secret), SecretSize)
	}

	// Second call should load
	secret2, generated2, err := store.LoadOrGenerate()
	if err != nil {
		t.Fatalf("LoadOrGenerate() second call error = %v", err)
	}
	if generated2 {
		t.Error("LoadOrGenerate() generated = true, want false for existing store")
	}
	if string(secret2) != string(secret) {
		t.Error("LoadOrGenerate() returned different secret than first call")
	}
}

func TestSecretStorePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	secretDir := filepath.Join(tmpDir, "secrets")
	store := NewSecretStore(secretDir)

	secret, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if err := store.Save(secret); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check directory permissions
	dirInfo, err := os.Stat(secretDir)
	if err != nil {
		t.Fatalf("Stat(dir) error = %v", err)
	}
	if dirInfo.Mode().Perm() != DirPermissions {
		t.Errorf("Directory permissions = %o, want %o", dirInfo.Mode().Perm(), DirPermissions)
	}

	// Check file permissions
	fileInfo, err := os.Stat(store.SecretPath())
	if err != nil {
		t.Fatalf("Stat(file) error = %v", err)
	}
	if fileInfo.Mode().Perm() != SecretPermissions {
		t.Errorf("File permissions = %o, want %o", fileInfo.Mode().Perm(), SecretPermissions)
	}
}

func TestCreateAndValidateToken(t *testing.T) {
	secret, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	tokenString, err := CreateToken("testuser", time.Hour, secret)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	// Validate the token
	token, err := ValidateToken(tokenString, secret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if token.Claims.Subject != "testuser" {
		t.Errorf("Claims.Subject = %q, want %q", token.Claims.Subject, "testuser")
	}
	if token.Claims.IssuedAt == 0 {
		t.Error("Claims.IssuedAt = 0, want non-zero")
	}
	if token.Claims.ExpiresAt == 0 {
		t.Error("Claims.ExpiresAt = 0, want non-zero")
	}
	if token.Claims.TokenID == "" {
		t.Error("Claims.TokenID = empty, want non-empty")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Create token with very short lifetime
	tokenString, err := CreateToken("testuser", -time.Hour, secret)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	// Validation should fail
	_, err = ValidateToken(tokenString, secret)
	if err != ErrTokenExpired {
		t.Errorf("ValidateToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestValidateTokenInvalidSignature(t *testing.T) {
	secret1, _ := Generate()
	secret2, _ := Generate()

	tokenString, err := CreateToken("testuser", time.Hour, secret1)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	// Validate with different secret should fail
	_, err = ValidateToken(tokenString, secret2)
	if err != ErrInvalidSignature {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidSignature", err)
	}
}

func TestValidateTokenMalformed(t *testing.T) {
	secret, _ := Generate()

	testCases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"one part", "header"},
		{"two parts", "header.payload"},
		{"wrong header", "wrongheader.payload.signature"},
		{"four parts", "a.b.c.d"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateToken(tc.token, secret)
			if err != ErrMalformedToken {
				t.Errorf("ValidateToken(%q) error = %v, want ErrMalformedToken", tc.token, err)
			}
		})
	}
}

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(ManagerConfig{
		SecretDir:     tmpDir,
		TokenLifetime: time.Hour,
		AutoGenerate:  true,
	})

	if mgr.IsInitialized() {
		t.Error("IsInitialized() = true before Initialize()")
	}

	if err := mgr.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if !mgr.IsInitialized() {
		t.Error("IsInitialized() = false after Initialize()")
	}

	// Create and validate token
	tokenString, err := mgr.CreateToken("admin")
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	token, err := mgr.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if token.Claims.Subject != "admin" {
		t.Errorf("Claims.Subject = %q, want %q", token.Claims.Subject, "admin")
	}

	// Test GetTokenLifetime
	if mgr.GetTokenLifetime() != time.Hour {
		t.Errorf("GetTokenLifetime() = %v, want %v", mgr.GetTokenLifetime(), time.Hour)
	}
}

func TestManagerForceRegenerate(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(ManagerConfig{
		SecretDir:     tmpDir,
		TokenLifetime: time.Hour,
		AutoGenerate:  true,
	})

	if err := mgr.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Create a token
	tokenString, err := mgr.CreateToken("user")
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	// Force regenerate
	if err := mgr.ForceRegenerate(); err != nil {
		t.Fatalf("ForceRegenerate() error = %v", err)
	}

	// Old token should now be invalid
	_, err = mgr.ValidateToken(tokenString)
	if err != ErrInvalidSignature {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidSignature after regenerate", err)
	}

	// New tokens should work
	newToken, err := mgr.CreateToken("user")
	if err != nil {
		t.Fatalf("CreateToken() after regenerate error = %v", err)
	}

	_, err = mgr.ValidateToken(newToken)
	if err != nil {
		t.Errorf("ValidateToken() after regenerate error = %v", err)
	}
}

func TestIsValidationError(t *testing.T) {
	if !IsValidationError(ErrTokenExpired) {
		t.Error("IsValidationError(ErrTokenExpired) = false, want true")
	}
	if !IsValidationError(ErrInvalidSignature) {
		t.Error("IsValidationError(ErrInvalidSignature) = false, want true")
	}
	if !IsValidationError(ErrMalformedToken) {
		t.Error("IsValidationError(ErrMalformedToken) = false, want true")
	}
	if IsValidationError(nil) {
		t.Error("IsValidationError(nil) = true, want false")
	}
}
