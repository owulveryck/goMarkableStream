package stream

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/rle"
)

const (
	rate = 200
)

var rawFrameBuffer = sync.Pool{
	New: func() any {
		return make([]uint8, remarkable.ScreenWidth*remarkable.ScreenHeight*2) // Adjust the initial capacity as needed
	},
}

// NewStreamHandler creates a new stream handler reading from file @pointerAddr
func NewStreamHandler(file io.ReaderAt, pointerAddr int64, inputEvents *pubsub.PubSub) *StreamHandler {
	return &StreamHandler{
		file:           file,
		pointerAddr:    pointerAddr,
		inputEventsBus: inputEvents,
	}
}

// StreamHandler is an http.Handler that serves the stream of data to the client
type StreamHandler struct {
	file           io.ReaderAt
	pointerAddr    int64
	inputEventsBus *pubsub.PubSub
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Send response to preflight request
		w.WriteHeader(http.StatusOK)
		return
	}

	eventC := h.inputEventsBus.Subscribe("stream")
	defer h.inputEventsBus.Unsubscribe(eventC)
	ticker := time.NewTicker(rate * time.Millisecond)
	ticker.Reset(rate * time.Millisecond)
	defer ticker.Stop()

	rawData := rawFrameBuffer.Get().([]uint8)
	defer rawFrameBuffer.Put(rawData) // Return the slice to the pool when done
	// the informations are int4, therefore store it in a uint8array to reduce data transfer
	rleWriter := rle.NewRLE(w)
	extractor := &oneOutOfTwo{rleWriter}
	writing := true
	stopWriting := time.NewTicker(2 * time.Second)
	defer stopWriting.Stop()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Transfer-Encoding", "chunked")

	for {
		select {
		case <-r.Context().Done():
			return
		case <-eventC:
			writing = true
			stopWriting.Reset(2 * time.Second)
		case <-stopWriting.C:
			writing = false
		case <-ticker.C:
			if writing {
				_, err := h.file.ReadAt(rawData, h.pointerAddr)
				if err != nil {
					log.Println(err)
					return
				}
				_, err = extractor.Write(rawData)
				if err != nil {
					log.Println("Error in writing", err)
					return
				}
				if w, ok := w.(http.Flusher); ok {
					w.Flush()
				}
			}
		}
	}
}
