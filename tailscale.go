//go:build tailscale

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// TailscaleManager encapsulates tsnet.Server lifecycle management
type TailscaleManager struct {
	server        *tsnet.Server
	config        *configuration
	started       bool
	funnelEnabled bool         // Current Funnel state
	tsListener    net.Listener // Current Tailscale listener
	mu            sync.Mutex   // Protect concurrent access
}

// NewTailscaleManager creates a new TailscaleManager with the given configuration
func NewTailscaleManager(cfg *configuration) *TailscaleManager {
	return &TailscaleManager{
		config: cfg,
	}
}

// generateRandomSuffix creates a short random hex string for ephemeral hostnames
func generateRandomSuffix() string {
	b := make([]byte, 3) // 3 bytes = 6 hex chars
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Start initializes the Tailscale server and returns a listener
func (tm *TailscaleManager) Start(ctx context.Context) (net.Listener, error) {
	// Create state directory with proper permissions
	if err := tm.ensureStateDir(); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Determine hostname - add random suffix for ephemeral nodes
	hostname := tm.config.TailscaleHostname
	if tm.config.TailscaleEphemeral {
		hostname = fmt.Sprintf("%s-%s", hostname, generateRandomSuffix())
	}

	// Create and configure tsnet.Server
	tm.server = &tsnet.Server{
		Hostname: hostname,
		Dir:      tm.config.TailscaleStateDir,
	}

	// Configure auth key for headless operation
	if tm.config.TailscaleAuthKey != "" {
		tm.server.AuthKey = tm.config.TailscaleAuthKey
	}

	// Configure ephemeral mode
	if tm.config.TailscaleEphemeral {
		tm.server.Ephemeral = true
	}

	// Configure logging
	if !tm.config.TailscaleVerbose {
		tm.server.Logf = func(string, ...any) {}
	}

	// Wait for Tailscale network to be ready
	log.Println("Waiting for Tailscale network to be ready...")
	status, err := tm.server.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start Tailscale: %w", err)
	}
	tm.started = true

	// Log connection information
	tm.logConnectionInfo(status)

	// Create the appropriate listener
	listener, err := tm.createListener()
	if err != nil {
		return nil, err
	}

	tm.mu.Lock()
	tm.tsListener = listener
	tm.funnelEnabled = tm.config.TailscaleFunnel
	tm.mu.Unlock()

	return listener, nil
}

// ensureStateDir creates the state directory with proper permissions
func (tm *TailscaleManager) ensureStateDir() error {
	dir := tm.config.TailscaleStateDir
	if dir == "" {
		return fmt.Errorf("state directory not configured")
	}

	// Create parent directories if needed
	parentDir := filepath.Dir(dir)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	// Create state directory
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}

	return nil
}

// logConnectionInfo logs the Tailscale connection information
func (tm *TailscaleManager) logConnectionInfo(status *ipnstate.Status) {
	if status == nil || status.Self == nil {
		return
	}

	log.Printf("Tailscale connected as: %s", tm.server.Hostname)

	// Log all Tailscale IPs
	for _, ip := range status.Self.TailscaleIPs {
		log.Printf("Tailscale IP: %s", ip)
	}

	// Log the MagicDNS name if available
	dnsName := status.Self.DNSName
	if dnsName != "" {
		log.Printf("Tailscale DNS name: %s", dnsName)
	}

	// Log the access URL
	port := tm.config.TailscalePort
	if tm.config.TailscaleFunnel {
		log.Printf("Funnel URL: https://%s%s", dnsName, port)
	} else if tm.config.TailscaleUseTLS {
		log.Printf("Access URL: https://%s%s", dnsName, port)
	} else {
		log.Printf("Access URL: http://%s%s", dnsName, port)
	}
}

// createListener creates the appropriate listener based on configuration
func (tm *TailscaleManager) createListener() (net.Listener, error) {
	addr := tm.config.TailscalePort

	if tm.config.TailscaleFunnel {
		// Funnel provides public internet access with TLS
		log.Println("Starting Tailscale Funnel listener...")
		return tm.server.ListenFunnel("tcp", addr)
	}

	if tm.config.TailscaleUseTLS {
		// Use Tailscale's automatic TLS certificates
		log.Println("Starting Tailscale TLS listener...")
		return tm.server.ListenTLS("tcp", addr)
	}

	// Plain listener (Tailscale still encrypts via WireGuard)
	log.Println("Starting Tailscale listener (WireGuard encrypted)...")
	return tm.server.Listen("tcp", addr)
}

// Close shuts down the Tailscale server gracefully
func (tm *TailscaleManager) Close() error {
	if tm.server != nil && tm.started {
		log.Println("Shutting down Tailscale server...")
		return tm.server.Close()
	}
	return nil
}

// UseTLS returns whether the caller should apply TLS
// When using Tailscale, never apply additional TLS:
// - If TailscaleUseTLS=true, tsnet handles TLS via ListenTLS()
// - If TailscaleUseTLS=false, WireGuard encrypts, no TLS needed
func (tm *TailscaleManager) UseTLS() bool {
	return false
}

// GetFunnelInfo returns current funnel status and URL
func (tm *TailscaleManager) GetFunnelInfo() (enabled bool, url string, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.started || tm.server == nil {
		return false, "", nil
	}

	lc, err := tm.server.LocalClient()
	if err != nil {
		return false, "", err
	}

	status, err := lc.Status(context.Background())
	if err != nil {
		return false, "", err
	}

	if status == nil || status.Self == nil {
		return false, "", nil
	}

	dnsName := strings.TrimSuffix(status.Self.DNSName, ".")
	// Extract port number from TailscalePort (e.g., ":8443" -> "8443")
	portNum := strings.TrimPrefix(tm.config.TailscalePort, ":")
	url = fmt.Sprintf("https://%s:%s", dnsName, portNum)
	return tm.funnelEnabled, url, nil
}

// ToggleFunnel enables or disables Funnel, returning new listener
func (tm *TailscaleManager) ToggleFunnel(enable bool) (net.Listener, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.started || tm.server == nil {
		return nil, fmt.Errorf("Tailscale not started")
	}

	// Close existing Tailscale listener
	if tm.tsListener != nil {
		tm.tsListener.Close()
	}

	// Create new listener
	addr := tm.config.TailscalePort
	var newListener net.Listener
	var err error
	if enable {
		log.Println("Enabling Tailscale Funnel...")
		newListener, err = tm.server.ListenFunnel("tcp", addr)
	} else {
		log.Println("Disabling Tailscale Funnel...")
		newListener, err = tm.server.Listen("tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	tm.tsListener = newListener
	tm.funnelEnabled = enable
	return newListener, nil
}

// GetListener returns current Tailscale listener
func (tm *TailscaleManager) GetListener() net.Listener {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.tsListener
}
