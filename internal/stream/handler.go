package stream

import (
	"context"
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
	"github.com/owulveryck/goMarkableStream/internal/trace"
)

var (
	defaultRate time.Duration = 200
	// pressureThreshold defines the minimum pressure value to consider the pen as "touching"
	// Values below this are considered "hovering" and should not trigger frame streaming
	pressureThreshold int32 = 100
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
	flusher        http.Flusher // Cached flusher interface per connection
}

// ReleaseMemory releases large buffers held by the stream handler's delta encoder.
func (h *StreamHandler) ReleaseMemory() {
	h.deltaEncoder.ReleaseMemory()
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug.Log("Stream: new connection from %s", r.RemoteAddr)

	// Cache flusher interface for this connection
	h.flusher, _ = w.(http.Flusher)

	// Reset delta encoder to force a full frame for the new client
	h.deltaEncoder.Reset()

	// Parse query parameters - each client gets its own rate
	rate := defaultRate
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
	if rate < 1 {
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

	// Subscribe only to EvAbs events (pen position and pressure)
	// This filters out unnecessary EvSyn, EvKey, etc.
	absType := uint16(events.EvAbs)
	eventC := h.inputEventsBus.SubscribeWithFilter("stream", pubsub.EventFilter{
		Type: &absType,
	})
	defer h.inputEventsBus.Unsubscribe(eventC)
	debug.Log("Stream: subscribed to EvAbs events")

	ticker := time.NewTicker(rate * time.Millisecond)
	defer ticker.Stop()

	// Start async frame reader: a background goroutine continuously reads
	// the framebuffer using triple buffering, so the ReadAt I/O overlaps
	// with delta encoding on the Cortex-A9's second core.
	asyncCtx, asyncCancel := context.WithCancel(r.Context())
	defer asyncCancel()
	asyncReader := NewAsyncFrameReader(h.file, h.pointerAddr, remarkable.Config.SizeBytes)
	go asyncReader.Run(asyncCtx)

	writing := true
	asyncReader.Resume() // start reading since writing defaults to true
	stopWriting := time.NewTicker(2 * time.Second)
	defer stopWriting.Stop()

	// Track current pressure value to distinguish hover from touch
	var currentPressure int32

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
			// Track pressure value from ABS_PRESSURE events (code 24)
			if event.Code == 24 {
				currentPressure = event.Value
			}

			// Only trigger frame streaming when:
			// 1. Touch events (finger touch, always active)
			// 2. Pen events with pressure above threshold (pen touching, not hovering)
			shouldWrite := false
			if event.Source == events.Touch {
				shouldWrite = true
			} else if event.Source == events.Pen && currentPressure > pressureThreshold {
				shouldWrite = true
			}

			if shouldWrite {
				if !writing {
					debug.Log("Stream: writing resumed (source=%v, pressure=%d)", event.Source, currentPressure)
					asyncReader.Resume()
				}
				writing = true
				stopWriting.Reset(2000 * time.Millisecond)
			} else if writing && event.Source == events.Pen && currentPressure <= pressureThreshold {
				// Pen lifted or hovering - stop writing immediately
				debug.Log("Stream: writing paused (pen hover/lifted, pressure=%d)", currentPressure)
				writing = false
				asyncReader.Pause()
			}
		case <-stopWriting.C:
			if writing {
				debug.Log("Stream: writing paused (no input for 2s)")
				asyncReader.Pause()
			}
			writing = false
		case <-ticker.C:
			if writing {
				if frameSize := h.fetchAndSendDeltaAsync(w, asyncReader); frameSize > 0 {
					ticker.Reset(adaptRate(frameSize, rate*time.Millisecond))
				}
			}
		}
	}
}

// fetchAndSendDelta reads the framebuffer synchronously and sends a delta-encoded frame.
// Used by tests and benchmarks that don't need the async reader.
func (h *StreamHandler) fetchAndSendDelta(w io.Writer, rawData []uint8) int {
	span := trace.BeginSpan("fetch_and_send")
	defer trace.EndSpan(span, nil)

	n, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println("Error reading framebuffer:", err)
		// Clear buffer to avoid sending stale data
		for i := range rawData[:n] {
			rawData[i] = 0
		}
		return 0
	}
	frameSize, err := h.deltaEncoder.EncodeWithSize(rawData, w)
	if err != nil {
		log.Println("Error in delta encoding", err)
		return 0
	}
	debug.Log("Stream: sent frame (%d bytes)", frameSize)
	if h.flusher != nil {
		h.flusher.Flush()
	}
	return frameSize
}

// adaptRate adjusts the frame ticker interval based on the last encoded frame size.
// Small deltas get faster updates for responsiveness, large deltas get slower
// updates to avoid saturating the network.
func adaptRate(frameSize int, baseRate time.Duration) time.Duration {
	const (
		minRate        = 50 * time.Millisecond
		maxRate        = 1000 * time.Millisecond
		smallThreshold = 50_000  // 50KB
		largeThreshold = 200_000 // 200KB
	)
	switch {
	case frameSize <= smallThreshold:
		r := baseRate / 2
		if r < minRate {
			r = minRate
		}
		return r
	case frameSize <= largeThreshold:
		return baseRate
	default:
		r := baseRate * 2
		if r > maxRate {
			r = maxRate
		}
		return r
	}
}

func (h *StreamHandler) fetchAndSendDeltaAsync(w io.Writer, reader *AsyncFrameReader) int {
	frame := reader.Latest()
	if frame == nil {
		return 0 // no new frame available yet
	}

	span := trace.BeginSpan("fetch_and_send")
	defer trace.EndSpan(span, nil)

	frameSize, err := h.deltaEncoder.EncodeWithSize(frame, w)
	if err != nil {
		log.Println("Error in delta encoding", err)
		return 0
	}
	debug.Log("Stream: sent frame (%d bytes)", frameSize)
	if h.flusher != nil {
		h.flusher.Flush()
	}
	return frameSize
}
