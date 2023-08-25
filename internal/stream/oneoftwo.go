package stream

import (
	"io"
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

func (oneoutoftwo *oneOutOfTwo) Write(p []byte) (n int, err error) {
	imageData := imagePool.Get().([]uint8)
	defer imagePool.Put(imageData) // Return the slice to the pool when done
	for i := 0; i < len(p); i += 2 {
		imageData[i/2] = p[i]
	}
	n, err = oneoutoftwo.w.Write(imageData)
	return
}
