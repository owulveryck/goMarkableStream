package main

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"time"

	"github.com/mattn/go-mjpeg"
)

func (g *grabber) imageHandler(ctx context.Context) {
	idle := 2 * time.Second
	sleep := false
	tick := time.NewTicker(idle)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if !sleep {
				g.sleep <- true
			}
			sleep = true
		case img := <-g.imageC:
			if sleep {
				g.sleep <- false
				sleep = false
			}
			tick.Reset(idle)
			err := displayPicture(img, g.mjpegStream)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func displayPicture(img *image.Gray, mjpegStream *mjpeg.Stream) error {
	var b bytes.Buffer

	err := jpeg.Encode(&b, img, nil)
	if err != nil {
		return err
	}
	err = mjpegStream.Update(b.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func createTransparentImage(img *image.Gray) *image.RGBA {
	mask := image.NewAlpha(img.Bounds())
	//Direct pixel access for performance
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		yp := (y - img.Rect.Min.Y) * img.Stride
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			r := img.Pix[yp+(x-img.Rect.Min.X)]
			mask.Pix[yp+(x-img.Rect.Min.X)] = uint8(255 - r)
		}
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)

	draw.DrawMask(m, img.Bounds(), img, image.Point{}, mask, image.Point{}, draw.Over)
	return m
}
