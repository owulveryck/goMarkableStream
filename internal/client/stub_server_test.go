package client

import (
	"compress/zlib"
	"io/ioutil"
	"net"

	"github.com/owulveryck/goMarkableStream/stream"
	"golang.org/x/net/nettest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

func init() {
	err := gzip.SetLevel(zlib.BestSpeed)
	if err != nil {
		panic(err)
	}
}

type stub struct {
	img *stream.Image
}

// GetImage input is nil
func (s *stub) GetImage(_ *stream.Input, stream stream.Stream_GetImageServer) error {
	for {
		if err := stream.Send(s.img); err != nil {
			return err
		}
	}
}

func newStub() *stub {
	content, err := ioutil.ReadFile("testdata/screenshot.raw")
	if err != nil {
		panic(err)
	}
	return &stub{
		img: &stream.Image{
			Width:     Width,
			Height:    Height,
			ImageData: content,
		},
	}
}

func startStub() (*grpc.Server, net.Listener) {
	s := newStub()
	grpcServer := grpc.NewServer()

	stream.RegisterStreamServer(grpcServer, s)

	ln, err := nettest.NewLocalListener("tcp")
	if err != nil {
		panic(err)
	}

	go func() {
		if err := grpcServer.Serve(ln); err != nil {
			panic(err)
		}
	}()
	return grpcServer, ln
}
