package main

import (
	"compress/gzip"
	"embed"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

type configuration struct {
	BindAddr string `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Username string `envconfig:"SERVER_USERNAME" default:"admin"`
	Password string `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS      bool   `envconfig:"HTTPS" default:"true"`
	DevMode  bool   `envconfig:"DEV_MODE" default:"false"`
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

	//go:embed assets/favicon.ico
	favicon []byte
	//go:embed assets/index.html
	index []byte
	//go:embed assets/stream.js
	js []byte
	//go:embed assets/cert.pem assets/key.pem
	tlsAssets    embed.FS
	waitingQueue = make(chan struct{}, 2)
)

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

	file, pointerAddr, err = remarkable.GetFileAndPointer()
	pointerAddr = pointerAddr + 8
	if err != nil {
		log.Fatal(err)
	}
	mux := setMux()

	handler := BasicAuthMiddleware(gzMiddleware(mux))
	if *unsafe {
		handler = mux
	}
	if c.TLS {
		log.Fatal(runTLS(handler))
	}
	log.Fatal(http.ListenAndServe(c.BindAddr, handler))
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzMiddleware(fn http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz, _ := gzip.NewWriterLevel(w, 1)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn.ServeHTTP(gzr, r)
	}
}
