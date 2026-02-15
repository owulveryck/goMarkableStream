//go:build linux && (arm || arm64)

package remarkable

import (
	"io"
	"os"
)

// FramebufferReader wraps an os.File to provide framebuffer reading with proper cleanup.
type FramebufferReader struct {
	file   *os.File
	closed bool
}

// ReadAt implements io.ReaderAt interface.
func (r *FramebufferReader) ReadAt(p []byte, off int64) (n int, err error) {
	return r.file.ReadAt(p, off)
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
