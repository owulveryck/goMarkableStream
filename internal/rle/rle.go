package rle

import (
	"io"
	"net/http"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

var encodedPool = sync.Pool{
	New: func() interface{} {
		return make([]uint8, 0, remarkable.ScreenHeight*remarkable.ScreenWidth)
	},
}

// NewRLE creates a default RLE
func NewRLE(w io.Writer) *RLE {
	return &RLE{
		sub: w,
	}
}

// RLE implements an io.Writer that implements the Run Length Encoder
type RLE struct {
	sub io.Writer
}

// Write encodes the data using run-length encoding (RLE) and writes the results to the subwriter.
//
// The data parameter is expected to be in the format []uint4, but is passed as []byte.
// The result is packed before being written to the subwriter. The packing scheme
// combines the count and value into a single uint8, with the count ranging from 0 to 15.
//
// Implements: io.Writer
func (rlewriter *RLE) Write(data []byte) (n int, err error) {
	length := len(data)
	if length == 0 {
		return 0, nil
	}
	encoded := encodedPool.Get().([]uint8) // Borrow a slice from the pool
	defer encodedPool.Put(encoded)

	current := data[0]
	count := 0

	for _, datum := range data {
		if count < 254 && datum == current {
			count++
		} else {
			encoded = append(encoded, uint8(count))
			encoded = append(encoded, uint8(current))
			current = datum
			count = 1
		}
	}

	encoded = append(encoded, uint8(count))
	encoded = append(encoded, uint8(current))

	n, err = rlewriter.sub.Write(encoded)
	if err != nil {
		return
	}
	if flusher, ok := rlewriter.sub.(http.Flusher); ok {
		flusher.Flush()
	}
	return
}
