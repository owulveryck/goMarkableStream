package client

import (
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPalette(t *testing.T) {
	yellow := color.RGBA{
		R: 227,
		G: 227,
		B: 45,
		A: 255,
	}
	f, err := os.Open("testdata/screenshot.raw")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var img image.Gray
	err = dec.Decode(&img)
	if err != nil {
		t.Fatal(err)
	}
	// Create mask for highlighting
	maskHighlight := image.NewAlpha(img.Bounds())
	maskBlack := image.NewAlpha(img.Bounds())
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		yp := (y - img.Rect.Min.Y) * img.Stride
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			r := img.Pix[yp+(x-img.Rect.Min.X)]
			if r <= 250 && r > 110 {
				maskHighlight.Pix[yp+(x-img.Rect.Min.X)] = uint8(255 - r)
			} else {
				maskHighlight.Pix[yp+(x-img.Rect.Min.X)] = 255
				maskBlack.Pix[yp+(x-img.Rect.Min.X)] = uint8(255 - r)
			}
		}
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), image.NewUniform(yellow), image.Point{}, draw.Src)

	draw.DrawMask(m, img.Bounds(), &img, image.Point{}, maskHighlight, image.Point{}, draw.Over)
	/*
		palette := color.Palette([]color.Color{
			color.Opaque, color.Transparent,
		})
		palettedImg := image.NewPaletted(img.Bounds(), palette)
		//draw.DrawMask(palettedImg, img.Bounds(), image.NewUniform(color.Transparent), image.Point{}, &img, image.Point{}, draw.Src)
		draw.Draw(palettedImg, palettedImg.Bounds(), &img, image.Point{}, draw.Over)
	*/
	outputF := filepath.Join(os.TempDir(), "test.png")
	fmt.Println(outputF)
	output, err := os.Create(outputF)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	err = png.Encode(output, m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(outputF)
}
