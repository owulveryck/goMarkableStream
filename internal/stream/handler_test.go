package stream

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// Assuming StreamHandler is defined somewhere in your package.
//
//	type StreamHandler struct {
//	    ...
//	}
func getFileAndPointer() (io.ReaderAt, int64, error) {
	return &dummyPicture{}, 0, nil

}

type dummyPicture struct{}

func (dummypicture *dummyPicture) ReadAt(p []byte, off int64) (n int, err error) {
	f, err := os.Open("../../testdata/full_memory_region.raw")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.ReadAt(p, off)
}

func TestStreamHandlerRaceCondition(t *testing.T) {
	// Initialize your StreamHandler here
	file, pointerAddr, err := getFileAndPointer()
	if err != nil {
		t.Fatal(err)
	}
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(file, pointerAddr, eventPublisher, 0.30)

	server := httptest.NewServer(handler)
	defer server.Close()

	// Simulate concurrent requests
	concurrentRequests := 100
	doneChan := make(chan struct{}, concurrentRequests)
	// Create a HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Millisecond,
	}

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			// Introduce a random delay up to 1 second before starting the request
			delay := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(delay)
			// Perform an HTTP request to the test server
			resp, err := client.Get(server.URL)
			if err == nil {
				defer resp.Body.Close()
				// Optionally read the response body
				_, _ = io.ReadAll(resp.Body)
			}

			doneChan <- struct{}{}
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < concurrentRequests; i++ {
		<-doneChan
	}
}

// TestConcurrentRateModification tests for race conditions when multiple
// clients set different rate values simultaneously.
// Run with: go test -race -run TestConcurrentRateModification
func TestConcurrentRateModification(t *testing.T) {
	file, pointerAddr, err := getFileAndPointer()
	if err != nil {
		t.Fatal(err)
	}
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(file, pointerAddr, eventPublisher, 0.30)

	server := httptest.NewServer(handler)
	defer server.Close()

	// Simulate concurrent requests with different rate parameters
	concurrentRequests := 50
	doneChan := make(chan struct{}, concurrentRequests)

	client := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	// Each goroutine sets a different rate value
	for i := 0; i < concurrentRequests; i++ {
		go func(rateValue int) {
			// Each client uses a different rate parameter
			url := server.URL + "?rate=" + string(rune(100+rateValue*10))
			resp, err := client.Get(url)
			if err == nil {
				defer resp.Body.Close()
				_, _ = io.ReadAll(resp.Body)
			}
			doneChan <- struct{}{}
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < concurrentRequests; i++ {
		<-doneChan
	}
}

// streamingWriter discards frames and signals when the first one lands, which
// is the point at which the handler is known to own the encoder.
type streamingWriter struct {
	hdr    http.Header
	once   sync.Once
	closed chan struct{}
}

func newStreamingWriter() *streamingWriter {
	return &streamingWriter{hdr: http.Header{}, closed: make(chan struct{})}
}

func (s *streamingWriter) Header() http.Header { return s.hdr }
func (s *streamingWriter) WriteHeader(int)     {}
func (s *streamingWriter) Flush()              {}
func (s *streamingWriter) Write(p []byte) (int, error) {
	s.once.Do(func() { close(s.closed) })
	return len(p), nil
}

// TestStreamHandler_SecondConnectionRejected covers the concurrency contract of
// StreamHandler itself. Its delta encoder is per-connection state, and until now
// two concurrent ServeHTTP calls would trample it — ThrottlingMiddleware kept
// that from biting in production, but the handler must hold on its own, since
// net/http is free to call any handler concurrently.
func TestStreamHandler_SecondConnectionRejected(t *testing.T) {
	h := NewStreamHandler(&countingReaderAt{}, 0, pubsub.NewPubSub(), 0.30)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := newStreamingWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/stream?rate=50", nil).WithContext(ctx))
	}()

	select {
	case <-first.closed:
	case <-time.After(5 * time.Second):
		t.Fatal("first connection never streamed a frame")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second connection: status %d, want %d", rec.Code, http.StatusTooManyRequests)
	}

	// Releasing memory under an active stream must be a no-op, not a block and
	// not a concurrent reset of buffers the encoder is using.
	released := make(chan struct{})
	go func() { h.ReleaseMemory(); close(released) }()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Error("ReleaseMemory blocked while a stream was active")
	}

	cancel()
	<-done

	// Once the stream is gone the handler accepts a new connection again.
	if !h.session.TryLock() {
		t.Error("encoder still held after the connection closed")
	} else {
		h.session.Unlock()
	}
}

// TestStreamHandler_PreflightDuringStream checks that the CORS preflight is
// answered even while a stream holds the encoder: it is taken before the lock
// precisely so a browser check is never turned away with a 429.
func TestStreamHandler_PreflightDuringStream(t *testing.T) {
	h := NewStreamHandler(&countingReaderAt{}, 0, pubsub.NewPubSub(), 0.30)
	h.session.Lock()
	defer h.session.Unlock()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/stream", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("preflight status %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

// TestPoolGetHandling tests that the handler safely handles pool.Get() results.
// This tests Bug #8 fix: sync.Pool type assertion without nil check.
func TestPoolGetHandling(t *testing.T) {
	// This test verifies that the code handles pool.Get() safely
	// The pool has a New function so it shouldn't return nil,
	// but we test the safety mechanism anyway

	// Reset the pool to ensure clean state
	ResetFrameBufferPool()

	file, pointerAddr, err := getFileAndPointer()
	if err != nil {
		t.Fatal(err)
	}
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(file, pointerAddr, eventPublisher, 0.30)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	// Make request - should not panic
	resp, err := client.Get(server.URL + "?rate=200")
	if err == nil {
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
	}
}

// mockFailingReaderAt is a ReaderAt that always returns an error
type mockFailingReaderAt struct{}

func (m *mockFailingReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	// Fill with non-zero data to simulate stale buffer
	for i := range p {
		p[i] = 0xFF
	}
	return len(p), io.ErrUnexpectedEOF
}

// TestFrameReadErrorHandling tests that read errors are handled properly.
// This tests Bug #14 fix: unchecked frame read error.
func TestFrameReadErrorHandling(t *testing.T) {
	failingReader := &mockFailingReaderAt{}
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(failingReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	rawData := make([]uint8, 100)

	// Fill buffer with non-zero data to simulate stale buffer
	for i := range rawData {
		rawData[i] = 0xFF
	}

	// Call fetchAndSendDelta - should handle error gracefully
	handler.fetchAndSendDelta(&buf, rawData)

	// Buffer should be cleared (all zeros for the read portion)
	for i, b := range rawData {
		if b != 0 {
			t.Errorf("Buffer not cleared at index %d: got %d, want 0", i, b)
			break
		}
	}
}

func TestStreamHandler_fetchAndSendDelta(t *testing.T) {
	type fields struct {
		file           io.ReaderAt
		pointerAddr    int64
		inputEventsBus *pubsub.PubSub
	}
	type args struct {
		rawData []uint8
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantW  string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &StreamHandler{
				file:           tt.fields.file,
				pointerAddr:    tt.fields.pointerAddr,
				inputEventsBus: tt.fields.inputEventsBus,
			}
			w := &bytes.Buffer{}
			h.fetchAndSendDelta(w, tt.args.rawData)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("StreamHandler.fetchAndSendDelta() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

// TestThrottlingMiddleware_ReleasesSlotOnPanic checks that a panicking handler
// does not leave the single streaming slot taken forever.
func TestThrottlingMiddleware_ReleasesSlotOnPanic(t *testing.T) {
	panicking := ThrottlingMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	func() {
		defer func() { _ = recover() }()
		panicking.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
	}()

	rec := httptest.NewRecorder()
	ThrottlingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("after a panic: status %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestStreamHandler_RejectsOutOfRangeRate covers the overflow that used to make
// time.NewTicker panic, and rates fast enough to saturate the device.
func TestStreamHandler_RejectsOutOfRangeRate(t *testing.T) {
	h := NewStreamHandler(&countingReaderAt{}, 0, pubsub.NewPubSub(), 0.30)
	for _, rate := range []string{"0", "1", "49", "1001", "9300000000000", "-5"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream?rate="+rate, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("rate=%s: status %d, want %d", rate, rec.Code, http.StatusBadRequest)
		}
	}
}
