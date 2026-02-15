package stream

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
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
