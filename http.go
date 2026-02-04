package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/owulveryck/goMarkableStream/internal/eventhttphandler"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
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

	// Custom handler to serve index.html for root path
	mux.HandleFunc("/", newIndexHandler(stripFS{http.FS(assetsFS)}))

	streamHandler := stream.NewStreamHandler(file, pointerAddr, eventPublisher, c.RLECompression, c.DeltaCompression, c.DeltaThreshold)
	if c.Compression {
		mux.Handle("/stream", gzMiddleware(stream.ThrottlingMiddleware(streamHandler)))
	} else if c.ZSTDCompression {
		mux.Handle("/stream", zstdMiddleware(stream.ThrottlingMiddleware(streamHandler), c.ZSTDCompressionLevel))
	} else {
		mux.Handle("/stream", stream.ThrottlingMiddleware(streamHandler))
	}

	wsHandler := eventhttphandler.NewEventHandler(eventPublisher)
	mux.Handle("/events", wsHandler)
	gestureHandler := eventhttphandler.NewGestureHandler(eventPublisher)
	mux.Handle("/gestures", gestureHandler)

	// Version endpoint
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			http.Error(w, "Unable to read build info", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s", bi.Main.Version)
	})

	if c.DevMode {
		rawHandler := stream.NewRawHandler(file, pointerAddr)
		mux.Handle("/raw", rawHandler)
	}
	return mux
}

func parseIndexTemplate(templatePath string) (*template.Template, error) {
	indexData, err := assetsFS.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("index.html").Parse(string(indexData))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func newIndexHandler(fs http.FileSystem) http.HandlerFunc {
	tmpl, err := parseIndexTemplate("client/index.html")
	if err != nil {
		log.Fatalf("Error parsing index template: %v", err)
		panic(err)
	}

	staticFileServer := http.FileServer(fs)

	data := struct {
		ScreenWidth    int
		ScreenHeight   int
		MaxXValue      int
		MaxYValue      int
		UseRLE         bool
		UseDelta       bool
		DeviceModel    string
		UseBGRA        bool
		TextureFlipped bool
	}{
		ScreenWidth:    remarkable.Config.Width,
		ScreenHeight:   remarkable.Config.Height,
		MaxXValue:      remarkable.MaxXValue,
		MaxYValue:      remarkable.MaxYValue,
		UseRLE:         c.RLECompression,
		UseDelta:       c.DeltaCompression,
		DeviceModel:    remarkable.Model.String(),
		UseBGRA:        remarkable.Config.UseBGRA,
		TextureFlipped: remarkable.Config.TextureFlipped,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.Execute(w, data); err != nil {
				http.Error(w, "Error rendering template", http.StatusInternalServerError)
				log.Printf("Error rendering template: %v", err)
			}
			return
		}

		staticFileServer.ServeHTTP(w, r)
	}
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
