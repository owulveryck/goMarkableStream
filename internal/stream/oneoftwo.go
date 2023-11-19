package stream

import (
	"encoding/binary"
	"io"
	"net/http"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

var imagePool = sync.Pool{
	New: func() any {
		return make([]uint8, remarkable.ScreenWidth*remarkable.ScreenHeight) // Adjust the initial capacity as needed
	},
}

type oneOutOfTwo struct {
	w io.Writer
}

func (oneoutoftwo *oneOutOfTwo) Write(src []byte) (n int, err error) {
	imageData := imagePool.Get().([]uint8)
	defer imagePool.Put(imageData) // Return the slice to the pool when done
	for i := 0; i < remarkable.ScreenHeight*remarkable.ScreenWidth; i++ {
		imageData[i] = uint8(binary.LittleEndian.Uint16(src[i*2 : i*2+2]))
	}
	n, err = oneoutoftwo.w.Write(imageData)
	// If using streaming or chunked responses
	if err != nil {
		return
	}
	if flusher, ok := oneoutoftwo.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return
}
