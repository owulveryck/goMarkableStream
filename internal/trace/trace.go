//go:build trace

// Package trace provides runtime performance tracing capabilities for goMarkableStream.
// It supports both Go runtime tracing (runtime/trace) and custom application spans.
package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"
	"sync"
)

// Enabled indicates whether the trace system is enabled globally
var Enabled bool

// Status represents the current state of the tracing system
type Status struct {
	Enabled        bool      `json:"enabled"`
	Active         bool      `json:"active"`
	Mode           string    `json:"mode"`            // "runtime", "spans", or "both"
	RuntimeFile    string    `json:"runtime_file"`
	RuntimeSize    int64     `json:"runtime_size"`
	SpansFile      string    `json:"spans_file"`
	SpansSize      int64     `json:"spans_size"`
	TraceFiles     []FileInfo `json:"trace_files"`
}

var (
	mu               sync.Mutex
	active           bool
	currentMode      string
	runtimeTraceFile *os.File
	runtimeFilename  string
)

// Initialize sets up the trace system with the given configuration
func Initialize(cfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	config = cfg
	Enabled = cfg.Enabled

	if !Enabled {
		return nil
	}

	// Create trace directory
	if err := ensureDir(); err != nil {
		return err
	}

	return nil
}

// Start begins tracing with the specified mode
func Start(mode string) error {
	mu.Lock()
	defer mu.Unlock()

	if !Enabled {
		return fmt.Errorf("tracing system is not enabled")
	}

	if active {
		return fmt.Errorf("tracing is already active")
	}

	// Validate mode
	if mode != "runtime" && mode != "spans" && mode != "both" {
		return fmt.Errorf("invalid trace mode: %s (must be 'runtime', 'spans', or 'both')", mode)
	}

	currentMode = mode

	// Start runtime trace if requested
	if mode == "runtime" || mode == "both" {
		if err := startRuntimeTrace(); err != nil {
			return fmt.Errorf("failed to start runtime trace: %w", err)
		}
	}

	// Start span recording if requested
	if mode == "spans" || mode == "both" {
		if err := initSpanRecorder(); err != nil {
			// If runtime trace was started, clean it up
			if runtimeTraceFile != nil {
				trace.Stop()
				runtimeTraceFile.Close()
				runtimeTraceFile = nil
			}
			return fmt.Errorf("failed to start span recorder: %w", err)
		}
	}

	active = true
	return nil
}

// Stop ends the current trace session and returns the generated file paths
func Stop() ([]string, error) {
	mu.Lock()
	defer mu.Unlock()

	if !active {
		return nil, fmt.Errorf("tracing is not active")
	}

	var files []string
	var stopErr error

	// Stop runtime trace
	if runtimeTraceFile != nil {
		trace.Stop()
		if err := runtimeTraceFile.Close(); err != nil {
			stopErr = fmt.Errorf("failed to close runtime trace file: %w", err)
		} else {
			files = append(files, runtimeFilename)
			// Cleanup old runtime trace files
			if err := cleanupOldFiles("runtime"); err != nil {
				fmt.Fprintf(os.Stderr, "trace: failed to cleanup old runtime trace files: %v\n", err)
			}
		}
		runtimeTraceFile = nil
		runtimeFilename = ""
	}

	// Stop span recording
	if recorder != nil {
		spanFile := recorder.filename
		if err := recorder.close(); err != nil {
			if stopErr == nil {
				stopErr = fmt.Errorf("failed to close span recorder: %w", err)
			}
		} else {
			files = append(files, spanFile)
			// Cleanup old span files
			if err := cleanupOldFiles("spans"); err != nil {
				fmt.Fprintf(os.Stderr, "trace: failed to cleanup old span files: %v\n", err)
			}
		}
		recorder = nil
	}

	active = false
	currentMode = ""

	return files, stopErr
}

// IsActive returns true if tracing is currently active
func IsActive() bool {
	mu.Lock()
	defer mu.Unlock()
	return active
}

// GetStatus returns the current tracing system status
func GetStatus() Status {
	mu.Lock()
	defer mu.Unlock()

	status := Status{
		Enabled: Enabled,
		Active:  active,
		Mode:    currentMode,
	}

	// Get runtime trace file info
	if runtimeTraceFile != nil {
		status.RuntimeFile = filepath.Base(runtimeFilename)
		if info, err := os.Stat(runtimeFilename); err == nil {
			status.RuntimeSize = info.Size()
		}
	}

	// Get span file info
	if recorder != nil {
		status.SpansFile = filepath.Base(recorder.filename)
		status.SpansSize = getSpanFileSize()
	}

	// List all trace files
	if files, err := listTraceFiles(); err == nil {
		status.TraceFiles = files
	}

	return status
}

// ListFiles returns information about all available trace files
func ListFiles() ([]FileInfo, error) {
	mu.Lock()
	defer mu.Unlock()

	return listTraceFiles()
}

// DeleteFile removes a trace file by name
func DeleteFile(name string) error {
	mu.Lock()
	defer mu.Unlock()

	// Security check: ensure name doesn't contain path separators
	if filepath.Base(name) != name {
		return fmt.Errorf("invalid file name: must be a simple filename without path")
	}

	// Check if file is currently being written
	if active {
		if (runtimeTraceFile != nil && filepath.Base(runtimeFilename) == name) ||
			(recorder != nil && filepath.Base(recorder.filename) == name) {
			return fmt.Errorf("cannot delete active trace file")
		}
	}

	path := filepath.Join(config.Dir, name)

	// Verify it's a trace file - ensure it's within the trace directory
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(config.Dir)
	if !strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) &&
		cleanPath != cleanDir {
		return fmt.Errorf("invalid file path")
	}

	return os.Remove(path)
}

// GetFilePath returns the absolute path for a trace file name
func GetFilePath(name string) (string, error) {
	// Security check: ensure name doesn't contain path separators
	if filepath.Base(name) != name {
		return "", fmt.Errorf("invalid file name: must be a simple filename without path")
	}

	path := filepath.Join(config.Dir, name)

	// Verify file exists and is within trace directory
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("not a file")
	}

	return path, nil
}

// startRuntimeTrace starts the Go runtime trace
func startRuntimeTrace() error {
	filename := generateFilename("runtime")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err := trace.Start(file); err != nil {
		file.Close()
		os.Remove(filename)
		return err
	}

	runtimeTraceFile = file
	runtimeFilename = filename
	return nil
}
