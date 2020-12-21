package main

import (
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	iop "github.com/gogo/protobuf/io"
	"github.com/owulveryck/goMarkableStream/message"
	"github.com/sethvargo/go-envconfig"
)

type configuration struct {
	BindAddr  string `env:"RK_SERVER_BIND_ADDR,default=:2000"`
	FbAddress int    `env:"RK_FB_ADDRESS,default=4387048"`
}

const (
	screenWidth  = 1872
	screenHeight = 1404

//	fbAddress    = 4387048
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}

	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	addr, err := getPointer(file, int64(c.FbAddress))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Memory Address is: ", addr)
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	log.Println("listening on tcp " + c.BindAddr)
	l, err := net.Listen("tcp", c.BindAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	pixels := make([]byte, screenHeight*screenWidth)
	tick := time.NewTicker(200 * time.Millisecond)
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
			w, err := zlib.NewWriterLevel(conn, zlib.BestSpeed)
			if err != nil {
				log.Fatal(err)
			}
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
				fmt.Printf("Time to process %v\r", time.Since(now))
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
