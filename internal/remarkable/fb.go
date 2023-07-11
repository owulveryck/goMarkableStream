//go:build !linux || !arm

package remarkable

import (
	"io"
	"math/rand"
)

func GetFileAndPointer() (io.ReaderAt, int64, error) {
	return &dummyPicture{}, 0, nil

}

type dummyPicture struct{}

func (dummypicture *dummyPicture) ReadAt(p []byte, off int64) (n int, err error) {
	for i := 0; i < len(p); i++ {
		// Generate a random number between 1 and 10
		num := rand.Intn(10) + 1

		// Set the byte to zero with a probability of 80%
		if num <= 8 {
			p[i] = 255
		} else {
			// Otherwise, generate a random byte between 1 and 15
			p[i] = uint8(rand.Intn(15))
		}
	}
	return len(p), nil
}
