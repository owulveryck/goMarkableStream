package main

import (
	"compress/zlib"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"time"

	iop "github.com/gogo/protobuf/io"
	"github.com/owulveryck/remarkable_screenshot/message"
)

const (
	screenWidth  = 1872
	screenHeight = 1404
	fbAddress    = 4387048
)

func main() {

	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	addr, err := getPointer(file, fbAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Address is: ", addr)
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	log.Println("listening on tcp 2000")
	l, err := net.Listen("udp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	pixels := make([]byte, screenHeight*screenWidth)
	tick := time.NewTicker(100 * time.Millisecond)
	imgP := &message.Image{
		Width:     screenWidth,
		Height:    screenHeight,
		ImageData: pixels,
	}
	for {
		// Wait for a connection.
		func() {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			w := zlib.NewWriter(conn)
			pbWriter := iop.NewDelimitedWriter(w)
			for ; ; <-tick.C {
				_, err := file.ReadAt(pixels, addr)
				if err != nil {
					log.Fatal(err)
				}
				now := time.Now()
				err = pbWriter.WriteMsg(imgP)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(time.Since(now))
			}
		}()
	}
}
func getPointer(r io.ReaderAt, offset int64) (int64, error) {
	pointer := make([]byte, 4)
	_, err := r.ReadAt(pointer, offset)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint32(pointer)), nil
}
