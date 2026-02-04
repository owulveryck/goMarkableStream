package stream

import (
	"context"
	"net/http"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/debug"
)

var (
	activeWriters int
	maxWriters    = 1 // Maximum allowed writers
	mu            sync.Mutex
	cond          = sync.NewCond(&mu)

	// Stream cancellation context
	streamCtx    context.Context
	streamCancel context.CancelFunc
)

func init() {
	streamCtx, streamCancel = context.WithCancel(context.Background())
}

// CancelActiveStreams cancels all active stream connections
// and creates a fresh context for new connections
func CancelActiveStreams() {
	mu.Lock()
	defer mu.Unlock()

	// Cancel existing streams
	if streamCancel != nil {
		streamCancel()
	}

	// Create new context for future streams
	streamCtx, streamCancel = context.WithCancel(context.Background())
}

// ThrottlingMiddleware to allow new connections only if there are no active writers or if max writers is exceeded.
func ThrottlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Log("Throttle: connection from %s, activeWriters=%d", r.RemoteAddr, activeWriters)
		mu.Lock()
		if activeWriters >= maxWriters {
			mu.Unlock()
			debug.Log("Throttle: too many requests, rejecting (%s)", r.RemoteAddr)
			// If too many requests, send a 429 status code
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		for activeWriters > 0 {
			debug.Log("Throttle: waiting for active writers to finish (%s)", r.RemoteAddr)
			cond.Wait() // Wait for active writers to finish
		}

		// Capture current stream context while holding lock
		currentStreamCtx := streamCtx
		activeWriters++
		debug.Log("Throttle: proceeding with request (%s)", r.RemoteAddr)
		mu.Unlock()

		// Create merged context: cancels if client disconnects OR server cancels
		mergedCtx, mergedCancel := context.WithCancel(r.Context())
		defer mergedCancel()

		// Monitor stream context cancellation in background
		go func() {
			select {
			case <-currentStreamCtx.Done():
				mergedCancel()
			case <-mergedCtx.Done():
				// Already cancelled
			}
		}()

		// Create new request with merged context
		r = r.WithContext(mergedCtx)

		next.ServeHTTP(w, r) // Serve the request

		mu.Lock()
		activeWriters--
		debug.Log("Throttle: request completed, activeWriters=%d (%s)", activeWriters, r.RemoteAddr)
		cond.Broadcast() // Notify waiting goroutines
		mu.Unlock()
	})
}
