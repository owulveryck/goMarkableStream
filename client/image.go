package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"time"
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
			err := g.displayPicture(img)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (g *grabber) displayPicture(img *image.Gray) error {
	var b bytes.Buffer

	var err error
	if g.conf.Colorize {
		err = jpeg.Encode(&b, colorize(img), nil)
	} else {
		err = jpeg.Encode(&b, img, nil)
	}
	if err != nil {
		return err
	}
	err = g.mjpegStream.Update(b.Bytes())
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

func colorize(img *image.Gray) *image.RGBA {
	yellow := color.RGBA{
		R: 255,
		G: 253,
		B: 84,
		A: 255,
	}
	// Create mask for highlighting
	maskHighlight := image.NewAlpha(img.Bounds())
	//maskBlack := image.NewAlpha(img.Bounds())
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		yp := (y - img.Rect.Min.Y) * img.Stride
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			r := img.Pix[yp+(x-img.Rect.Min.X)]
			if r <= 250 && r > 110 {
				maskHighlight.Pix[yp+(x-img.Rect.Min.X)] = uint8(255 - r)
			} else {
				maskHighlight.Pix[yp+(x-img.Rect.Min.X)] = 255
				//maskBlack.Pix[yp+(x-img.Rect.Min.X)] = uint8(255 - r)
			}
		}
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), image.NewUniform(yellow), image.Point{}, draw.Src)

	draw.DrawMask(m, img.Bounds(), img, image.Point{}, maskHighlight, image.Point{}, draw.Over)
	return m
}
