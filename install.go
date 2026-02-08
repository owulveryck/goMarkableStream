//go:build linux && (arm || arm64)

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	serviceFilePath = "/etc/systemd/system/goMarkableStream.service"
	envFilePath     = "/home/root/.config/goMarkableStream/env"
	configDir       = "/home/root/.config/goMarkableStream"
)

const envFileContent = `# goMarkableStream Environment Configuration
# Uncomment and modify values as needed. All variables use the RK_ prefix.

# ==============================================================================
# Server Configuration
# ==============================================================================
# RK_SERVER_BIND_ADDR=:2001
# RK_SERVER_USERNAME=admin
# RK_SERVER_PASSWORD=password
# RK_HTTPS=true
# RK_DEV_MODE=false
# RK_DELTA_THRESHOLD=0.30
# RK_DEBUG=false

# ==============================================================================
# TLS Certificate Configuration
# ==============================================================================
# RK_TLS_CERT_FILE=
# RK_TLS_KEY_FILE=
# RK_TLS_CERT_DIR=/home/root/.config/goMarkableStream/certs
# RK_TLS_AUTO_GENERATE=true
# RK_TLS_HOSTNAMES=
# RK_TLS_VALID_DAYS=365

# ==============================================================================
# JWT Authentication Configuration
# ==============================================================================
# RK_JWT_ENABLED=true
# RK_JWT_SECRET_DIR=/home/root/.config/goMarkableStream/secrets
# RK_JWT_TOKEN_LIFETIME=24h
# RK_JWT_AUTO_GENERATE=true

# ==============================================================================
# Tailscale Configuration (requires tailscale build tag)
# ==============================================================================
# RK_TAILSCALE_ENABLED=false
# RK_TAILSCALE_PORT=:8443
# RK_TAILSCALE_HOSTNAME=gomarkablestream
# RK_TAILSCALE_STATE_DIR=/home/root/.tailscale/gomarkablestream
# RK_TAILSCALE_AUTHKEY=
# RK_TAILSCALE_EPHEMERAL=false
# RK_TAILSCALE_FUNNEL=false
# RK_TAILSCALE_USE_TLS=false
# RK_TAILSCALE_VERBOSE=false
`

const serviceFileTemplate = `[Unit]
Description=goMarkableStream - Screen streaming for reMarkable
After=xochitl.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
EnvironmentFile=/home/root/.config/goMarkableStream/env
Restart=on-failure
RestartSec=5
TimeoutStopSec=10

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=goMarkableStream

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true
ReadWritePaths=/home/root/.config/goMarkableStream
ReadWritePaths=/home/root/.tailscale

[Install]
WantedBy=multi-user.target
`

func runInstall() error {
	fmt.Println("Installing goMarkableStream as a systemd service...")

	if err := ensureConfigDir(); err != nil {
		return err
	}

	if err := createEnvFile(); err != nil {
		return err
	}

	if err := createServiceFile(); err != nil {
		return err
	}

	if err := runSystemctlCommands(); err != nil {
		return err
	}

	fmt.Println("\nInstallation complete!")
	fmt.Println("  - Service file: " + serviceFilePath)
	fmt.Println("  - Environment file: " + envFilePath)
	fmt.Println("\nUseful commands:")
	fmt.Println("  systemctl status goMarkableStream")
	fmt.Println("  journalctl -u goMarkableStream -f")
	return nil
}

func ensureConfigDir() error {
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		fmt.Printf("Creating config directory: %s\n", configDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	return nil
}

func createEnvFile() error {
	if fileExists(envFilePath) {
		fmt.Printf("Environment file already exists (skipping): %s\n", envFilePath)
		return nil
	}

	fmt.Printf("Creating environment file: %s\n", envFilePath)
	if err := os.WriteFile(envFilePath, []byte(envFileContent), 0644); err != nil {
		return fmt.Errorf("failed to create environment file: %w", err)
	}
	return nil
}

func createServiceFile() error {
	if fileExists(serviceFilePath) {
		fmt.Printf("Service file already exists (skipping): %s\n", serviceFilePath)
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	serviceContent := fmt.Sprintf(serviceFileTemplate, execPath)

	fmt.Printf("Creating service file: %s\n", serviceFilePath)
	if err := os.WriteFile(serviceFilePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	return nil
}

func runSystemctlCommands() error {
	commands := []struct {
		name string
		args []string
	}{
		{"Reloading systemd daemon", []string{"systemctl", "daemon-reload"}},
		{"Enabling service", []string{"systemctl", "enable", "goMarkableStream.service"}},
		{"Starting service", []string{"systemctl", "start", "goMarkableStream.service"}},
	}

	for _, cmd := range commands {
		fmt.Printf("%s...\n", cmd.name)
		c := exec.Command(cmd.args[0], cmd.args[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", cmd.name, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
