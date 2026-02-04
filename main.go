package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"

	dbg "github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

type configuration struct {
	BindAddr       string  `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Username       string  `envconfig:"SERVER_USERNAME" default:"admin"`
	Password       string  `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS            bool    `envconfig:"HTTPS" default:"true"`
	DevMode        bool    `envconfig:"DEV_MODE" default:"false"`
	DeltaThreshold float64 `envconfig:"DELTA_THRESHOLD" default:"0.30" description:"Change ratio threshold (0.0-1.0) above which full frame is sent"`
	Debug          bool    `envconfig:"DEBUG" default:"false" description:"Enable debug logging"`

	// Tailscale configuration
	TailscaleHostname string `envconfig:"TAILSCALE_HOSTNAME" default:"gomarkablestream" description:"Device name in tailnet"`
	TailscaleStateDir string `envconfig:"TAILSCALE_STATE_DIR" default:"/home/root/.tailscale/gomarkablestream" description:"State directory for Tailscale"`
	TailscaleAuthKey  string `envconfig:"TAILSCALE_AUTHKEY" default:"" description:"Auth key for headless setup"`
	TailscaleEphemeral bool  `envconfig:"TAILSCALE_EPHEMERAL" default:"false" description:"Register as ephemeral node"`
	TailscaleFunnel   bool   `envconfig:"TAILSCALE_FUNNEL" default:"false" description:"Enable public internet access via Funnel"`
	TailscaleUseTLS   bool   `envconfig:"TAILSCALE_USE_TLS" default:"false" description:"Use Tailscale's TLS certs"`
	TailscaleVerbose  bool   `envconfig:"TAILSCALE_VERBOSE" default:"false" description:"Verbose Tailscale logging"`
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

	dbg.Enabled = c.Debug

	file, pointerAddr, err = remarkable.GetFileAndPointer()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup listener first to get TailscaleManager
	listenerResult, err := setupListener(ctx, &c)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if listenerResult.Cleanup != nil {
			if err := listenerResult.Cleanup(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		}
	}()

	for _, l := range listenerResult.Listeners {
		log.Printf("listening on %v", l.Addr())
	}

	eventPublisher := pubsub.NewPubSub()
	eventScanner := remarkable.NewEventScanner()
	eventScanner.StartAndPublish(ctx, eventPublisher)

	// Channel to signal Tailscale listener restart
	restartCh := make(chan bool, 1)

	// Pass TailscaleManager and restart channel to setMuxer
	mux := setMuxer(eventPublisher, listenerResult.TailscaleManager, restartCh)

	var handler http.Handler
	handler = BasicAuthMiddleware(mux)
	if *unsafe {
		handler = mux
	}

	// Create HTTP server for graceful shutdown
	server := &http.Server{
		Handler: handler,
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server goroutines for each listener
	serverErr := make(chan error, len(listenerResult.Listeners)+1)
	for i, listener := range listenerResult.Listeners {
		isTailscale := i == 0 && listenerResult.TailscaleManager != nil
		go func(l net.Listener, isTailscale bool) {
			for {
				log.Printf("Serving on %v", l.Addr())
				var serveErr error
				if listenerResult.UseTLS {
					serveErr = runTLS(l, handler)
				} else {
					serveErr = server.Serve(l)
				}

				if serveErr == http.ErrServerClosed {
					return
				}

				// If Tailscale listener, wait for restart signal
				if isTailscale && listenerResult.TailscaleManager != nil {
					select {
					case <-restartCh:
						l = listenerResult.TailscaleManager.GetListener()
						continue
					case <-ctx.Done():
						return
					}
				}

				serverErr <- serveErr
				return
			}
		}(listener, isTailscale)
	}

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()

		// Give the server time to finish ongoing requests
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("Server shutdown complete")

	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
