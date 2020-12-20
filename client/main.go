package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"image"
	"image/jpeg"
	"log"
	"net"
	"net/http"
	"time"

	iop "github.com/gogo/protobuf/io"
	"github.com/mattn/go-mjpeg"
	"github.com/owulveryck/remarkable_screenshot/message"
)

func main() {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	stream := mjpeg.NewStream()
	go func(stream *mjpeg.Stream) {
		conn, err := d.DialContext(ctx, "udp", "192.168.88.192:2000")
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
			log.Println("received data")

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
	mux.Handle("/video", stream)
	http.ListenAndServe(":8080", mux)
}
