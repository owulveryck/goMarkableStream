package main

import (
	"context"
	"log"
	"net"
	"time"

	_ "embed"

	"github.com/owulveryck/goMarkableStream/certs"
	"github.com/owulveryck/goMarkableStream/internal/streammock"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type configuration struct {
	BindAddr string `env:"RK_SERVER_BIND_ADDR,default=:2000"`
	WithTLS  bool   `env:"RK_SERVER_WITHTLS,default=true"`
}

func main() {
	cert, err := certs.GetCertificateWrapper()
	if err != nil {
		log.Fatal(err)
	}
	grpcCreds := &callInfoAuthenticator{credentials.NewTLS(cert.ServerTLSConf)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}

	log.Println("listening on tcp " + c.BindAddr)
	ln, err := net.Listen("tcp", c.BindAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	s := streammock.NewServer()
	s.Start()
	var grpcServer *grpc.Server
	if c.WithTLS {
		grpcServer = grpc.NewServer(grpc.Creds(grpcCreds))
	} else {
		grpcServer = grpc.NewServer() // without credentials
	}
	streammock.RegisterStreamServer(grpcServer, s)

	if err := grpcServer.Serve(ln); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
