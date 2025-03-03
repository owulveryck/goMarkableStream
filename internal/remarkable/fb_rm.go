//go:build linux && (arm || arm64)

package remarkable

import (
	"io"
	"os"
)

// GetFileAndPointer returns the memory file handle and pointer address for the reMarkable framebuffer
func GetFileAndPointer() (io.ReaderAt, int64, error) {
	pid := findXochitlPID()
	file, err := os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return file, 0, err
	}
	pointerAddr, err := getFramePointer(pid)
	if err != nil {
		return file, 0, err
	}
	return file, pointerAddr, nil
}
