//go:build trace

package trace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	// Create temporary directory for tests
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "both",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize trace system: %v", err)
	}

	if !Enabled {
		t.Error("Trace system should be enabled after initialization")
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Errorf("Trace directory was not created: %s", tmpDir)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test Start
	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start tracing: %v", err)
	}

	if !IsActive() {
		t.Error("Tracing should be active after Start")
	}

	// Record some spans
	for i := 0; i < 10; i++ {
		span := BeginSpan("test_operation")
		time.Sleep(1 * time.Millisecond)
		EndSpan(span, map[string]any{"iteration": i})
	}

	// Test Stop
	files, err := Stop()
	if err != nil {
		t.Fatalf("Failed to stop tracing: %v", err)
	}

	if IsActive() {
		t.Error("Tracing should not be active after Stop")
	}

	if len(files) == 0 {
		t.Error("Expected at least one trace file")
	}

	// Verify file exists
	for _, file := range files {
		fullPath := filepath.Join(tmpDir, filepath.Base(file))
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Trace file does not exist: %s", fullPath)
		}
	}
}

func TestStartWithInvalidMode(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "both",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test invalid mode
	err := Start("invalid_mode")
	if err == nil {
		t.Error("Start should fail with invalid mode")
	}
}

func TestDoubleStart(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Try to start again
	err := Start("spans")
	if err == nil {
		t.Error("Second Start should fail when already active")
	}

	// Cleanup
	Stop()
}

func TestStopWhenNotActive(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Try to stop without starting
	_, err := Stop()
	if err == nil {
		t.Error("Stop should fail when not active")
	}
}

func TestGetStatus(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Check initial status
	status := GetStatus()
	if !status.Enabled {
		t.Error("Status should show enabled")
	}
	if status.Active {
		t.Error("Status should show not active initially")
	}

	// Start tracing
	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Check active status
	status = GetStatus()
	if !status.Active {
		t.Error("Status should show active after Start")
	}
	if status.Mode != "spans" {
		t.Errorf("Expected mode 'spans', got '%s'", status.Mode)
	}

	// Cleanup
	Stop()
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Initially no files
	files, err := ListFiles()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	if len(files) != 0 {
		t.Error("Expected no files initially")
	}

	// Create a trace
	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	span := BeginSpan("test")
	EndSpan(span, nil)

	if _, err := Stop(); err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	// Should have files now
	files, err = ListFiles()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	if len(files) == 0 {
		t.Error("Expected at least one file after tracing")
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create a trace file
	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	span := BeginSpan("test")
	EndSpan(span, nil)
	files, err := Stop()
	if err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No trace files created")
	}

	// Delete the file
	filename := filepath.Base(files[0])
	if err := DeleteFile(filename); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}
}

func TestDeleteFileWithPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Try path traversal attack
	err := DeleteFile("../../../etc/passwd")
	if err == nil {
		t.Error("DeleteFile should reject path traversal attempts")
	}
}

func TestDisabledTracing(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   false,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Try to start when disabled
	err := Start("spans")
	if err == nil {
		t.Error("Start should fail when tracing is disabled")
	}

	// Spans should be no-ops
	span := BeginSpan("test")
	if span != nil {
		t.Error("BeginSpan should return nil when disabled")
	}
	EndSpan(span, nil) // Should not panic
}
