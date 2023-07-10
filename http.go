package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"

	"github.com/owulveryck/goMarkableStream/internal/stream"
)

func setMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		io.Copy(w, bytes.NewReader(favicon))
	})
	mux.HandleFunc("/stream.js", func(w http.ResponseWriter, _ *http.Request) {
		io.Copy(w, bytes.NewReader(js))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		select {
		case waitingQueue <- struct{}{}:
			defer func() {
				<-waitingQueue
			}()
			io.Copy(w, bytes.NewReader(index))
		default:
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
	})
	streanHandler := stream.NewStreamHandler(file, pointerAddr)
	mux.Handle("/stream", streanHandler)
	return mux
}

func runTLS(handler http.Handler) error {
	// Load the certificate and key from embedded files
	cert, err := tlsAssets.ReadFile("cert.pem")
	if err != nil {
		log.Fatal("Error reading embedded certificate:", err)
	}

	key, err := tlsAssets.ReadFile("key.pem")
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

	// Create the server
	server := &http.Server{
		Addr:      c.BindAddr,
		TLSConfig: config,
		Handler:   handler,
	}

	// Start the server
	return server.ListenAndServeTLS("", "")
}
