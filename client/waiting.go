package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"time"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

func (l *grabber) setWaitingPicture(ctx context.Context) error {

	x := 50
	y := 50
	var img *image.Gray
	if l.rot.orientation == portrait {
		img = image.NewGray(image.Rect(0, 0, height, width))
	} else {
		img = image.NewGray(image.Rect(0, 0, width, height))
	}
	draw.Draw(img, img.Bounds(), image.Black, image.Point{}, draw.Src)
	label := "Waiting for reMarkable server at " + l.conf.ServerAddr
	fnt, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	face := truetype.NewFace(fnt, &truetype.Options{
		Size: 36,
	})
	// img is now an *image.RGBA
	point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: face, //basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
	var run bool
	tick := time.Tick(500 * time.Millisecond)
	col := color.White
	d.Face = truetype.NewFace(fnt, &truetype.Options{
		Size: 64,
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		case run = <-l.sleep:
		case <-tick:
			if run {
				switch col {
				case color.White:
					col = color.Black
				case color.Black:
					col = color.White
				}
				d.Src = image.NewUniform(col)
				d.Dot = fixed.Point26_6{fixed.Int26_6(height / 2 * 64), fixed.Int26_6(width / 2 * 64)}
				d.DrawString("X")
				var b bytes.Buffer
				err = jpeg.Encode(&b, img, nil)
				if err != nil {
					return err
				}

				err = l.mjpegStream.Update(b.Bytes())
				if err != nil {
					return err
				}
			}
		}
	}
}
