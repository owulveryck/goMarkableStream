//go:build linux && (arm || arm64)

package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest"
	maxRetries   = 3
	retryDelay   = 2 * time.Second
)

// GitHubRelease represents a GitHub release from the API
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset represents an asset in a GitHub release
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// runDownload is the main entry point for the download subcommand
func runDownload() error {
	fmt.Println("goMarkableStream Download")
	fmt.Println("=========================")

	// Get current binary checksum
	fmt.Println("Computing checksum of current binary...")
	currentChecksum, err := getCurrentBinaryChecksum()
	if err != nil {
		return fmt.Errorf("failed to compute current binary checksum: %w", err)
	}
	fmt.Printf("Current binary checksum: %s\n", currentChecksum[:16]+"...")

	// Get latest release from GitHub
	fmt.Println("Checking latest release...")
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	fmt.Printf("Latest release: %s\n", release.TagName)

	// Find checksums.txt in the release assets
	var checksumsURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumsURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumsURL == "" {
		return fmt.Errorf("checksums.txt not found in release %s", release.TagName)
	}

	// Download checksums
	fmt.Println("Downloading checksums...")
	checksumsData, err := downloadFile(checksumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Check if current checksum matches any entry in checksums.txt
	found, binaryName := findCurrentBinaryInChecksums(currentChecksum, string(checksumsData))
	if found {
		fmt.Printf("\nYou are running the latest version (matches %s)\n", binaryName)
		return nil
	}

	// Current checksum differs from all entries in the release
	fmt.Printf("\nThere is a latest version with a different checksum (tagged %s).\n", release.TagName)
	if !promptYesNo("Do you want to download it?") {
		fmt.Println("Aborted.")
		return nil
	}

	// List available binaries (excluding checksums.txt)
	var availableBinaries []GitHubAsset
	for _, asset := range release.Assets {
		if asset.Name != "checksums.txt" {
			availableBinaries = append(availableBinaries, asset)
		}
	}

	if len(availableBinaries) == 0 {
		return fmt.Errorf("no binaries found in release %s", release.TagName)
	}

	// Prompt user to select which binary to download
	fmt.Println("\nAvailable binaries:")
	for i, asset := range availableBinaries {
		fmt.Printf("  %d. %s\n", i+1, asset.Name)
	}

	selectedIndex := promptBinarySelection(len(availableBinaries))
	selectedAsset := availableBinaries[selectedIndex]

	// Download the selected binary
	fmt.Printf("\nDownloading %s...\n", selectedAsset.Name)
	binaryData, err := downloadFile(selectedAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	fmt.Printf("Downloaded %d bytes\n", len(binaryData))

	// Verify checksum
	fmt.Println("Verifying checksum...")
	if err := verifyChecksum(binaryData, string(checksumsData), selectedAsset.Name); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Println("Downloaded and verified successfully.")

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Ask user if they want to replace the current binary
	if !promptYesNo("Replace current binary with downloaded version?") {
		// Save to current directory instead
		downloadPath := filepath.Join(filepath.Dir(execPath), selectedAsset.Name)
		if err := os.WriteFile(downloadPath, binaryData, 0755); err != nil {
			return fmt.Errorf("failed to save downloaded binary: %w", err)
		}
		fmt.Printf("\nDownloaded binary saved to: %s\n", downloadPath)
		return nil
	}

	// Replace the binary
	fmt.Println("Installing update...")
	if err := replaceBinary(execPath, binaryData); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("\nSuccessfully updated to %s\n", release.TagName)

	// Check if running as systemd service and offer restart
	if isSystemdService() {
		fmt.Println("\nDetected systemd service. To apply the update, restart the service:")
		fmt.Println("  systemctl restart gomarkablestream")
	} else {
		fmt.Println("\nPlease restart the application to use the new version.")
	}

	return nil
}

// getCurrentBinaryChecksum computes the SHA256 checksum of the currently running executable
func getCurrentBinaryChecksum() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}

	f, err := os.Open(execPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// findCurrentBinaryInChecksums checks if the current checksum matches any entry in checksums.txt
// Returns true and the binary name if found, false and empty string otherwise
func findCurrentBinaryInChecksums(currentSum, checksums string) (found bool, binaryName string) {
	scanner := bufio.NewScanner(strings.NewReader(checksums))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 && parts[0] == currentSum {
			return true, parts[1]
		}
	}
	return false, ""
}

// promptYesNo prompts the user with a yes/no question and returns true if they answer yes
func promptYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", question)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// promptBinarySelection prompts the user to select a binary by number and returns the 0-based index
func promptBinarySelection(count int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nSelect binary to download [1-%d]: ", count)
		response, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		response = strings.TrimSpace(response)
		num, err := strconv.Atoi(response)
		if err != nil || num < 1 || num > count {
			fmt.Printf("Please enter a number between 1 and %d\n", count)
			continue
		}
		return num - 1
	}
}

// getLatestRelease queries GitHub API for the latest release
func getLatestRelease() (*GitHubRelease, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		release, err := fetchRelease()
		if err == nil {
			return release, nil
		}
		lastErr = err

		// Check for rate limiting
		if strings.Contains(err.Error(), "403") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait a few minutes and try again")
		}

		if attempt < maxRetries {
			fmt.Printf("Retry %d/%d after error: %v\n", attempt, maxRetries, err)
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}
	return nil, lastErr
}

func fetchRelease() (*GitHubRelease, error) {
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "goMarkableStream-updater")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// downloadFile downloads a file from the given URL with retry logic
func downloadFile(url string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		data, err := fetchURL(url)
		if err == nil {
			return data, nil
		}
		lastErr = err

		if attempt < maxRetries {
			fmt.Printf("Retry %d/%d: %v\n", attempt, maxRetries, err)
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}
	return nil, lastErr
}

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// verifyChecksum verifies the SHA256 checksum of the downloaded data
func verifyChecksum(data []byte, checksums, binaryName string) error {
	// Calculate SHA256 of downloaded data
	hash := sha256.Sum256(data)
	actualSum := hex.EncodeToString(hash[:])

	// Find expected checksum in checksums file
	scanner := bufio.NewScanner(strings.NewReader(checksums))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			expectedSum := parts[0]
			filename := parts[1]
			// Handle both "checksum  filename" and "checksum filename" formats
			if strings.HasSuffix(filename, binaryName) || filename == binaryName {
				if actualSum == expectedSum {
					return nil
				}
				return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSum, actualSum)
			}
		}
	}

	return fmt.Errorf("checksum for %s not found in checksums.txt", binaryName)
}

// replaceBinary safely replaces the current binary with the new one
func replaceBinary(execPath string, newData []byte) error {
	dir := filepath.Dir(execPath)
	base := filepath.Base(execPath)

	// Create temp file in same directory (ensures same filesystem for rename)
	tmpFile, err := os.CreateTemp(dir, base+".new.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write new binary to temp file
	if _, err := tmpFile.Write(newData); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Make new binary executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Create backup path
	backupPath := execPath + ".old"

	// Remove any existing backup
	os.Remove(backupPath)

	// Rename current binary to backup
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Rename new binary to original name
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore backup
		if restoreErr := os.Rename(backupPath, execPath); restoreErr != nil {
			return fmt.Errorf("failed to install new binary: %w (also failed to restore backup: %v)", err, restoreErr)
		}
		return fmt.Errorf("failed to install new binary (restored backup): %w", err)
	}

	// Remove backup on success
	os.Remove(backupPath)

	return nil
}

// isSystemdService checks if the application is running as a systemd service
func isSystemdService() bool {
	// Check for INVOCATION_ID which is set by systemd for services
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}

	// Check if gomarkablestream service exists
	cmd := exec.Command("systemctl", "is-active", "--quiet", "gomarkablestream")
	err := cmd.Run()
	return err == nil
}

// restartService restarts the systemd service
func restartService() error {
	cmd := exec.Command("systemctl", "restart", "gomarkablestream")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}
