package client

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

const (
	h = 400
	w = 400
)

func BenchmarkJPEGEncodingCMYK(b *testing.B) {
	img := image.NewYCbCr(image.Rect(0, 0, h, w), image.YCbCrSubsampleRatio422)
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		err := jpeg.Encode(&buf, img, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJPEGEncodingGray(b *testing.B) {
	img := image.NewGray(image.Rect(0, 0, h, w))
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		err := jpeg.Encode(&buf, img, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkJPEGEncodingGrayLow(b *testing.B) {
	img := image.NewGray(image.Rect(0, 0, h, w))
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		err := jpeg.Encode(&buf, img, &jpeg.Options{
			Quality: 20,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkJPEGEncodingRGBA(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, h, w))
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		err := jpeg.Encode(&buf, img, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkJPEGEncodingRGBALow(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, h, w))
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 40})
		if err != nil {
			b.Fatal(err)
		}
	}
}
