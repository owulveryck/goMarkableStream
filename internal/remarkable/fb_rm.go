//go:build linux && (arm || arm64)

package remarkable

import (
	"errors"
	"io"
	"log"
	"os"
	"syscall"

	"github.com/owulveryck/goMarkableStream/internal/delta"
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

// MmapFramebufferReader maps /dev/fb0 read-only into process address space on
// the reMarkable 1. This eliminates ~1,287 4KB page-loop iterations inside the
// Linux kernel's fb_read (_memcpy_fromio + copy_to_user) per frame and lets
// NEON VLD1.8 issue full 32-byte AXI burst reads directly from the mapped
// Write-Combine framebuffer.
type MmapFramebufferReader struct {
	file   *os.File
	data   []byte
	closed bool
}

// Bytes returns the memory-mapped framebuffer slice for zero-copy reading.
func (r *MmapFramebufferReader) Bytes() []byte {
	if r.closed {
		return nil
	}
	return r.data
}

// ReadAt implements io.ReaderAt using NEON burst CopyFrame on ARM32.
func (r *MmapFramebufferReader) ReadAt(p []byte, off int64) (n int, err error) {
	span := trace.BeginSpan("frame_capture")
	defer func() {
		trace.EndSpan(span, map[string]any{
			"bytes_read": n,
			"mmap":       true,
			"error":      err != nil,
		})
	}()

	if off < 0 {
		return 0, errors.New("remarkable: negative offset")
	}
	if r.closed || off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = delta.CopyFrame(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Close unmaps the framebuffer and closes the underlying file handle.
// Safe to call multiple times.
func (r *MmapFramebufferReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	var munmapErr error
	if r.data != nil {
		munmapErr = syscall.Munmap(r.data)
		r.data = nil
	}
	closeErr := r.file.Close()
	if munmapErr != nil {
		return munmapErr
	}
	return closeErr
}

// GetFileAndPointer returns the memory file handle and pointer address for the reMarkable framebuffer.
//   - rM1: maps /dev/fb0 read-only via mmap (falling back to ReadAt if mmap is unavailable)
//   - rM2 and Paper Pro: reads xochitl's process memory at the framebuffer mapping address
func GetFileAndPointer() (io.ReaderAt, int64, error) {
	if Model == Remarkable1 {
		return rm1Framebuffer()
	}
	return xochitlFramebuffer()
}

// rm1Framebuffer opens /dev/fb0 and maps it read-only via mmap for zero-syscall
// NEON burst access, falling back to FramebufferReader if mmap fails.
func rm1Framebuffer() (io.ReaderAt, int64, error) {
	file, err := os.OpenFile("/dev/fb0", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, 0, err
	}
	if Config.SizeBytes > 0 {
		data, mmapErr := syscall.Mmap(int(file.Fd()), 0, Config.SizeBytes, syscall.PROT_READ, syscall.MAP_SHARED)
		if mmapErr == nil {
			log.Printf("Mapped /dev/fb0 via mmap (%d bytes)", len(data))
			return &MmapFramebufferReader{
				file: file,
				data: data,
			}, 0, nil
		}
		log.Printf("Warning: mmap /dev/fb0 failed (%v), falling back to ReadAt", mmapErr)
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
