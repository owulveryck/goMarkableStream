package stream

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/delta"
	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

var (
	rate time.Duration = 200
)

var rawFrameBuffer = sync.Pool{
	New: func() any {
		buf := make([]uint8, remarkable.Config.SizeBytes)
		return &buf
	},
}

// ResetFrameBufferPool replaces the frame buffer pool with a fresh one,
// allowing old buffers to be garbage collected.
func ResetFrameBufferPool() {
	rawFrameBuffer = sync.Pool{
		New: func() any {
			buf := make([]uint8, remarkable.Config.SizeBytes)
			return &buf
		},
	}
}

// NewStreamHandler creates a new stream handler reading from file @pointerAddr
func NewStreamHandler(file io.ReaderAt, pointerAddr int64, inputEvents *pubsub.PubSub, deltaThreshold float64) *StreamHandler {
	return &StreamHandler{
		file:           file,
		pointerAddr:    pointerAddr,
		inputEventsBus: inputEvents,
		deltaEncoder:   delta.NewEncoder(deltaThreshold),
	}
}

// StreamHandler is an http.Handler that serves the stream of data to the client
type StreamHandler struct {
	file           io.ReaderAt
	pointerAddr    int64
	inputEventsBus *pubsub.PubSub
	deltaEncoder   *delta.Encoder
}

// ReleaseMemory releases large buffers held by the stream handler's delta encoder.
func (h *StreamHandler) ReleaseMemory() {
	h.deltaEncoder.ReleaseMemory()
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug.Log("Stream: new connection from %s", r.RemoteAddr)

	// Reset delta encoder to force a full frame for the new client
	h.deltaEncoder.Reset()

	// Parse query parameters
	query := r.URL.Query()
	rateStr := query.Get("rate")
	// If 'rate' parameter exists and is valid, use its value
	if rateStr != "" {
		var err error
		rateInt, err := strconv.Atoi(rateStr)
		if err != nil {
			// Handle error or keep the default value
			// For example, you can send a response with an error message
			http.Error(w, "Invalid 'rate' parameter", http.StatusBadRequest)
			return
		}
		rate = time.Duration(rateInt)
	}
	if rate < 100 {
		http.Error(w, "rate value is too low", http.StatusBadRequest)
		return
	}
	debug.Log("Stream: rate=%dms", rate)

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
	debug.Log("Stream: subscribed to events")

	ticker := time.NewTicker(rate * time.Millisecond)
	ticker.Reset(rate * time.Millisecond)
	defer ticker.Stop()

	rawDataPtr := rawFrameBuffer.Get().(*[]uint8)
	rawData := *rawDataPtr
	defer rawFrameBuffer.Put(rawDataPtr)
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
			debug.Log("Stream: client disconnected (%s)", r.RemoteAddr)
			return
		case event := <-eventC:
			if event.Code == 24 || event.Source == events.Touch {
				if !writing {
					debug.Log("Stream: writing resumed (input event code=%d, source=%v)", event.Code, event.Source)
				}
				writing = true
				stopWriting.Reset(2000 * time.Millisecond)
			}
		case <-stopWriting.C:
			if writing {
				debug.Log("Stream: writing paused (no input for 2s)")
			}
			writing = false
		case <-ticker.C:
			if writing {
				h.fetchAndSendDelta(w, rawData)
			}
		}
	}
}

func (h *StreamHandler) fetchAndSendDelta(w io.Writer, rawData []uint8) {
	_, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println(err)
		return
	}
	frameSize, err := h.deltaEncoder.EncodeWithSize(rawData, w)
	if err != nil {
		log.Println("Error in delta encoding", err)
		return
	}
	debug.Log("Stream: sent frame (%d bytes)", frameSize)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
