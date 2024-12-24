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
	handler := NewStreamHandler(file, pointerAddr, eventPublisher, true)

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
				io.ReadAll(resp.Body)
			}

			doneChan <- struct{}{}
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < concurrentRequests; i++ {
		<-doneChan
	}
}

func TestStreamHandler_fetchAndSend(t *testing.T) {
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
			h.fetchAndSend(w, tt.args.rawData)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("StreamHandler.fetchAndSend() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
