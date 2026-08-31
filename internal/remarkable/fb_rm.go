//go:build linux && (arm || arm64)

package remarkable

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/trace"
)

// FramebufferReader wraps an os.File to provide framebuffer reading with proper cleanup.
type FramebufferReader struct {
	file    *os.File
	closed  bool
	mu      sync.Mutex
	scratch []byte
}

// ReadAt implements io.ReaderAt interface.
func (r *FramebufferReader) ReadAt(p []byte, off int64) (n int, err error) {
	span := trace.BeginSpan("frame_capture")
	defer func() {
		trace.EndSpan(span, map[string]any{
			"bytes_read": n,
			"error":      err != nil,
		})
	}()

	if FramebufferStorageWidth == ScreenWidth {
		return r.file.ReadAt(p, off)
	}

	if len(p) != ScreenSizeBytes {
		return 0, fmt.Errorf("padded framebuffer read requires %d-byte destination, got %d", ScreenSizeBytes, len(p))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	storageSize := FramebufferStorageWidth * ScreenHeight * BytesPerPixelBGRA
	if len(r.scratch) != storageSize {
		r.scratch = make([]byte, storageSize)
	}
	if _, err := r.file.ReadAt(r.scratch, off); err != nil {
		return 0, err
	}

	if err := copyVisibleRows(
		p,
		r.scratch,
		ScreenWidth*BytesPerPixelBGRA,
		FramebufferStorageWidth*BytesPerPixelBGRA,
		ScreenHeight,
	); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the underlying file handle. Safe to call multiple times.
func (r *FramebufferReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.file.Close()
}

// GetFileAndPointer returns the memory file handle and pointer address for the reMarkable framebuffer
func GetFileAndPointer() (io.ReaderAt, int64, error) {
	pid, err := findXochitlPID()
	if err != nil {
		return nil, 0, err
	}
	file, err := os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, 0, err
	}
	pointerAddr, err := getFramePointer(pid)
	if err != nil {
		file.Close() // Close file on error
		return nil, 0, err
	}
	return &FramebufferReader{file: file}, pointerAddr, nil
}
