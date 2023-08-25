package main

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

func main() {
	f, err := os.Open("../testdata/full_memory_region.raw")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	backend := make([]uint8, remarkable.ScreenWidth*remarkable.ScreenHeight*4)
	n, err := f.ReadAt(backend, 0)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("read %v bytes from the file", n)
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
	for i := 0; i < remarkable.ScreenHeight*remarkable.ScreenWidth*2; i += 2 {
		img.Pix[i/2] = backend[i] << 4
	}
	png.Encode(os.Stdout, img)
}
