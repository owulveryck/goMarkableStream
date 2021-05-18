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
}

func newGrabber(c configuration, s *mjpeg.Stream) *grabber {
	return &grabber{
		conf:        c,
		mjpegStream: s,
		imageC:      make(chan *image.Gray),
	}
}

func (l *grabber) run(ctx context.Context) error {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		return err
	}
	grpcCreds := credentials.NewTLS(cert.ClientTLSConf)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go l.imageHandler(ctx)
	for {
		// Create a connection with the TLS credentials
		conn, err := grpc.DialContext(ctx, l.conf.ServerAddr, grpc.WithTransportCredentials(grpcCreds), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("Connection established")
		err = l.grab(ctx, conn)
		if err != nil {
			conn.Close()
			log.Println("cannot grab picture", err)
		}
	}
}

func (l *grabber) grab(ctx context.Context, conn *grpc.ClientConn) error {

	client := stream.NewStreamClient(conn)
	getImageClient, err := client.GetImage(ctx, &stream.Input{})
	if err != nil {
		return err
	}

	var img image.Gray
	rot := &rotation{
		orientation: portrait,
		isActive:    l.conf.AutoRotate,
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			response, err := getImageClient.Recv()
			if err != nil {
				return err
			}
			img.Pix = response.ImageData
			img.Stride = int(response.Width)
			img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
			if l.conf.paperTextureLandscape != nil {
				rot.rotate(&img)
				texture := l.conf.paperTextureLandscape
				if rot.orientation == portrait {
					texture = l.conf.paperTexturePortrait
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
				l.imageC <- dst
			} else {
				rot.rotate(&img)
				l.imageC <- &img

			}
		}
	}
}

func (l *grabber) imageHandler(ctx context.Context) {
	for img := range l.imageC {
		select {
		case <-ctx.Done():
			return
		default:
			err := displayPicture(img, l.mjpegStream)
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

func (l *grabber) getScreenshot(w http.ResponseWriter, r *http.Request) {
	tick := time.Tick(1 * time.Second)
	select {
	case img := <-l.imageC:
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
