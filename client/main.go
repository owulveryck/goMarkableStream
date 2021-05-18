package main

import (
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"log"
	"net/http"

	"github.com/mattn/go-mjpeg"
	"github.com/sethvargo/go-envconfig"
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
	g := newGrabber(c, mjpegStream)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, index)
	})
	mux.HandleFunc("/screenshot", g.getScreenshot)
	mux.HandleFunc("/video", makeGzipHandler(mjpegStream))
	log.Printf("listening on %v", c.BindAddr)
	go func() {
		err = http.ListenAndServe(c.BindAddr, mux)
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = g.run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
