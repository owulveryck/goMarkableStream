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
		// Typical RLE-compressed RM1 frame is ~200-500 KB; start with 256 KB
		// capacity rather than pre-allocating a full 5.27 MB / 10.54 MB slice.
		return make([]byte, 0, 256*1024)
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

// Flush propagates the flush to the underlying writer if it supports it.
// This ensures data is sent to the HTTP client promptly.
func (rlewriter *RLE) Flush() {
	type flusher interface {
		Flush()
	}
	if f, ok := rlewriter.sub.(flusher); ok {
		f.Flush()
	}
}

// Write encodes the data using run-length encoding (RLE) and writes the results to the subwriter.
//
// Reads every even byte (stride 2) and emits [count, value] pairs with count in 1..255.
//
// Implements: io.Writer
func (rlewriter *RLE) Write(data []byte) (int, error) {
	length := len(data)
	if length == 0 {
		return 0, nil
	}
	buf := bufferPool.Get().([]uint8)
	buf = buf[:0]
	defer func() {
		bufferPool.Put(buf)
	}()

	limit := length
	if remarkable.Config.SizeBytes > 0 && remarkable.Config.SizeBytes < limit {
		limit = remarkable.Config.SizeBytes
	}

	current := data[0]
	count := uint8(0)

	for i := 0; i < limit; i += 2 {
		datum := data[i]
		if count < 254 && datum == current {
			count++
		} else {
			buf = append(buf, count, current)
			current = datum
			count = 1
		}
	}
	buf = append(buf, count, current)

	return rlewriter.sub.Write(buf)
}
