package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/owulveryck/goMarkableStream/internal/eventhttphandler"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/stream"
)

type stripFS struct {
	fs http.FileSystem
}

func (s stripFS) Open(name string) (http.File, error) {
	return s.fs.Open("client" + name)
}

func setMuxer(eventPublisher *pubsub.PubSub) *http.ServeMux {
	mux := http.NewServeMux()

	fs := http.FileServer(stripFS{http.FS(assetsFS)})

	// Custom handler to serve index.html for root path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			index, err := assetsFS.ReadFile("client/index.html")
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(w, bytes.NewReader(index))
			return
		}
		fs.ServeHTTP(w, r)
	})

	streamHandler := stream.NewStreamHandler(file, pointerAddr, eventPublisher)
	mux.Handle("/stream", streamHandler)
	wsHandler := eventhttphandler.NewEventHandler(eventPublisher)
	mux.Handle("/events", wsHandler)

	if c.DevMode {
		rawHandler := stream.NewRawHandler(file, pointerAddr)
		mux.Handle("/raw", rawHandler)
	}
	return mux
}

func runTLS(l net.Listener, handler http.Handler) error {
	// Load the certificate and key from embedded files
	cert, err := tlsAssets.ReadFile("assets/cert.pem")
	if err != nil {
		log.Fatal("Error reading embedded certificate:", err)
	}

	key, err := tlsAssets.ReadFile("assets/key.pem")
	if err != nil {
		log.Fatal("Error reading embedded key:", err)
	}

	certPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.Fatal("Error creating X509 key pair:", err)
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{certPair},
		InsecureSkipVerify: true,
	}

	tlsListener := tls.NewListener(l, config)

	// Start the server
	return http.Serve(tlsListener, handler)
}
