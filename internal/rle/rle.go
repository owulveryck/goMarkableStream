package rle

import (
	"bytes"
	"io"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

var encodedPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, remarkable.ScreenSize)
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
func (rlewriter *RLE) Write(data []byte) (int, error) {
	length := len(data)
	if length == 0 {
		return 0, nil
	}
	buf := bufferPool.Get().([]uint8)
	defer bufferPool.Put(buf)

	current := data[0]
	count := uint8(0)

	for i := 0; i < remarkable.ScreenSize; i += 2 {
		datum := data[i]
		if count < 254 && datum == current {
			count++
		} else {
			buf = append(buf, count)
			buf = append(buf, current)
			current = datum
			count = 1
		}
	}
	/*
		for i := 0; i < remarkable.ScreenWidth*remarkable.ScreenHeight; i++ {
			datum := data[i*2]
			if count < 254 && datum == current {
				count++
			} else {
				buf = append(buf, count)
				buf = append(buf, current)
				current = datum
				count = 1
			}
		}
	*/
	buf = append(buf, count)
	buf = append(buf, current)

	return rlewriter.sub.Write(buf)
}
