package client

import (
	"context"
	"image"
	"image/draw"
	"log"
	"time"
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
			grayPool.Put(img)
		}
	}
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
	m := image.NewRGBA(img.Bounds())
	// Create mask for highlighting
	// 85 = red / 153 = blue
	for i := 0; i < len(img.Pix); i++ {
		r := img.Pix[i]
		switch r {
		case 85:
			m.Pix[i*4] = 255
			m.Pix[i*4+1] = 0
			m.Pix[i*4+2] = 0
			m.Pix[i*4+3] = 255
		case 153:
			m.Pix[i*4] = 0
			m.Pix[i*4+1] = 0
			m.Pix[i*4+2] = 255
			m.Pix[i*4+3] = 255
		default:
			m.Pix[i*4] = r
			m.Pix[i*4+1] = r
			m.Pix[i*4+2] = r
			m.Pix[i*4+3] = 255
		}
	}
	return m
}

func highlight(img *image.Gray) *image.RGBA {
	m := image.NewRGBA(img.Bounds())
	// Create mask for highlighting
	maskHighlight := image.NewAlpha(img.Bounds())
	for i := 0; i < len(img.Pix); i++ {
		// Draw a uniform yellow picture
		m.Pix[i*4] = 255
		m.Pix[i*4+1] = 253
		m.Pix[i*4+2] = 84
		m.Pix[i*4+3] = 255
		r := img.Pix[i]
		if r <= 220 && r > 130 {
			maskHighlight.Pix[i] = uint8(255 - r)
		} else {
			maskHighlight.Pix[i] = 255
		}
	}
	err := drawRGBAOver(m, img.Bounds(), img, image.Point{}, maskHighlight, image.Point{})
	if err != nil {
		log.Fatal(err)
	}
	return m
}

// m is the maximum color value returned by image.Color.RGBA.
const m = 1<<16 - 1

// this is a form of the drawRGBA specialised for image.Gray and image.Alpha made for performance reasons
func drawRGBAOver(dst *image.RGBA, r image.Rectangle, src *image.Gray, sp image.Point, mask *image.Alpha, mp image.Point) error {
	x0, x1, dx := r.Min.X, r.Max.X, 1
	y0, y1, dy := r.Min.Y, r.Max.Y, 1
	if r.Overlaps(r.Add(sp.Sub(r.Min))) {
		if sp.Y < r.Min.Y || sp.Y == r.Min.Y && sp.X < r.Min.X {
			x0, x1, dx = x1-1, x0-1, -1
			y0, y1, dy = y1-1, y0-1, -1
		}
	}

	sy := sp.Y + y0 - r.Min.Y
	my := mp.Y + y0 - r.Min.Y
	sx0 := sp.X + x0 - r.Min.X
	mx0 := mp.X + x0 - r.Min.X
	sx1 := sx0 + (x1 - x0)
	i0 := dst.PixOffset(x0, y0)
	di := dx * 4
	for y := y0; y != y1; y, sy, my = y+dy, sy+dy, my+dy {
		for i, sx, mx := i0, sx0, mx0; sx != sx1; i, sx, mx = i+di, sx+dx, mx+dx {
			mi := mask.PixOffset(mx, my)
			ma := uint32(mask.Pix[mi])
			ma |= ma << 8
			si := src.PixOffset(sx, sy)
			sy := uint32(src.Pix[si])
			sy |= sy << 8
			sa := uint32(0xffff)

			d := dst.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
			dr := uint32(d[0])
			dg := uint32(d[1])
			db := uint32(d[2])
			da := uint32(d[3])

			// dr, dg, db and da are all 8-bit color at the moment, ranging in [0,255].
			// We work in 16-bit color, and so would normally do:
			// dr |= dr << 8
			// and similarly for dg, db and da, but instead we multiply a
			// (which is a 16-bit color, ranging in [0,65535]) by 0x101.
			// This yields the same result, but is fewer arithmetic operations.
			a := (m - (sa * ma / m)) * 0x101

			d[0] = uint8((dr*a + sy*ma) / m >> 8)
			d[1] = uint8((dg*a + sy*ma) / m >> 8)
			d[2] = uint8((db*a + sy*ma) / m >> 8)
			d[3] = uint8((da*a + sa*ma) / m >> 8)
		}
		i0 += dy * dst.Stride
	}
	return nil
}
