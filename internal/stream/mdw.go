package stream

import (
	"net/http"
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/debug"
)

var (
	activeWriters int
	maxWriters    = 1 // Maximum allowed writers
	mu            sync.Mutex
	cond          = sync.NewCond(&mu)
)

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
		activeWriters++
		debug.Log("Throttle: proceeding with request (%s)", r.RemoteAddr)
		mu.Unlock()

		next.ServeHTTP(w, r) // Serve the request

		mu.Lock()
		activeWriters--
		debug.Log("Throttle: request completed, activeWriters=%d (%s)", activeWriters, r.RemoteAddr)
		cond.Broadcast() // Notify waiting goroutines
		mu.Unlock()
	})
}
