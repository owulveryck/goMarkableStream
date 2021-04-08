package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"time"

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
	ServerAddr     string `env:"RK_SERVER_ADDR,default=remarkable:2000"`
	BindAddr       string `env:"RK_CLIENT_BIND_ADDR,default=:8080"`
	AutoRotate     bool   `env:"RK_CLIENT_AUTOROTATE,default=true"`
	ScreenShotDest string `env:"RK_CLIENT_SCREENSHOT_DEST,default=."`
}

func main() {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		log.Fatal(err)
	}
	//var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}

	grpcCreds := credentials.NewTLS(cert.ClientTLSConf)
	// Create a connection with the TLS credentials
	conn, err := grpc.Dial(c.ServerAddr, grpc.WithTransportCredentials(grpcCreds), grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))

	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	mjpegStream := mjpeg.NewStream()
	go runGrabber(c, mjpegStream, conn)
	mux := http.NewServeMux()
	mux.HandleFunc("/video", makeGzipHandler(mjpegStream))
	log.Printf("listening on %v, registered /video", c.BindAddr)
	err = http.ListenAndServe(c.BindAddr, mux)
	if err != nil {
		log.Fatal(err)
	}
}

func runGrabber(c configuration, mjpegStream *mjpeg.Stream, conn *grpc.ClientConn) {
	client := stream.NewStreamClient(conn)
	var err error
	var response *stream.Image

	var img image.Gray
	rot := &rotation{
		orientation: portrait,
		isActive:    c.AutoRotate,
	}
	screenshotC := make(chan struct{})
	imageC := make(chan *image.Gray)
	go screenshotEvent(screenshotC)
	go imageHandler(c, screenshotC, imageC, mjpegStream)
	for err == nil {
		response, err = client.GetImage(context.Background(), &stream.Input{})
		if err != nil {
			log.Fatalf("Error when calling GetImage: %s", err)
		}

		img.Pix = response.ImageData
		img.Stride = int(response.Width)
		img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
		rot.rotate(&img)
		imageC <- &img
	}
}

func imageHandler(conf configuration, screenshotC <-chan struct{}, imageC <-chan *image.Gray, mjpegStream *mjpeg.Stream) {
	for img := range imageC {
		select {
		case <-screenshotC:
			err := savePicture(conf, img)
			if err != nil {
				log.Println(err)
			}
		default:
		}
		err := displayPicture(img, mjpegStream)
		if err != nil {
			log.Fatal(err)
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
