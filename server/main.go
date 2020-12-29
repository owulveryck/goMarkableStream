package main

import (
	"compress/zlib"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"time"

	_ "embed"

	"github.com/owulveryck/goMarkableStream/certs"
	"github.com/owulveryck/goMarkableStream/stream"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
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

func init() {
	err := gzip.SetLevel(zlib.BestSpeed)
	if err != nil {
		panic(err)
	}
}

func main() {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		log.Fatal(err)
	}
	grpcCreds := credentials.NewTLS(cert.ServerTLSConf)
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
	log.Println("listening on tcp " + c.BindAddr)
	ln, err := net.Listen("tcp", c.BindAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	s := stream.NewServer(file, addr)
	s.Start()
	grpcServer := grpc.NewServer(grpc.Creds(grpcCreds))

	stream.RegisterStreamServer(grpcServer, s)

	if err := grpcServer.Serve(ln); err != nil {
		log.Fatalf("failed to serve: %s", err)
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
