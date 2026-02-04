package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/kelseyhightower/envconfig"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

type configuration struct {
	BindAddr       string  `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Username       string  `envconfig:"SERVER_USERNAME" default:"admin"`
	Password       string  `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS            bool    `envconfig:"HTTPS" default:"true"`
	Compression    bool    `envconfig:"COMPRESSION" default:"false"`
	DevMode        bool    `envconfig:"DEV_MODE" default:"false"`
	DeltaThreshold float64 `envconfig:"DELTA_THRESHOLD" default:"0.30" description:"Change ratio threshold (0.0-1.0) above which full frame is sent"`
}

const (
	// ConfigPrefix for environment variable based configuration
	ConfigPrefix = "RK"
)

var (
	pointerAddr int64
	file        io.ReaderAt
	// Define the username and password for authentication
	c configuration

	//go:embed client/*
	assetsFS embed.FS
	//go:embed assets/cert.pem assets/key.pem
	tlsAssets embed.FS
)

func validateConfiguration(c *configuration) error {
	return nil
}

func main() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("not ok")
		return
	}
	fmt.Printf("Version: %s\n", bi.Main.Version)
	var err error

	ifaces()
	help := flag.Bool("h", false, "print usage")
	unsafe := flag.Bool("unsafe", false, "disable authentication")
	flag.Parse()
	if *help {
		envconfig.Usage(ConfigPrefix, &c)
		return
	}
	if err := envconfig.Process(ConfigPrefix, &c); err != nil {
		envconfig.Usage(ConfigPrefix, &c)
		log.Fatal(err)
	}

	if err := validateConfiguration(&c); err != nil {
		panic(err)
	}

	file, pointerAddr, err = remarkable.GetFileAndPointer()
	if err != nil {
		log.Fatal(err)
	}
	eventPublisher := pubsub.NewPubSub()
	eventScanner := remarkable.NewEventScanner()
	eventScanner.StartAndPublish(context.Background(), eventPublisher)

	mux := setMuxer(eventPublisher)

	//	handler := BasicAuthMiddleware(gzMiddleware(mux))
	var handler http.Handler
	handler = BasicAuthMiddleware(mux)
	if *unsafe {
		handler = mux
	}
	l, err := setupListener(context.Background(), &c)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %v", l.Addr())
	if c.TLS {
		log.Fatal(runTLS(l, handler))
	}
	log.Fatal(http.Serve(l, handler))
}
