package main

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

func main() {
	palette := make(map[uint8]int64)
	spectre := make(map[uint8]int64)
	//testdata := "../testdata/full_memory_region.raw"
	testdata := "../testdata/multi.raw"
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
	for i := 0; i < len(picture); i += 2 {
		spectre[picture[i]]++
	}
	for _, v := range img.Pix {
		palette[v]++
	}
	log.Println(spectre)
	log.Println(palette)

	png.Encode(os.Stdout, img)
}
func unflipAndExtract(src, dst []uint8, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIndex := (y*w + x) * 2 // every second byte is useful
			dstIndex := (h-y-1)*w + x // unflip position
			dst[dstIndex] = src[srcIndex] * 17
		}
	}
}
