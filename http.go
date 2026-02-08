package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"runtime"
	godebug "runtime/debug"

	"github.com/owulveryck/goMarkableStream/internal/delta"
	internalDebug "github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/eventhttphandler"
	"github.com/owulveryck/goMarkableStream/internal/jwtutil"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/stream"
	"github.com/owulveryck/goMarkableStream/internal/tlsutil"
)

type stripFS struct {
	fs http.FileSystem
}

func (s stripFS) Open(name string) (http.File, error) {
	return s.fs.Open("client" + name)
}

func setMuxer(eventPublisher *pubsub.PubSub, tm *TailscaleManager, restartCh chan<- bool, jwtMgr *jwtutil.Manager) *http.ServeMux {
	mux := http.NewServeMux()

	// Custom handler to serve index.html for root path
	mux.HandleFunc("/", newIndexHandler(stripFS{http.FS(assetsFS)}, jwtMgr != nil))

	// Login endpoint for JWT authentication
	mux.HandleFunc("/login", handleLogin(jwtMgr))

	streamHandler := stream.NewStreamHandler(file, pointerAddr, eventPublisher, c.DeltaThreshold)
	mux.Handle("/stream", stream.ThrottlingMiddleware(streamHandler))

	// Register idle callback to release memory when streaming ends
	stream.SetOnIdleCallback(func() {
		internalDebug.Log("Idle: releasing memory pools")
		stream.ResetFrameBufferPool()
		delta.ResetEncoderPool()
		streamHandler.ReleaseMemory()
		// Force garbage collection and return memory to OS
		runtime.GC()
		godebug.FreeOSMemory()
		internalDebug.Log("Idle: memory returned to OS")
	})

	wsHandler := eventhttphandler.NewEventHandler(eventPublisher)
	mux.Handle("/events", wsHandler)
	gestureHandler := eventhttphandler.NewGestureHandler(eventPublisher)
	mux.Handle("/gestures", gestureHandler)

	screenshotHandler := stream.NewScreenshotHandler(file, pointerAddr)
	mux.Handle("/screenshot", screenshotHandler)

	// Version endpoint
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		bi, ok := godebug.ReadBuildInfo()
		if !ok {
			http.Error(w, "Unable to read build info", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s", bi.Main.Version)
	})

	// Funnel status and toggle endpoint
	mux.HandleFunc("/funnel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if tm == nil {
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"available": false,
				"enabled":   false,
				"url":       "",
			}); err != nil {
				log.Printf("failed to encode JSON response: %v", err)
			}
			return
		}

		if r.Method == "POST" {
			var req struct {
				Enable bool `json:"enable"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("Funnel toggle: failed to decode request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			log.Printf("Funnel toggle: requested enable=%v", req.Enable)

			// Cancel active streams before toggling listener
			stream.CancelActiveStreams()

			_, err := tm.ToggleFunnel(req.Enable)
			if err != nil {
				log.Printf("Funnel toggle: failed: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Funnel toggle: success, enabled=%v", req.Enable)

			// Manage temporary credentials based on Funnel state
			if req.Enable {
				// Generate temporary credentials when enabling Funnel
				if funnelCreds != nil {
					username, password := funnelCreds.Generate()
					log.Printf("Funnel: temporary credentials generated (user: %s)", username)
					_ = password // password is returned in response
				}
			} else {
				// Clear temporary credentials when disabling Funnel
				if funnelCreds != nil {
					funnelCreds.Clear()
					log.Println("Funnel: temporary credentials cleared")
				}
			}

			// Signal main to restart Tailscale server goroutine
			if restartCh != nil {
				select {
				case restartCh <- req.Enable:
					log.Println("Funnel toggle: restart signal sent")
				default:
					log.Println("Funnel toggle: restart channel full, skipping signal")
				}
			}
		}

		enabled, url, err := tm.GetFunnelInfo()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"available": true,
			"enabled":   enabled,
			"url":       url,
		}

		// Include temporary credentials if Funnel is enabled and credentials are active
		if enabled && funnelCreds != nil {
			username, password, active := funnelCreds.GetCredentials()
			if active {
				response["tempCredentials"] = map[string]string{
					"username": username,
					"password": password,
				}
			}
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("failed to encode JSON response: %v", err)
		}
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

// handleLogin handles the /login endpoint for JWT authentication.
func handleLogin(jwtMgr *jwtutil.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if JWT is enabled
		if jwtMgr == nil {
			http.Error(w, "JWT authentication not enabled", http.StatusServiceUnavailable)
			return
		}

		// Parse request body
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		// Validate credentials
		if !checkCredentials(req.Username, req.Password) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		// Create JWT token
		token, err := jwtMgr.CreateToken(req.Username)
		if err != nil {
			log.Printf("Failed to create JWT token: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create token"})
			return
		}

		// Return token
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":     token,
			"expiresIn": int64(jwtMgr.GetTokenLifetime().Seconds()),
		})
	}
}

func newIndexHandler(fs http.FileSystem, jwtEnabled bool) http.HandlerFunc {
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
		DeviceModel    string
		UseBGRA        bool
		TextureFlipped bool
		JWTEnabled     bool
	}{
		ScreenWidth:    remarkable.Config.Width,
		ScreenHeight:   remarkable.Config.Height,
		MaxXValue:      remarkable.MaxXValue,
		MaxYValue:      remarkable.MaxYValue,
		DeviceModel:    remarkable.Model.String(),
		UseBGRA:        remarkable.Config.UseBGRA,
		TextureFlipped: remarkable.Config.TextureFlipped,
		JWTEnabled:     jwtEnabled,
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

func runTLS(l net.Listener, server *http.Server, tlsMgr *tlsutil.Manager) error {
	tlsConfig, _, err := tlsMgr.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to get TLS config: %w", err)
	}

	tlsListener := tls.NewListener(l, tlsConfig)

	// Start the server
	return server.Serve(tlsListener)
}

// createTLSManager creates a TLS manager from the configuration.
func createTLSManager(cfg *configuration) *tlsutil.Manager {
	// Load embedded certificates as fallback
	embeddedCert, certErr := tlsAssets.ReadFile("assets/cert.pem")
	embeddedKey, keyErr := tlsAssets.ReadFile("assets/key.pem")

	var cert, key []byte
	if certErr == nil && keyErr == nil {
		cert = embeddedCert
		key = embeddedKey
	}

	return tlsutil.NewManager(tlsutil.ManagerConfig{
		CertFile:            cfg.TLSCertFile,
		KeyFile:             cfg.TLSKeyFile,
		CertDir:             cfg.TLSCertDir,
		AutoGenerate:        cfg.TLSAutoGenerate,
		Hostnames:           cfg.TLSHostnames,
		ValidDays:           cfg.TLSValidDays,
		ExpiryThresholdDays: 30,
		EmbeddedCert:        cert,
		EmbeddedKey:         key,
	})
}
