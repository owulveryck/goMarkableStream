//go:build linux && arm

package remarkable

import (
	"io"
	"os"
)

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
