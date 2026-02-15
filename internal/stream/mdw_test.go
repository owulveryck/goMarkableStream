package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestThrottleMiddlewareGoroutineCleanup verifies that the throttle middleware
// doesn't leak goroutines under high connection churn.
// This tests Bug #11: goroutine cleanup in throttling middleware.
func TestThrottleMiddlewareGoroutineCleanup(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	var mu sync.Mutex
	currentStreamCancel := make(map[*http.Request]context.CancelFunc)

	// Create middleware
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate stream context management
			streamCtx, streamCancel := context.WithCancel(context.Background())
			mu.Lock()
			currentStreamCancel[r] = streamCancel
			currentStreamCtx := streamCtx
			mu.Unlock()

			defer func() {
				mu.Lock()
				if cancel, ok := currentStreamCancel[r]; ok {
					cancel()
					delete(currentStreamCancel, r)
				}
				mu.Unlock()
			}()

			// Create merged context (same pattern as mdw.go)
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

			// Serve request
			next.ServeHTTP(w, r.WithContext(mergedCtx))
		})
	}

	// Simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Make many concurrent requests to simulate high connection churn
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			wrappedHandler.ServeHTTP(w, r)
		}()
	}

	wg.Wait()

	// Allow goroutines time to exit
	time.Sleep(200 * time.Millisecond)
	runtime.GC()

	// Check goroutine count
	finalGoroutines := runtime.NumGoroutine()

	// Should be close to baseline (allowing some tolerance for test goroutines)
	leak := finalGoroutines - baselineGoroutines
	if leak > 10 {
		t.Errorf("Potential goroutine leak: baseline=%d, final=%d (leak=%d)",
			baselineGoroutines, finalGoroutines, leak)
	} else {
		t.Logf("Goroutine count OK: baseline=%d, final=%d (diff=%d)",
			baselineGoroutines, finalGoroutines, leak)
	}
}
