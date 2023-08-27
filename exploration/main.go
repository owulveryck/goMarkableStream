package main

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

func main() {
	testdata := "../testdata/full_memory_region.raw"
	stats, _ := os.Stat(testdata)
	f, err := os.Open(testdata)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	backend := make([]uint8, stats.Size())
	log.Printf("backend is %v bytes", len(backend))
	n, err := f.ReadAt(backend, 0)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("read %v bytes from the file", n)
	picture := backend[8 : 1872*1404*2+8]
	boundaries := image.Rectangle{
		Min: image.Point{
			X: 0,
			Y: 0,
		},
		Max: image.Point{
			X: remarkable.ScreenWidth,
			Y: remarkable.ScreenHeight,
		},
	}
	img := image.NewGray(boundaries)
	w := remarkable.ScreenWidth
	h := remarkable.ScreenHeight
	unflipAndExtract(picture, img.Pix, w, h)

	png.Encode(os.Stdout, img)
}
func unflipAndExtract(src, dst []uint8, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIndex := (y*w + x) * 2 // every second byte is useful
			dstIndex := (h-y-1)*w + x // unflip position
			dst[dstIndex] = src[srcIndex] << 4
		}
	}
}
