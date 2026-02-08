package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateCertificate(t *testing.T) {
	opts := DefaultGenerateOptions()
	opts.ValidDays = 30
	opts.Hostnames = []string{"custom.local", "192.168.1.100"}

	certInfo, err := GenerateCertificate(opts)
	if err != nil {
		t.Fatalf("GenerateCertificate failed: %v", err)
	}

	// Verify certificate PEM is valid
	block, _ := pem.Decode(certInfo.CertPEM)
	if block == nil {
		t.Fatal("Failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Verify organization and common name
	if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "goMarkableStream" {
		t.Errorf("Expected organization 'goMarkableStream', got %v", cert.Subject.Organization)
	}

	if cert.Subject.CommonName != "goMarkableStream" {
		t.Errorf("Expected common name 'goMarkableStream', got %s", cert.Subject.CommonName)
	}

	// Verify DNS names include localhost
	hasLocalhost := false
	for _, dns := range cert.DNSNames {
		if dns == "localhost" {
			hasLocalhost = true
			break
		}
	}
	if !hasLocalhost {
		t.Error("Certificate should include 'localhost' in DNS names")
	}

	// Verify custom hostname was added
	hasCustom := false
	for _, dns := range cert.DNSNames {
		if dns == "custom.local" {
			hasCustom = true
			break
		}
	}
	if !hasCustom {
		t.Error("Certificate should include 'custom.local' in DNS names")
	}

	// Verify IP was added
	hasIP := false
	expectedIP := net.ParseIP("192.168.1.100")
	for _, ip := range cert.IPAddresses {
		if ip.Equal(expectedIP) {
			hasIP = true
			break
		}
	}
	if !hasIP {
		t.Error("Certificate should include 192.168.1.100 in IP addresses")
	}

	// Verify validity period
	if cert.NotAfter.Before(time.Now().AddDate(0, 0, 29)) {
		t.Error("Certificate expiry should be at least 29 days from now")
	}
	if cert.NotAfter.After(time.Now().AddDate(0, 0, 31)) {
		t.Error("Certificate expiry should be at most 31 days from now")
	}

	// Verify key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("Certificate should have KeyUsageDigitalSignature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
		t.Error("Certificate should have KeyUsageKeyEncipherment")
	}

	// Verify ext key usage
	hasServerAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		t.Error("Certificate should have ExtKeyUsageServerAuth")
	}

	// Verify we can create a tls.Certificate from the generated PEM
	_, err = tls.X509KeyPair(certInfo.CertPEM, certInfo.KeyPEM)
	if err != nil {
		t.Fatalf("Failed to create X509KeyPair: %v", err)
	}
}

func TestStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tlsutil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	// Verify paths
	if store.CertPath() != filepath.Join(tmpDir, "server.crt") {
		t.Errorf("Unexpected cert path: %s", store.CertPath())
	}
	if store.KeyPath() != filepath.Join(tmpDir, "server.key") {
		t.Errorf("Unexpected key path: %s", store.KeyPath())
	}

	// Initially should not exist
	if store.Exists() {
		t.Error("Store should not exist initially")
	}

	// Generate a certificate
	opts := DefaultGenerateOptions()
	opts.ValidDays = 365
	certInfo, err := GenerateCertificate(opts)
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Save it
	err = store.Save(certInfo.CertPEM, certInfo.KeyPEM)
	if err != nil {
		t.Fatalf("Failed to save certificate: %v", err)
	}

	// Now should exist
	if !store.Exists() {
		t.Error("Store should exist after save")
	}

	// Verify file permissions
	certStat, err := os.Stat(store.CertPath())
	if err != nil {
		t.Fatalf("Failed to stat cert file: %v", err)
	}
	if certStat.Mode().Perm() != 0644 {
		t.Errorf("Certificate file should have 0644 permissions, got %o", certStat.Mode().Perm())
	}

	keyStat, err := os.Stat(store.KeyPath())
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}
	if keyStat.Mode().Perm() != 0600 {
		t.Errorf("Key file should have 0600 permissions, got %o", keyStat.Mode().Perm())
	}

	// Load it back
	loadedCert, loadedKey, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	if string(loadedCert) != string(certInfo.CertPEM) {
		t.Error("Loaded certificate doesn't match saved certificate")
	}
	if string(loadedKey) != string(certInfo.KeyPEM) {
		t.Error("Loaded key doesn't match saved key")
	}

	// Get expiry
	expiry, err := store.GetExpiry()
	if err != nil {
		t.Fatalf("Failed to get expiry: %v", err)
	}
	if expiry.Before(time.Now().AddDate(0, 0, 364)) {
		t.Error("Expiry should be about 365 days from now")
	}

	// Check expiring soon (should not be expiring within 30 days)
	expiring, err := store.IsExpiringSoon(30)
	if err != nil {
		t.Fatalf("Failed to check expiring soon: %v", err)
	}
	if expiring {
		t.Error("Certificate should not be expiring soon")
	}

	// Check expiring within 400 days (should be true)
	expiring, err = store.IsExpiringSoon(400)
	if err != nil {
		t.Fatalf("Failed to check expiring soon: %v", err)
	}
	if !expiring {
		t.Error("Certificate should be expiring within 400 days")
	}

	// Load as tls.Certificate
	tlsCert, err := store.LoadCertificate()
	if err != nil {
		t.Fatalf("Failed to load as tls.Certificate: %v", err)
	}
	if len(tlsCert.Certificate) == 0 {
		t.Error("Loaded tls.Certificate should have certificate data")
	}

	// Delete
	err = store.Delete()
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}
	if store.Exists() {
		t.Error("Store should not exist after delete")
	}
}

func TestManager(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tlsutil-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Embedded fallback certificate for testing
	embeddedOpts := DefaultGenerateOptions()
	embeddedOpts.ValidDays = 10
	embeddedCert, err := GenerateCertificate(embeddedOpts)
	if err != nil {
		t.Fatalf("Failed to generate embedded cert: %v", err)
	}

	t.Run("auto-generate certificate", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "auto-gen")
		mgr := NewManager(ManagerConfig{
			CertDir:      subDir,
			AutoGenerate: true,
			ValidDays:    90,
			Hostnames:    "test.local",
			EmbeddedCert: embeddedCert.CertPEM,
			EmbeddedKey:  embeddedCert.KeyPEM,
		})

		cert, source, err := mgr.GetCertificate()
		if err != nil {
			t.Fatalf("GetCertificate failed: %v", err)
		}

		if source != SourceGenerated {
			t.Errorf("Expected SourceGenerated, got %s", source)
		}

		if len(cert.Certificate) == 0 {
			t.Error("Certificate should have data")
		}

		// Certificate should be persisted now
		if !mgr.GetStore().Exists() {
			t.Error("Certificate should be persisted")
		}
	})

	t.Run("use persisted certificate", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "persisted")
		mgr := NewManager(ManagerConfig{
			CertDir:             subDir,
			AutoGenerate:        true,
			ValidDays:           365,
			ExpiryThresholdDays: 30,
			EmbeddedCert:        embeddedCert.CertPEM,
			EmbeddedKey:         embeddedCert.KeyPEM,
		})

		// First call generates
		_, source1, err := mgr.GetCertificate()
		if err != nil {
			t.Fatalf("First GetCertificate failed: %v", err)
		}
		if source1 != SourceGenerated {
			t.Errorf("First call expected SourceGenerated, got %s", source1)
		}

		// Second call should use persisted
		_, source2, err := mgr.GetCertificate()
		if err != nil {
			t.Fatalf("Second GetCertificate failed: %v", err)
		}
		if source2 != SourcePersisted {
			t.Errorf("Second call expected SourcePersisted, got %s", source2)
		}
	})

	t.Run("fallback to embedded when auto-generate disabled", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "fallback")
		mgr := NewManager(ManagerConfig{
			CertDir:      subDir,
			AutoGenerate: false,
			EmbeddedCert: embeddedCert.CertPEM,
			EmbeddedKey:  embeddedCert.KeyPEM,
		})

		_, source, err := mgr.GetCertificate()
		if err != nil {
			t.Fatalf("GetCertificate failed: %v", err)
		}

		if source != SourceEmbedded {
			t.Errorf("Expected SourceEmbedded, got %s", source)
		}
	})

	t.Run("user-provided certificate", func(t *testing.T) {
		// Create user-provided cert files
		userDir := filepath.Join(tmpDir, "user-provided")
		os.MkdirAll(userDir, 0755)

		userOpts := DefaultGenerateOptions()
		userOpts.ValidDays = 365
		userCert, err := GenerateCertificate(userOpts)
		if err != nil {
			t.Fatalf("Failed to generate user cert: %v", err)
		}

		certFile := filepath.Join(userDir, "user.crt")
		keyFile := filepath.Join(userDir, "user.key")
		os.WriteFile(certFile, userCert.CertPEM, 0644)
		os.WriteFile(keyFile, userCert.KeyPEM, 0600)

		mgr := NewManager(ManagerConfig{
			CertFile:     certFile,
			KeyFile:      keyFile,
			CertDir:      filepath.Join(tmpDir, "not-used"),
			AutoGenerate: true,
			EmbeddedCert: embeddedCert.CertPEM,
			EmbeddedKey:  embeddedCert.KeyPEM,
		})

		_, source, err := mgr.GetCertificate()
		if err != nil {
			t.Fatalf("GetCertificate failed: %v", err)
		}

		if source != SourceUserProvided {
			t.Errorf("Expected SourceUserProvided, got %s", source)
		}
	})

	t.Run("GetTLSConfig", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "tls-config")
		mgr := NewManager(ManagerConfig{
			CertDir:      subDir,
			AutoGenerate: true,
			ValidDays:    365,
		})

		tlsConfig, source, err := mgr.GetTLSConfig()
		if err != nil {
			t.Fatalf("GetTLSConfig failed: %v", err)
		}

		if source != SourceGenerated {
			t.Errorf("Expected SourceGenerated, got %s", source)
		}

		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Error("MinVersion should be TLS 1.2")
		}

		if len(tlsConfig.Certificates) == 0 {
			t.Error("Should have at least one certificate")
		}
	})
}

func TestGetLocalAddresses(t *testing.T) {
	ips, err := GetLocalAddresses()
	if err != nil {
		t.Fatalf("GetLocalAddresses failed: %v", err)
	}

	// Should return at least some IPs (unless running in a very restricted environment)
	t.Logf("Found %d local IPs", len(ips))
	for _, ip := range ips {
		t.Logf("  %s", ip)
	}
}

func TestAppendUnique(t *testing.T) {
	slice := []string{"a", "b"}
	result := appendUnique(slice, "c")
	if len(result) != 3 {
		t.Errorf("Expected length 3, got %d", len(result))
	}

	result = appendUnique(result, "b")
	if len(result) != 3 {
		t.Errorf("Expected length 3 (no duplicate), got %d", len(result))
	}
}

func TestAppendUniqueIP(t *testing.T) {
	slice := []net.IP{net.ParseIP("127.0.0.1")}
	result := appendUniqueIP(slice, net.ParseIP("192.168.1.1"))
	if len(result) != 2 {
		t.Errorf("Expected length 2, got %d", len(result))
	}

	result = appendUniqueIP(result, net.ParseIP("127.0.0.1"))
	if len(result) != 2 {
		t.Errorf("Expected length 2 (no duplicate), got %d", len(result))
	}
}
