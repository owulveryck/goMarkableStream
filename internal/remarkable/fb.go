//go:build !linux || (!arm && !arm64)

package remarkable

import (
	"io"
	"os"
)

// GetFileAndPointer finds the filedescriptor of the xochitl process and the pointer address of the virtual framebuffer
func GetFileAndPointer() (io.ReaderAt, int64, error) {
	return &dummyPicture{}, 0, nil

}

type dummyPicture struct{}

func (dummypicture *dummyPicture) ReadAt(p []byte, off int64) (n int, err error) {
	f, err := os.Open("./testdata/full_memory_region.raw")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.ReadAt(p, off)
}
