package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"nhooyr.io/websocket"
)

type configuration struct {
	BindAddr    string `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Dev         bool   `envconfig:"SERVER_DEV" default:"false" description:"Development mode: serves a local picture"`
	Username    string `envconfig:"SERVER_USERNAME" default:"admin"`
	Password    string `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS         bool   `envconfig:"HTTPS" default:"true"`
	Compression bool   `envconfig:"COMPRESSION" default:"false"`
}

const (
	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404
	// ConfigPrefix for environment variable based configuration
	ConfigPrefix = "RK"
)

var (
	pointerAddr int64
	file        io.ReaderAt
	// Define the username and password for authentication
	c configuration

	//go:embed favicon.ico
	favicon []byte
	//go:embed index.html
	index []byte
	//go:embed cert.pem key.pem
	tlsAssets    embed.FS
	waitingQueue = make(chan struct{}, 2)
)

func main() {

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

	var err error
	if c.Dev {
		file, err = os.OpenFile("testdata/empty.raw", os.O_RDONLY, os.ModeDevice)
		if err != nil {
			log.Fatal("cannot open file: ", err)
		}
		pointerAddr = 0
	} else {
		pid := findPid()
		file, err = os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
		if err != nil {
			log.Fatal("cannot open file: ", err)
		}
		pointerAddr, err = getPointer(pid)
		if err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		io.Copy(w, bytes.NewReader(favicon))
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
	mux.HandleFunc("/ws", handleWebSocket)
	handler := BasicAuthMiddleware(mux)
	if *unsafe {
		handler = mux
	}
	if c.TLS {
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
		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}
	log.Fatal(http.ListenAndServe(c.BindAddr, handler))

}
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	select {
	case waitingQueue <- struct{}{}:
		defer func() {
			<-waitingQueue
		}()
		// Generate a random integer between 0 and 100
		tick := time.Tick(200 * time.Millisecond) // Create a tick channel that emits a value every 200 milliseconds
		timeout := time.Tick(1 * time.Hour)

		// Create a context with a cancellation function
		options := &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		}
		if c.Compression {
			options = &websocket.AcceptOptions{
				CompressionMode: websocket.CompressionContextTakeover,
			}
		}

		conn, err := websocket.Accept(w, r, options)
		//conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "Internal Server Error")

		// Simulated pixel data

		imageData := make([]byte, ScreenWidth*ScreenHeight)
		// the informations are int4, therefore store it in a uint8array to reduce data transfer
		uint8Array := make([]uint8, len(imageData)/2)

		for {
			select {
			case <-timeout:
				conn.Close(websocket.StatusNormalClosure, "timeout")
				return
			case <-r.Context().Done():
				return
			case <-tick:
				ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
				defer cancel()
				_, err := file.ReadAt(imageData, pointerAddr)
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < len(imageData); i += 2 {
					packedValue := (uint8(imageData[i]) << 4) | uint8(imageData[i+1])
					uint8Array[i/2] = packedValue
				}

				err = conn.Write(ctx, websocket.MessageBinary, uint8Array)
				if err != nil {
					//					log.Println(err)
					return
				}
			}
		}
	default:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
}
