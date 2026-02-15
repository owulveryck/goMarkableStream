//go:build linux

package remarkable

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// TestEventScannerGoroutineCleanup tests that goroutines terminate when context is cancelled.
// This test verifies Bug #2 fix: goroutines must respect context cancellation.
func TestEventScannerGoroutineCleanup(t *testing.T) {
	// Skip if input devices don't exist (not on reMarkable device)
	if !fileExists(PenInputDevice) || !fileExists(TouchInputDevice) {
		t.Skip("Input devices not available, skipping test")
	}

	// Count initial goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	scanner := NewEventScanner()
	defer scanner.Close()

	ps := pubsub.NewPubSub()
	ctx, cancel := context.WithCancel(context.Background())

	// Start the scanner
	scanner.StartAndPublish(ctx, ps)

	// Give goroutines time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for goroutines to exit
	time.Sleep(500 * time.Millisecond)
	runtime.GC()

	// Check goroutine count returned to baseline (within tolerance)
	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Goroutine leak detected: initial=%d, final=%d (expected <= %d)",
			initialGoroutines, finalGoroutines, initialGoroutines+2)
	}
}

// TestEventScannerClose tests that EventScanner can be closed properly.
// This test verifies Bug #4 fix: file handles must be closeable.
func TestEventScannerClose(t *testing.T) {
	// Skip if input devices don't exist
	if !fileExists(PenInputDevice) || !fileExists(TouchInputDevice) {
		t.Skip("Input devices not available, skipping test")
	}

	scanner := NewEventScanner()

	// Close should not panic
	err := scanner.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Second close should not panic (idempotent)
	err = scanner.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v, want nil", err)
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
