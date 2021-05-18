package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"

	"github.com/mattn/go-mjpeg"
	"github.com/owulveryck/goMarkableStream/certs"
	"github.com/owulveryck/goMarkableStream/stream"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcGzip "google.golang.org/grpc/encoding/gzip"
)

func init() {
	err := grpcGzip.SetLevel(zlib.BestSpeed)
	if err != nil {
		panic(err)
	}
}

type configuration struct {
	ServerAddr            string `env:"RK_SERVER_ADDR,default=remarkable:2000"`
	BindAddr              string `env:"RK_CLIENT_BIND_ADDR,default=:8080"`
	AutoRotate            bool   `env:"RK_CLIENT_AUTOROTATE,default=true"`
	ScreenShotDest        string `env:"RK_CLIENT_SCREENSHOT_DEST,default=."`
	PaperTexture          string `env:"RK_CLIENT_PAPER_TEXTURE"`
	paperTextureLandscape *image.Gray
	paperTexturePortrait  *image.Gray
}

func main() {
	//var d net.Dialer
	ctx := context.Background()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}
	err := processTexture(&c)
	if err != nil {
		log.Println("Cannot process texture, ", err)
	}

	mjpegStream := mjpeg.NewStream()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, index)
	})
	mux.HandleFunc("/video", makeGzipHandler(mjpegStream))
	log.Printf("listening on %v", c.BindAddr)
	go func() {
		err = http.ListenAndServe(c.BindAddr, mux)
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = runLoop(ctx, c, mjpegStream)
	if err != nil {
		log.Fatal(err)
	}
}

func runLoop(ctx context.Context, c configuration, mjpegStream *mjpeg.Stream) error {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		return err
	}
	grpcCreds := credentials.NewTLS(cert.ClientTLSConf)
	for {
		// Create a connection with the TLS credentials
		conn, err := grpc.DialContext(ctx, c.ServerAddr, grpc.WithTransportCredentials(grpcCreds), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("Connection established")
		err = runGrabber(ctx, c, mjpegStream, conn)
		if err != nil {
			conn.Close()
			log.Println("cannot grab picture", err)
		}
	}
	return nil
}

func runGrabber(ctx context.Context, c configuration, mjpegStream *mjpeg.Stream, conn *grpc.ClientConn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := stream.NewStreamClient(conn)
	getImageClient, err := client.GetImage(ctx, &stream.Input{})
	if err != nil {
		return err
	}

	var img image.Gray
	rot := &rotation{
		orientation: portrait,
		isActive:    c.AutoRotate,
	}
	imageC := make(chan *image.Gray)
	screenshotC := make(chan struct{})
	go screenshotEvent(ctx, screenshotC)
	go imageHandler(ctx, c, screenshotC, imageC, mjpegStream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			response, err := getImageClient.Recv()
			if err != nil {
				cancel()
				return err
			}
			img.Pix = response.ImageData
			img.Stride = int(response.Width)
			img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
			if c.paperTextureLandscape != nil {
				rot.rotate(&img)
				texture := c.paperTextureLandscape
				if rot.orientation == portrait {
					texture = c.paperTexturePortrait
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
				imageC <- dst
			} else {
				rot.rotate(&img)
				imageC <- &img

			}
		}
	}
}

func imageHandler(ctx context.Context, conf configuration, screenshotC <-chan struct{}, imageC <-chan *image.Gray, mjpegStream *mjpeg.Stream) {
	for img := range imageC {
		select {
		case <-screenshotC:
			err := savePicture(conf, img)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			return
		default:
		}
		err := displayPicture(img, mjpegStream)
		if err != nil {
			log.Println(err)
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
