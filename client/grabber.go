package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"time"

	"github.com/mattn/go-mjpeg"
	"github.com/owulveryck/goMarkableStream/certs"
	"github.com/owulveryck/goMarkableStream/stream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type grabber struct {
	conf        configuration
	mjpegStream *mjpeg.Stream
	imageC      chan *image.Gray
	rot         *rotation
	sleep       chan bool
}

func newGrabber(c configuration, s *mjpeg.Stream) *grabber {
	return &grabber{
		conf:        c,
		mjpegStream: s,
		imageC:      make(chan *image.Gray),
		rot: &rotation{
			orientation: portrait,
			isActive:    c.AutoRotate,
		},
		sleep: make(chan bool),
	}
}

func (g *grabber) run(ctx context.Context) error {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		return err
	}
	grpcCreds := credentials.NewTLS(cert.ClientTLSConf)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go g.imageHandler(ctx)
	go g.setWaitingPicture(ctx)
	for {
		// Create a connection with the TLS credentials
		conn, err := grpc.DialContext(ctx, g.conf.ServerAddr, grpc.WithTransportCredentials(grpcCreds), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("Connection established")
		err = g.grab(ctx, conn)
		if err != nil {
			conn.Close()
			log.Println("cannot grab picture", err)
		}
	}
}

func (g *grabber) grab(ctx context.Context, conn *grpc.ClientConn) error {

	client := stream.NewStreamClient(conn)
	getImageClient, err := client.GetImage(ctx, &stream.Input{})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var img image.Gray
			response, err := getImageClient.Recv()
			if err != nil {
				return err
			}
			img.Pix = make([]uint8, response.Height*response.Width)
			copy(img.Pix, response.ImageData)
			//img.Pix = response.ImageData
			img.Stride = int(response.Width)
			img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
			if g.conf.paperTextureLandscape != nil {
				g.rot.rotate(&img)
				texture := g.conf.paperTextureLandscape
				if g.rot.orientation == portrait {
					texture = g.conf.paperTexturePortrait
				}
				dst := cloneImage(texture)
				for x := 0; x < img.Rect.Dx(); x++ {
					for y := 0; y < img.Rect.Dy(); y++ {
						r, _, _, _ := img.At(x, y).RGBA()
						if r != 65535 {
							dst.Set(x, y, img.At(x, y))
						}
					}
				}
				g.imageC <- dst
			} else {
				g.rot.rotate(&img)
				g.imageC <- &img

			}
		}
	}
}

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

func (g *grabber) getScreenshot(w http.ResponseWriter, r *http.Request) {
	tick := time.Tick(1 * time.Second)
	select {
	case img := <-g.imageC:
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

		draw.DrawMask(m, img.Bounds(), img, image.Point{}, mask, image.Point{}, draw.Over)

		//if err := png.Encode(f, img); err != nil {
		w.Header().Add("Content-Type", "image/png")
		if err := png.Encode(w, m); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case <-tick:
		http.Error(w, "no content", http.StatusNoContent)
		return
	}
}
