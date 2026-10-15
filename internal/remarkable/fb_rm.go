//go:build linux && (arm || arm64)

package remarkable

import (
	"io"
	"os"

	"github.com/owulveryck/goMarkableStream/internal/trace"
)

// FramebufferReader wraps an os.File to provide framebuffer reading with proper cleanup.
type FramebufferReader struct {
	file   *os.File
	closed bool
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

// GetFileAndPointer returns the memory file handle and pointer address for the reMarkable framebuffer.
//   - rM1: reads /dev/fb0 directly at offset 0
//   - rM2 and Paper Pro: reads xochitl's process memory at the framebuffer mapping address
func GetFileAndPointer() (io.ReaderAt, int64, error) {
	if Model == Remarkable1 {
		return rm1Framebuffer()
	}
	return xochitlFramebuffer()
}

// rm1Framebuffer opens /dev/fb0 directly. The rM1 exposes its framebuffer as a standard
// Linux framebuffer device that can be read at offset 0.
func rm1Framebuffer() (io.ReaderAt, int64, error) {
	file, err := os.OpenFile("/dev/fb0", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, 0, err
	}
	return &FramebufferReader{file: file}, 0, nil
}

// xochitlFramebuffer reads the framebuffer from xochitl's process memory.
// Used by rM2 and Paper Pro, where the framebuffer is not directly accessible via /dev/fb0.
func xochitlFramebuffer() (io.ReaderAt, int64, error) {
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
