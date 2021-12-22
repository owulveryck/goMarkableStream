package main

import (
	"bufio"
	"compress/zlib"
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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
	BindAddr string `env:"RK_SERVER_BIND_ADDR,default=:2000"`
}

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
	grpcCreds := &callInfoAuthenticator{credentials.NewTLS(cert.ServerTLSConf)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}
	pid := findPid()
	if len(os.Args) == 2 {
		pid = os.Args[1]
	}

	file, err := os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	addr, err := getPointer(pid)
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

func getPointer(pid string) (int64, error) {
	file, err := os.OpenFile("/proc/"+pid+"/maps", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanAddr := false
	var addr int64
	for scanner.Scan() {
		if scanAddr {
			hex := strings.Split(scanner.Text(), "-")[0]
			addr, err = strconv.ParseInt("0x"+hex, 0, 64)
			break
		}
		if scanner.Text() == `/dev/fb0` {
			scanAddr = true
		}
	}
	return addr, err
}
