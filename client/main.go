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
	"net"
	"net/http"
	"strings"
	"time"

	iop "github.com/gogo/protobuf/io"
	"github.com/mattn/go-mjpeg"
	"github.com/owulveryck/goMarkableStream/message"
	"github.com/sethvargo/go-envconfig"
)

type configuration struct {
	ServerAddr string `env:"RK_SERVER_ADDR,default=remarkable:2000"`
	BindAddr   string `env:"RK_CLIENT_BIND_ADDR,default=:8080"`
}

func main() {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}

	stream := mjpeg.NewStream()
	go func(stream *mjpeg.Stream) {
		conn, err := d.DialContext(ctx, "tcp", c.ServerAddr)
		//conn, err := d.DialContext(ctx, "udp", "10.11.99.1:2000")
		if err != nil {
			log.Fatalf("Failed to dial: %v", err)
		}
		defer conn.Close()
		r, err := zlib.NewReader(conn)
		if err != nil {
			log.Fatalf("Failed to dial: %v", err)
		}
		rdr := iop.NewDelimitedReader(r, 1872*1404*2)
		var img image.Gray
		var imgP message.Image
		for rdr.ReadMsg(&imgP); err == nil; err = rdr.ReadMsg(&imgP) {

			var b bytes.Buffer
			img.Pix = imgP.ImageData
			img.Stride = 1872
			img.Rect = image.Rect(0, 0, 1872, 1404)
			err := jpeg.Encode(&b, &img, nil)
			if err != nil {
				log.Fatal(err)
			}
			err = stream.Update(b.Bytes())
			if err != nil {
				log.Fatal(err)
			}
		}
	}(stream)
	mux := http.NewServeMux()
	mux.HandleFunc("/video", makeGzipHandler(stream))
	log.Printf("listening on %v, registered /video", c.BindAddr)
	http.ListenAndServe(c.BindAddr, mux)
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
