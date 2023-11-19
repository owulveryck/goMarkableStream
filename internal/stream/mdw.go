package stream

import (
	"net/http"
	"sync"
)

var (
	activeWriters int
	maxWriters    = 1 // Maximum allowed writers
	mu            sync.Mutex
	cond          = sync.NewCond(&mu)
)

// TrhorrlingMiddleware to allow new connections only if there are no active writers or if max writers is exceeded.
func ThrottlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if activeWriters >= maxWriters {
			mu.Unlock()
			// If too many requests, send a 429 status code
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		for activeWriters > 0 {
			cond.Wait() // Wait for active writers to finish
		}
		activeWriters++
		mu.Unlock()

		next.ServeHTTP(w, r) // Serve the request

		mu.Lock()
		activeWriters--
		cond.Broadcast() // Notify waiting goroutines
		mu.Unlock()
	})
}
