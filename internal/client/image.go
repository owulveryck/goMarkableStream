package client

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"time"

	"github.com/mattn/go-mjpeg"
)

func (g *Grabber) imageHandler(ctx context.Context) {
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
			err := g.displayer.Display(img)
			//err := g.displayPicture(img)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

type MJPEGDisplayer struct {
	conf        *Configuration
	mjpegStream *mjpeg.Stream
}

func NewMJPEGDisplayer(c *Configuration, stream *mjpeg.Stream) *MJPEGDisplayer {
	return &MJPEGDisplayer{
		conf:        c,
		mjpegStream: stream,
	}
}

func (m *MJPEGDisplayer) Display(img *image.Gray) error {
	var b bytes.Buffer

	var err error
	if m.conf.Colorize {
		err = jpeg.Encode(&b, colorize(img), nil)
	} else {
		err = jpeg.Encode(&b, img, nil)
	}
	if err != nil {
		return err
	}
	err = m.mjpegStream.Update(b.Bytes())
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
	for i := 0; i < len(img.Pix); i++ {
		r := img.Pix[i]
		if r <= 250 && r > 110 {
			maskHighlight.Pix[i] = uint8(255 - r)
		} else {
			maskHighlight.Pix[i] = 255
		}
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), image.NewUniform(yellow), image.Point{}, draw.Src)
	draw.DrawMask(m, img.Bounds(), img, image.Point{}, maskHighlight, image.Point{}, draw.Over)
	return m
}
