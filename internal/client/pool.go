package client

import (
	"bytes"
	"image"
	"sync"
)

func releaseRGBA(img *image.RGBA) {
	switch {
	case img.Rect.Dx() == Width && img.Rect.Dy() == Height:
		rgbaPoolHW.Put(img)
	case img.Rect.Dx() == Height && img.Rect.Dy() == Width:
		rgbaPoolWH.Put(img)
	}
}

var rgbaPoolWH = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return image.NewRGBA(image.Rect(0, 0, Width, Height))
	},
}
var rgbaPoolHW = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return image.NewRGBA(image.Rect(0, 0, Height, Width))
	},
}

var bufPool = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return new(bytes.Buffer)
	},
}

var grayPool = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return image.NewGray(image.Rect(0, 0, Height, Width))
	},
}
