package main

import (
	"compress/zlib"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/mattn/go-mjpeg"
	"github.com/owulveryck/goMarkableStream/internal/client"
	"github.com/sethvargo/go-envconfig"
	grpcGzip "google.golang.org/grpc/encoding/gzip"
)

func init() {
	err := grpcGzip.SetLevel(zlib.BestSpeed)
	if err != nil {
		panic(err)
	}
}

func main() {
	ctx := context.Background()
	var c client.Configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}
	err := client.ProcessTexture(&c)
	if err != nil {
		log.Println("Cannot process texture, ", err)
	}

	mjpegStream := mjpeg.NewStream()
	displayer := client.NewMJPEGDisplayer(&c, mjpegStream)
	g := client.NewGrabber(&c, displayer)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, index)
	})
	mux.HandleFunc("/favicon.ico", faviconHandler)
	mux.HandleFunc("/screenshot", g.GetScreenshot)
	mux.HandleFunc("/orientation", g.Rotate)
	mux.Handle("/conf", &c)
	mux.HandleFunc("/gob", g.GetGob)
	mux.HandleFunc("/raw", g.GetRaw)
	mux.HandleFunc("/video", makeGzipHandler(mjpegStream))
	log.Printf("listening on %v", c.BindAddr)
	go func() {
		err = http.ListenAndServe(c.BindAddr, mux)
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = g.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
