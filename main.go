package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/kelseyhightower/envconfig"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

type configuration struct {
	BindAddr             string `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Username             string `envconfig:"SERVER_USERNAME" default:"admin"`
	Password             string `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS                  bool   `envconfig:"HTTPS" default:"true"`
	Compression          int    `envconfig:"COMPRESSION" default:"0" description:"Compression level, 0 is no compression, and 9 maximum compression"`
	RLECompression       bool   `envconfig:"RLE_COMPRESSION" default:"true"`
	DevMode              bool   `envconfig:"DEV_MODE" default:"false"`
	ZSTDCompression      bool   `envconfig:"ZSTD_COMPRESSION" default:"false" description:"Enable zstd compression"`
	ZSTDCompressionLevel int    `envconfig:"ZSTD_COMPRESSION_LEVEL" default:"3" description:"Zstd compression level (1-22, where 1 is fastest and 22 is maximum compression)"`
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
	if remarkable.Model == remarkable.RemarkablePaperPro {
		if c.RLECompression {
			return errors.New("RLE compression is not supported on the Remarkable Paper Pro. Disable it by setting RLE_COMPRESSION=false")
		}
	}

	return nil
}

func main() {
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
