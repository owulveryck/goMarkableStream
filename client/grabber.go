package main

import (
	"context"
	"encoding/gob"
	"image"
	"image/draw"
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

func (g *grabber) getGob(w http.ResponseWriter, r *http.Request) {
	tick := time.Tick(1 * time.Second)
	select {
	case img := <-g.imageC:
		w.Header().Add("Content-Type", "application/octet-stream")
		enc := gob.NewEncoder(w)
		err := enc.Encode(img)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case <-tick:
		http.Error(w, "no content", http.StatusNoContent)
		return
	}
}

func (g *grabber) getScreenshot(w http.ResponseWriter, r *http.Request) {
	tick := time.Tick(1 * time.Second)
	select {
	case img := <-g.imageC:
		m := createTransparentImage(img)
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
