//go:build linux && (arm || arm64)

package remarkable

import (
	"io"
	"testing"
)

// TestFramebufferReaderImplementsInterfaces verifies that FramebufferReader
// implements both io.ReaderAt and io.Closer interfaces.
func TestFramebufferReaderImplementsInterfaces(t *testing.T) {
	// This test will compile only if FramebufferReader implements the interfaces
	var _ io.ReaderAt = (*FramebufferReader)(nil)
	var _ io.Closer = (*FramebufferReader)(nil)
}

// TestGetFileAndPointerReturnsCloser verifies that GetFileAndPointer returns
// a value that can be closed.
func TestGetFileAndPointerReturnsCloser(t *testing.T) {
	// Skip if not running on actual reMarkable hardware
	_, err := findXochitlPID()
	if err != nil {
		t.Skipf("xochitl process not found: %v", err)
	}

	reader, _, err := GetFileAndPointer()
	if err != nil {
		t.Fatalf("GetFileAndPointer() error = %v", err)
	}

	// Verify it can be closed
	if closer, ok := reader.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	} else {
		t.Error("GetFileAndPointer() returned value is not io.Closer")
	}
}

// TestFramebufferReaderCloseIdempotent verifies that Close can be called multiple times safely.
func TestFramebufferReaderCloseIdempotent(t *testing.T) {
	// Skip if not running on actual reMarkable hardware
	_, err := findXochitlPID()
	if err != nil {
		t.Skipf("xochitl process not found: %v", err)
	}

	reader, _, err := GetFileAndPointer()
	if err != nil {
		t.Fatalf("GetFileAndPointer() error = %v", err)
	}

	closer, ok := reader.(io.Closer)
	if !ok {
		t.Fatal("Reader does not implement io.Closer")
	}

	// First close
	if err := closer.Close(); err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Second close should not panic
	if err := closer.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}
