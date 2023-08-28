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

func (oneoutoftwo *oneOutOfTwo) Write(src []byte) (n int, err error) {
	imageData := imagePool.Get().([]uint8)
	defer imagePool.Put(imageData) // Return the slice to the pool when done
	unflipAndExtract(src, imageData, remarkable.ScreenWidth, remarkable.ScreenHeight)
	n, err = oneoutoftwo.w.Write(imageData)
	return
}

func unflipAndExtract(src, dst []uint8, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIndex := (y*w + x) * 2 // every second byte is useful
			dstIndex := (h-y-1)*w + x // unflip position
			dst[dstIndex] = src[srcIndex]
		}
	}
}
