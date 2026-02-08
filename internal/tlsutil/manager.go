package tlsutil

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
)

// CertSource indicates where the certificate was loaded from.
type CertSource int

const (
	// SourceUserProvided indicates the certificate was provided by the user.
	SourceUserProvided CertSource = iota
	// SourcePersisted indicates the certificate was loaded from the store.
	SourcePersisted
	// SourceGenerated indicates the certificate was newly generated.
	SourceGenerated
	// SourceEmbedded indicates the fallback embedded certificate was used.
	SourceEmbedded
)

func (s CertSource) String() string {
	switch s {
	case SourceUserProvided:
		return "user-provided"
	case SourcePersisted:
		return "persisted"
	case SourceGenerated:
		return "generated"
	case SourceEmbedded:
		return "embedded"
	default:
		return "unknown"
	}
}

// ManagerConfig configures the TLS manager.
type ManagerConfig struct {
	// CertFile is the path to a user-provided certificate file.
	CertFile string
	// KeyFile is the path to a user-provided key file.
	KeyFile string
	// CertDir is the directory to store generated certificates.
	CertDir string
	// AutoGenerate enables automatic certificate generation.
	AutoGenerate bool
	// Hostnames is a comma-separated list of additional hostnames for SANs.
	Hostnames string
	// ValidDays is the number of days generated certificates are valid.
	ValidDays int
	// ExpiryThresholdDays is the number of days before expiry to regenerate.
	ExpiryThresholdDays int
	// EmbeddedCert is the fallback embedded certificate PEM.
	EmbeddedCert []byte
	// EmbeddedKey is the fallback embedded key PEM.
	EmbeddedKey []byte
}

// Manager handles TLS certificate resolution with priority logic.
type Manager struct {
	config ManagerConfig
	store  *Store
}

// NewManager creates a new TLS manager.
func NewManager(config ManagerConfig) *Manager {
	if config.ExpiryThresholdDays == 0 {
		config.ExpiryThresholdDays = 30
	}
	if config.ValidDays == 0 {
		config.ValidDays = 365
	}

	return &Manager{
		config: config,
		store:  NewStore(config.CertDir),
	}
}

// GetCertificate returns a TLS certificate using priority logic.
// Priority: user-provided > persisted (if not expiring) > generated > embedded
func (m *Manager) GetCertificate() (tls.Certificate, CertSource, error) {
	// 1. Check for user-provided certificate
	if m.config.CertFile != "" && m.config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
		if err != nil {
			return tls.Certificate{}, SourceUserProvided, fmt.Errorf("failed to load user-provided certificate: %w", err)
		}
		log.Printf("TLS: Using user-provided certificate from %s", m.config.CertFile)
		return cert, SourceUserProvided, nil
	}

	// 2. Check for existing persisted certificate
	if m.store.Exists() {
		expiring, err := m.store.IsExpiringSoon(m.config.ExpiryThresholdDays)
		if err == nil && !expiring {
			cert, err := m.store.LoadCertificate()
			if err == nil {
				expiry, _ := m.store.GetExpiry()
				log.Printf("TLS: Using persisted certificate from %s (expires %s)", m.store.CertPath(), expiry.Format("2006-01-02"))
				return cert, SourcePersisted, nil
			}
			log.Printf("TLS: Failed to load persisted certificate: %v", err)
		} else if expiring {
			log.Printf("TLS: Persisted certificate is expiring soon, will regenerate")
		}
	}

	// 3. Generate new certificate if auto-generation is enabled
	if m.config.AutoGenerate {
		cert, err := m.generateAndPersist()
		if err == nil {
			return cert, SourceGenerated, nil
		}
		log.Printf("TLS: Failed to generate certificate: %v", err)
	}

	// 4. Fall back to embedded certificate
	if len(m.config.EmbeddedCert) > 0 && len(m.config.EmbeddedKey) > 0 {
		cert, err := tls.X509KeyPair(m.config.EmbeddedCert, m.config.EmbeddedKey)
		if err != nil {
			return tls.Certificate{}, SourceEmbedded, fmt.Errorf("failed to load embedded certificate: %w", err)
		}
		log.Println("WARNING: Using embedded fallback certificate. Browsers will show security warnings.")
		log.Println("WARNING: Set RK_TLS_AUTO_GENERATE=true to generate a device-specific certificate.")
		return cert, SourceEmbedded, nil
	}

	return tls.Certificate{}, SourceEmbedded, fmt.Errorf("no certificate available")
}

// generateAndPersist generates a new certificate and saves it to the store.
func (m *Manager) generateAndPersist() (tls.Certificate, error) {
	// Parse hostnames
	var hostnames []string
	if m.config.Hostnames != "" {
		hostnames = strings.Split(m.config.Hostnames, ",")
	}

	opts := GenerateOptions{
		ValidDays:    m.config.ValidDays,
		Hostnames:    hostnames,
		Organization: "goMarkableStream",
		CommonName:   "goMarkableStream",
	}

	certInfo, err := GenerateCertificate(opts)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate certificate: %w", err)
	}

	// Log what was generated
	logCertificateInfo(certInfo)

	// Save to store
	if err := m.store.Save(certInfo.CertPEM, certInfo.KeyPEM); err != nil {
		log.Printf("TLS: Warning: Failed to persist certificate: %v", err)
		// Continue anyway - we can use the generated cert in memory
	} else {
		log.Printf("TLS: Certificate stored in %s", m.store.CertPath())
	}

	// Create tls.Certificate
	cert, err := tls.X509KeyPair(certInfo.CertPEM, certInfo.KeyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse generated certificate: %w", err)
	}

	return cert, nil
}

// logCertificateInfo logs information about the generated certificate.
func logCertificateInfo(info *CertificateInfo) {
	// Build a summary of IPs
	var ipStrs []string
	for _, ip := range info.IPAddresses {
		if ip.To4() != nil {
			ipStrs = append(ipStrs, ip.String())
		}
	}

	// Limit to first few IPs for readability
	ipSummary := strings.Join(ipStrs, ", ")
	if len(ipStrs) > 5 {
		ipSummary = strings.Join(ipStrs[:5], ", ") + fmt.Sprintf(" (+%d more)", len(ipStrs)-5)
	}

	var hostnames []string
	for _, dns := range info.DNSNames {
		if dns != "localhost" {
			hostnames = append(hostnames, dns)
			if len(hostnames) >= 3 {
				break
			}
		}
	}

	hostnameStr := strings.Join(hostnames, ", ")
	if hostnameStr == "" {
		hostnameStr = "localhost"
	}

	log.Printf("TLS: Generated new certificate for %s (%s)", hostnameStr, ipSummary)
	log.Printf("TLS: Certificate valid until %s", info.NotAfter.Format("2006-01-02"))
}

// GetStore returns the certificate store.
func (m *Manager) GetStore() *Store {
	return m.store
}

// ForceRegenerate forces regeneration of the certificate.
func (m *Manager) ForceRegenerate() (tls.Certificate, error) {
	// Delete existing certificate
	if err := m.store.Delete(); err != nil {
		log.Printf("TLS: Warning: Failed to delete existing certificate: %v", err)
	}

	cert, err := m.generateAndPersist()
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

// GetTLSConfig returns a tls.Config configured with the manager's certificate.
func (m *Manager) GetTLSConfig() (*tls.Config, CertSource, error) {
	cert, source, err := m.GetCertificate()
	if err != nil {
		return nil, source, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, source, nil
}

// FormatSANs returns a formatted string of SANs for logging.
func FormatSANs(ips []net.IP, dnsNames []string) string {
	var parts []string

	for _, ip := range ips {
		if ip.To4() != nil {
			parts = append(parts, ip.String())
		}
	}

	for _, dns := range dnsNames {
		if dns != "localhost" {
			parts = append(parts, dns)
		}
	}

	return strings.Join(parts, ", ")
}
