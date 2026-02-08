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
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"

	dbg "github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/jwtutil"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/tlsutil"
)

type configuration struct {
	BindAddr       string  `envconfig:"SERVER_BIND_ADDR" default:":2001" required:"true" description:"The server bind address"`
	Username       string  `envconfig:"SERVER_USERNAME" default:"admin"`
	Password       string  `envconfig:"SERVER_PASSWORD" default:"password"`
	TLS            bool    `envconfig:"HTTPS" default:"true"`
	DevMode        bool    `envconfig:"DEV_MODE" default:"false"`
	DeltaThreshold float64 `envconfig:"DELTA_THRESHOLD" default:"0.30" description:"Change ratio threshold (0.0-1.0) above which full frame is sent"`
	Debug          bool    `envconfig:"DEBUG" default:"false" description:"Enable debug logging"`

	// TLS certificate configuration
	TLSCertFile     string `envconfig:"TLS_CERT_FILE" default:"" description:"Path to custom TLS certificate file"`
	TLSKeyFile      string `envconfig:"TLS_KEY_FILE" default:"" description:"Path to custom TLS key file"`
	TLSCertDir      string `envconfig:"TLS_CERT_DIR" default:"/home/root/.config/goMarkableStream/certs" description:"Directory for auto-generated certificates"`
	TLSAutoGenerate bool   `envconfig:"TLS_AUTO_GENERATE" default:"true" description:"Auto-generate device-specific certificates"`
	TLSHostnames    string `envconfig:"TLS_HOSTNAMES" default:"" description:"Additional hostnames for certificate SANs (comma-separated)"`
	TLSValidDays    int    `envconfig:"TLS_VALID_DAYS" default:"365" description:"Validity period for generated certificates in days"`

	// Tailscale configuration
	TailscaleEnabled   bool   `envconfig:"TAILSCALE_ENABLED" default:"false" description:"Enable Tailscale listener"`
	TailscalePort      string `envconfig:"TAILSCALE_PORT" default:":8443" description:"Tailscale listener port"`
	TailscaleHostname  string `envconfig:"TAILSCALE_HOSTNAME" default:"gomarkablestream" description:"Device name in tailnet"`
	TailscaleStateDir  string `envconfig:"TAILSCALE_STATE_DIR" default:"/home/root/.tailscale/gomarkablestream" description:"State directory for Tailscale"`
	TailscaleAuthKey   string `envconfig:"TAILSCALE_AUTHKEY" default:"" description:"Auth key for headless setup"`
	TailscaleEphemeral bool   `envconfig:"TAILSCALE_EPHEMERAL" default:"false" description:"Register as ephemeral node"`
	TailscaleFunnel    bool   `envconfig:"TAILSCALE_FUNNEL" default:"false" description:"Enable public internet access via Funnel"`
	TailscaleUseTLS    bool   `envconfig:"TAILSCALE_USE_TLS" default:"false" description:"Use Tailscale's TLS certs"`
	TailscaleVerbose   bool   `envconfig:"TAILSCALE_VERBOSE" default:"false" description:"Verbose Tailscale logging"`

	// JWT configuration
	JWTEnabled       bool   `envconfig:"JWT_ENABLED" default:"true" description:"Enable JWT authentication"`
	JWTSecretDir     string `envconfig:"JWT_SECRET_DIR" default:"/home/root/.config/goMarkableStream/secrets" description:"Directory for JWT secret key"`
	JWTTokenLifetime string `envconfig:"JWT_TOKEN_LIFETIME" default:"24h" description:"JWT token validity duration"`
	JWTAutoGenerate  bool   `envconfig:"JWT_AUTO_GENERATE" default:"true" description:"Auto-generate JWT secret key"`
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
	// JWT manager for token authentication
	jwtMgr *jwtutil.Manager

	//go:embed client/*
	assetsFS embed.FS
	//go:embed assets/cert.pem assets/key.pem
	tlsAssets embed.FS
)

func validateConfiguration(c *configuration) error {
	return nil
}

// tlsErrorFilter filters out TLS handshake errors from logs.
// These errors are expected when using self-signed certificates
// and browsers initially reject the certificate.
type tlsErrorFilter struct{}

func (f *tlsErrorFilter) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), "TLS handshake error") {
		return len(p), nil // Silently discard
	}
	return os.Stderr.Write(p)
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
		_ = envconfig.Usage(ConfigPrefix, &c)
		return
	}
	if err := envconfig.Process(ConfigPrefix, &c); err != nil {
		_ = envconfig.Usage(ConfigPrefix, &c)
		log.Fatal(err)
	}

	if err := validateConfiguration(&c); err != nil {
		panic(err)
	}

	dbg.Enabled = c.Debug

	// Initialize JWT manager if enabled
	if c.JWTEnabled {
		tokenLifetime, err := time.ParseDuration(c.JWTTokenLifetime)
		if err != nil {
			log.Printf("Invalid JWT token lifetime %q, using default 24h", c.JWTTokenLifetime)
			tokenLifetime = 24 * time.Hour
		}
		jwtMgr = jwtutil.NewManager(jwtutil.ManagerConfig{
			SecretDir:     c.JWTSecretDir,
			TokenLifetime: tokenLifetime,
			AutoGenerate:  c.JWTAutoGenerate,
		})
		if err := jwtMgr.Initialize(); err != nil {
			log.Printf("Warning: JWT initialization failed: %v (falling back to Basic Auth only)", err)
			jwtMgr = nil
		}
	}

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
	mux := setMuxer(eventPublisher, listenerResult.TailscaleManager, restartCh, jwtMgr)

	var handler http.Handler
	handler = AuthMiddleware(mux, jwtMgr)
	if *unsafe {
		handler = mux
	}

	// Create HTTP server for graceful shutdown
	server := &http.Server{
		Handler:  handler,
		ErrorLog: log.New(&tlsErrorFilter{}, "", 0),
	}

	// Create TLS manager if TLS is enabled
	var tlsMgr *tlsutil.Manager
	if listenerResult.UseTLS {
		tlsMgr = createTLSManager(&c)
		// Pre-load certificate to log information at startup
		_, _, err := tlsMgr.GetCertificate()
		if err != nil {
			log.Printf("Warning: TLS certificate issue: %v", err)
		}
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server goroutines for each listener (local listeners)
	serverErr := make(chan error, len(listenerResult.Listeners)+1)
	for _, listener := range listenerResult.Listeners {
		go func(l net.Listener) {
			log.Printf("Serving on %v", l.Addr())
			var serveErr error
			if listenerResult.UseTLS {
				serveErr = runTLS(l, server, tlsMgr)
			} else {
				serveErr = server.Serve(l)
			}

			if serveErr != http.ErrServerClosed {
				serverErr <- serveErr
			}
		}(listener)
	}

	// If Tailscale is enabled, wait for it to be ready in background and start serving
	if listenerResult.TailscaleManager != nil {
		go func() {
			tm := listenerResult.TailscaleManager
			select {
			case <-tm.Ready():
				if !tm.IsReady() {
					log.Println("Tailscale failed to start, continuing with local listener only")
					return
				}
				// Tailscale is ready, start serving on its listener
				l := tm.GetListener()
				if l == nil {
					return
				}
				for {
					log.Printf("Serving on Tailscale: %v", l.Addr())
					serveErr := server.Serve(l)
					if serveErr == http.ErrServerClosed {
						return
					}
					// Handle restart for funnel toggle
					select {
					case <-restartCh:
						l = tm.GetListener()
						if l == nil {
							return
						}
						continue
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}()
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
