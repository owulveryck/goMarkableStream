package client

import (
	"bytes"
	"context"
	"encoding/gob"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/owulveryck/goMarkableStream/certs"
	"github.com/owulveryck/goMarkableStream/stream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Displayer can display a gray image
type Displayer interface {
	Display(*image.Gray) error
}

// Grabber is the main object
type Grabber struct {
	conf              *Configuration
	displayer         Displayer
	imageC            chan *image.Gray
	rot               *rotation
	sleep             chan bool
	maxPictureGrabbed int // useful for benchmarking
}

// NewGrabber from configuration
func NewGrabber(c *Configuration, d Displayer) *Grabber {
	return &Grabber{
		conf:      c,
		displayer: d,
		imageC:    make(chan *image.Gray),
		rot: &rotation{
			orientation: portrait,
			isActive:    c.AutoRotate,
		},
		sleep: make(chan bool),
	}
}

// Run the grabber
func (g *Grabber) Run(ctx context.Context) error {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		return err
	}
	grpcCreds := credentials.NewTLS(cert.ClientTLSConf)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go g.imageHandler(ctx)
	go func() {
		err := g.setWaitingPicture(ctx)
		if err != nil {
			log.Println(err)
		}
	}()
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

func (g *Grabber) grab(ctx context.Context, conn *grpc.ClientConn) error {

	client := stream.NewStreamClient(conn)
	getImageClient, err := client.GetImage(ctx, &stream.Input{})
	if err != nil {
		return err
	}

	for i := 0; i < g.maxPictureGrabbed || g.maxPictureGrabbed == 0; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			img := grayPool.Get().(*image.Gray)
			response, err := getImageClient.Recv()
			if err != nil {
				return err
			}
			copy(img.Pix, response.ImageData)
			//img.Pix = response.ImageData
			img.Stride = int(response.Width)
			img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
			if g.conf.paperTextureLandscape != nil {
				g.rot.rotate(img)
				texture := g.conf.paperTextureLandscape
				if g.rot.orientation == portrait {
					texture = g.conf.paperTexturePortrait
				}
				dst := cloneImage(texture)
				for i := 0; i < len(img.Pix); i++ {
					if img.Pix[i] != 255 {
						dst.Pix[i] = img.Pix[i]
					}
				}
				g.imageC <- dst
			} else {
				g.rot.rotate(img)
				g.imageC <- img

			}
		}
	}
	return nil
}

// GetGob sends a gob encoded version of the image
func (g *Grabber) GetGob(w http.ResponseWriter, r *http.Request) {
	tick := time.NewTicker(1 * time.Second)
	select {
	case img := <-g.imageC:
		w.Header().Add("Content-Type", "application/octet-stream")
		enc := gob.NewEncoder(w)
		err := enc.Encode(img)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case <-tick.C:
		http.Error(w, "no content", http.StatusNoContent)
		return
	}
}

// GetScreenshot sends a png encoded version of the image currently in the grabber
func (g *Grabber) GetScreenshot(w http.ResponseWriter, r *http.Request) {
	tick := time.NewTicker(1 * time.Second)
	select {
	case img := <-g.imageC:
		m := createTransparentImage(img)
		w.Header().Add("Content-Type", "image/png")
		if err := png.Encode(w, m); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case <-tick.C:
		http.Error(w, "no content", http.StatusNoContent)
		return
	}
}

// GetRaw representation of the bitmap image
func (g *Grabber) GetRaw(w http.ResponseWriter, r *http.Request) {
	tick := time.NewTicker(1 * time.Second)
	select {
	case img := <-g.imageC:
		w.Header().Add("Cntent-Type", "application/octet-stream")
		_, err := io.Copy(w, bytes.NewReader(img.Pix))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case <-tick.C:
		http.Error(w, "no content", http.StatusNoContent)
		return
	}
}
