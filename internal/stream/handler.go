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
	"github.com/owulveryck/goMarkableStream/internal/rle"
	"github.com/owulveryck/goMarkableStream/internal/trace"
)

var (
	defaultRate time.Duration = 200
	// pressureThreshold defines the minimum pressure value to consider the pen as "touching"
	// Values below this are considered "hovering" and should not trigger frame streaming
	pressureThreshold int32 = 100
	// penLiftCooldown is the grace period after pen lift during which we continue
	// streaming to flush buffered frames and catch late xochitl renders.
	penLiftCooldown = 300 * time.Millisecond
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

	// session guards deltaEncoder, which is per-connection state: the previous
	// frame it diffs against and its scratch buffers. ThrottlingMiddleware
	// admits one streamer at a time, but net/http may call any http.Handler
	// concurrently, so the handler has to hold on its own.
	session      sync.Mutex
	deltaEncoder *delta.Encoder
	UseRLE       bool
}

// ReleaseMemory releases large buffers held by the stream handler's delta encoder.
func (h *StreamHandler) ReleaseMemory() {
	// A live stream owns those buffers; freeing them mid-session would only
	// force an immediate reallocation, so skipping is the right answer.
	if !h.session.TryLock() {
		return
	}
	defer h.session.Unlock()
	h.deltaEncoder.ReleaseMemory()
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug.Log("Stream: new connection from %s", r.RemoteAddr)

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

	// Claim the encoder for this connection. Taken after the preflight branch so
	// a CORS check never gets turned away by an in-flight stream.
	if !h.session.TryLock() {
		debug.Log("Stream: rejecting %s, another connection owns the encoder", r.RemoteAddr)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	defer h.session.Unlock()

	// Reset delta encoder to force a full frame for the new client
	h.deltaEncoder.Reset()

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

	// On the rM1 there are no xochitl input events, so we poll the
	// framebuffer on every tick and use a checksum to skip unchanged frames.
	useChangeDetection := remarkable.Model == remarkable.Remarkable1
	useRLE := useChangeDetection && h.UseRLE

	// For rM1 change detection / raw buffers
	var rawData []uint8
	if useChangeDetection {
		rawDataPtr, ok := rawFrameBuffer.Get().(*[]uint8)
		if !ok || rawDataPtr == nil {
			http.Error(w, "Internal error: buffer pool returned invalid value", http.StatusInternalServerError)
			return
		}
		rawData = *rawDataPtr
		defer rawFrameBuffer.Put(rawDataPtr)
	}

	// Select the appropriate writer once (RLE or raw) for RM1.
	var frameWriter io.Writer = w
	if useRLE {
		frameWriter = rle.NewRLE(w)
	}

	// Start async frame reader for RM2 / RMPRO: a background goroutine continuously reads
	// the framebuffer using triple buffering, so the ReadAt I/O overlaps
	// with delta encoding on the Cortex-A9's second core.
	var asyncReader *AsyncFrameReader
	if !useChangeDetection {
		asyncCtx, asyncCancel := context.WithCancel(r.Context())
		defer asyncCancel()
		asyncReader = NewAsyncFrameReader(h.file, h.pointerAddr, remarkable.Config.SizeBytes)
		asyncReader.SetInterval(rate * time.Millisecond)
		go asyncReader.Run(asyncCtx)
		asyncReader.Resume() // start reading since writing defaults to true
	}

	writing := true
	lastSum := 0
	stopWriting := time.NewTicker(2 * time.Second)
	defer stopWriting.Stop()

	// Cooldown timer: after pen lift, keep streaming for a grace period
	// to flush buffered frames and catch late xochitl renders.
	cooldownTimer := time.NewTimer(0)
	if !cooldownTimer.Stop() {
		<-cooldownTimer.C
	}
	defer cooldownTimer.Stop()
	cooldownActive := false

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
					if asyncReader != nil {
						asyncReader.Resume()
					}
				}
				writing = true
				stopWriting.Reset(2000 * time.Millisecond)
				// Cancel any pending pen-lift cooldown
				if cooldownActive {
					cooldownTimer.Stop()
					cooldownActive = false
				}
			} else if writing && event.Source == events.Pen && currentPressure <= pressureThreshold {
				// Pen lifted or hovering - start cooldown instead of stopping immediately.
				// This grace period flushes buffered frames and catches late xochitl renders.
				if !cooldownActive {
					debug.Log("Stream: pen lifted, starting cooldown (pressure=%d)", currentPressure)
					cooldownTimer.Reset(penLiftCooldown)
					cooldownActive = true
				}
			}
		case <-stopWriting.C:
			if !useChangeDetection {
				if writing {
					debug.Log("Stream: writing paused (no input for 2s)")
					if asyncReader != nil {
						asyncReader.Pause()
					}
				}
				writing = false
				cooldownActive = false
			}
		case <-cooldownTimer.C:
			if !useChangeDetection {
				if writing && cooldownActive {
					debug.Log("Stream: writing paused (pen lift cooldown expired)")
					writing = false
					if asyncReader != nil {
						asyncReader.Pause()
					}
				}
				cooldownActive = false
			}
		case <-ticker.C:
			if useChangeDetection {
				h.fetchAndSendIfChanged(frameWriter, rawData, &lastSum)
			} else if writing && asyncReader != nil {
				next := rate * time.Millisecond
				if frameSize := h.fetchAndSendDeltaAsync(w, asyncReader); frameSize > 0 {
					next = adaptRate(frameSize, rate*time.Millisecond)
				}
				ticker.Reset(next)
				// Keep the background reader paced to whatever rate we settled on.
				asyncReader.SetInterval(next)
			}
		}
	}
}

func (h *StreamHandler) fetchAndSendIfChanged(w io.Writer, rawData []uint8, lastSum *int) {
	_, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println(err)
		return
	}
	s := sum(rawData)
	if s == *lastSum {
		return
	}
	*lastSum = s
	// For rM1, convert RGB565 to the RLE-compatible color-coded format.
	// The even bytes (which the RLE encoder reads at stride 2) are
	// overwritten with the converted pixel values.
	if remarkable.Model == remarkable.Remarkable1 {
		convertRGB565InPlace(rawData)
	}
	_, err = w.Write(rawData)
	if err != nil {
		log.Println("Error in writing", err)
		return
	}
	flush(w)
}

// convertRGB565InPlace converts RGB565 little-endian pixel data to the
// single-byte color-coded format expected by the RLE encoder. Each even
// byte (the one the RLE reads) is overwritten with the converted value.
func convertRGB565InPlace(data []uint8) {
	for i := 0; i < len(data)-1; i += 2 {
		data[i] = rgb565ToColorByte(data[i], data[i+1])
	}
}

// rgb565ToColorByte converts one RGB565 LE pixel (lo, hi bytes) to the
// single-byte encoding used by the RLE stream and decoded on the client:
//
//	0-25  → grayscale (client renders value*10 per channel)
//	6,8   → red
//	12    → blue
//	20    → green
//	24    → yellow
//	30    → transparent (paper/background)
func rgb565ToColorByte(lo, hi byte) byte {
	// Extract 5-6-5 components
	r5 := (hi >> 3) & 0x1F
	g6 := uint16((hi&0x07)<<3) | uint16((lo>>5)&0x07)
	b5 := lo & 0x1F

	// Expand to 8-bit
	r := uint16(r5) * 255 / 31
	g := g6 * 255 / 63
	b := uint16(b5) * 255 / 31

	// Detect pen colors (approximate thresholds)
	if r > 180 && g < 80 && b < 80 {
		return 6 // Red
	}
	if r < 80 && g < 80 && b > 180 {
		return 12 // Blue
	}
	if g > 120 && r < 160 && b < 120 && g > r && g > b {
		return 20 // Green
	}
	if r > 180 && g > 180 && b < 120 {
		return 24 // Yellow
	}

	// Grayscale
	gray := (r + g + b) / 3
	if gray > 245 {
		return 30 // White → transparent (matches rM2 paper)
	}
	scaled := byte(gray * 25 / 245)
	// Avoid values the client interprets as colors
	switch scaled {
	case 6, 8, 12, 20, 24:
		scaled++
	}
	return scaled
}

func (h *StreamHandler) fetchAndSend(w io.Writer, rawData []uint8) {
	_, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = w.Write(rawData)
	if err != nil {
		log.Println("Error in writing", err)
		return
	}
	flush(w)
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
	flush(w)
	return frameSize
}

// flush pushes the frame out if the writer buffers. Asserting per frame costs a
// few nanoseconds next to encoding a 10 MB framebuffer, and keeps the flusher
// out of StreamHandler, where it was shared state across connections.
func flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
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
	flush(w)
	return frameSize
}

func sum(d []uint8) int {
	val := 0
	for _, v := range d {
		val += int(v)
	}
	return val
}
