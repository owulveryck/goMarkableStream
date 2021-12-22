package client

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

func (g *Grabber) setWaitingPicture(ctx context.Context) error {

	x := 50
	y := 50
	var img *image.Gray

	var run bool
	tick := time.NewTicker(500 * time.Millisecond)
	var d *font.Drawer
	var col color.Color
	for {
		select {
		case <-ctx.Done():
			return nil
		case run = <-g.sleep:
			if g.rot.orientation == portrait {
				img = image.NewGray(image.Rect(0, 0, Height, Width))
			} else {
				img = image.NewGray(image.Rect(0, 0, Width, Height))
			}
			draw.Draw(img, img.Bounds(), image.Black, image.Point{}, draw.Src)
			label := "Waiting for reMarkable server at " + g.conf.ServerAddr
			fnt, err := truetype.Parse(goregular.TTF)
			if err != nil {
				return err
			}
			face := truetype.NewFace(fnt, &truetype.Options{
				Size: 36,
			})
			// img is now an *image.RGBA
			point := fixed.Point26_6{
				X: fixed.Int26_6(x * 64),
				Y: fixed.Int26_6(y * 64),
			}

			d = &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(color.White),
				Face: face, //basicfont.Face7x13,
				Dot:  point,
			}
			d.DrawString(label)
			col = color.White
			d.Face = truetype.NewFace(fnt, &truetype.Options{
				Size: 64,
			})
		case <-tick.C:
			if run {
				switch col {
				case color.White:
					col = color.Black
				case color.Black:
					col = color.White
				}
				d.Src = image.NewUniform(col)
				d.Dot = fixed.Point26_6{
					X: fixed.Int26_6(Height / 2 * 64),
					Y: fixed.Int26_6(Width / 2 * 64),
				}
				d.DrawString("X")
				var b bytes.Buffer
				err := jpeg.Encode(&b, img, nil)
				if err != nil {
					return err
				}
				err = g.displayer.Display(img)
				if err != nil {
					return err
				}
			}
		}
	}
}

// Renderer ...
type Renderer interface {
	Update([]byte) error
}
