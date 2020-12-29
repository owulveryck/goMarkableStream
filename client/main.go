package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"strings"
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
	ServerAddr string `env:"RK_SERVER_ADDR,default=remarkable:2000"`
	BindAddr   string `env:"RK_CLIENT_BIND_ADDR,default=:8080"`
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
	client := stream.NewStreamClient(conn)

	mjpegStream := mjpeg.NewStream()
	go func(mjpegStream *mjpeg.Stream) {
		var err error
		var response *stream.Image

		var img image.Gray
		for err == nil {
			response, err = client.GetImage(context.Background(), &stream.Input{})
			if err != nil {
				log.Fatalf("Error when calling GetImage: %s", err)
			}

			var b bytes.Buffer
			img.Pix = response.ImageData
			img.Stride = int(response.Width)
			img.Rect = image.Rect(0, 0, int(response.Width), int(response.Height))
			err := jpeg.Encode(&b, &img, nil)
			if err != nil {
				log.Fatal(err)
			}
			err = mjpegStream.Update(b.Bytes())
			if err != nil {
				log.Fatal(err)
			}
		}
	}(mjpegStream)
	mux := http.NewServeMux()
	mux.HandleFunc("/video", makeGzipHandler(mjpegStream))
	log.Printf("listening on %v, registered /video", c.BindAddr)
	err = http.ListenAndServe(c.BindAddr, mux)
	if err != nil {
		log.Fatal(err)
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	if "" == w.Header().Get("Content-Type") {
		// If no content type, apply sniffing algorithm to un-gzipped body.
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn.ServeHTTP(gzr, r)
	}
}
