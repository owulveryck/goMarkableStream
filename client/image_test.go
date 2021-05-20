package main

import (
	"encoding/gob"
	"image"
	"image/color"
	"image/draw"
	"os"
	"testing"
)

func BenchmarkCreateTransparentImage(b *testing.B) {
	f, err := os.Open("testdata/screenshot.raw")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var img image.Gray
	err = dec.Decode(&img)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		createTransparentImage(&img)
	}
}
func BenchmarkCreateTransparentImageLegacy(b *testing.B) {
	f, err := os.Open("testdata/screenshot.raw")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var img image.Gray
	err = dec.Decode(&img)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		mask := image.NewAlpha(img.Bounds())
		//draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)
		for x := 0; x < mask.Rect.Dx(); x++ {
			for y := 0; y < mask.Rect.Dy(); y++ {
				//get one of r, g, b on the mask image ...
				r, _, _, _ := img.At(x, y).RGBA()
				//... and set it as the alpha value on the mask.
				mask.SetAlpha(x, y, color.Alpha{uint8(255 - r)}) //Assuming that white is your transparency, subtract it from 255
			}
		}
		m := image.NewRGBA(img.Bounds())
		draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)

		draw.DrawMask(m, img.Bounds(), &img, image.Point{}, mask, image.Point{}, draw.Over)

	}
}
