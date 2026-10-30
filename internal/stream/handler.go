package stream

import (
	"context"
	"encoding/binary"
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
	// rm1PressureThreshold matches the Wacom digitizer hover baseline on reMarkable 1
	// (0..1000 = hovering / laser pointer, >1000 = pen touching screen).
	rm1PressureThreshold int32 = 1000
	// penLiftCooldown is the grace period after pen lift during which we continue
	// streaming to flush buffered frames and catch late xochitl renders.
	penLiftCooldown    = 300 * time.Millisecond
	rm1PenLiftCooldown = 500 * time.Millisecond
	// rm1IdlePollInterval is the low-duty background check interval on reMarkable 1
	// when neither pen nor touch is active. Because the rM1 has physical hardware
	// page-turn buttons (/dev/input/event1) that do not emit EvAbs events, a 1 Hz
	// single-buffer NEON checksum check detects button page turns while sending
	// 0 bytes over the network when the screen is static.
	rm1IdlePollInterval = 1000 * time.Millisecond
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

type encoderMode uint8

const (
	encoderModeBGRA encoderMode = iota
	encoderModeColor8
	encoderModeRGB565
)

// NewStreamHandler creates a new stream handler reading from file @pointerAddr
func NewStreamHandler(file io.ReaderAt, pointerAddr int64, inputEvents *pubsub.PubSub, deltaThreshold float64) *StreamHandler {
	h := &StreamHandler{
		file:           file,
		pointerAddr:    pointerAddr,
		inputEventsBus: inputEvents,
		deltaThreshold: deltaThreshold,
	}
	if remarkable.Model == remarkable.Remarkable1 {
		h.deltaEncoder = delta.NewColor8Encoder(deltaThreshold)
		h.encMode = encoderModeColor8
	} else {
		h.deltaEncoder = delta.NewEncoder(deltaThreshold)
		h.encMode = encoderModeBGRA
	}
	return h
}

// StreamHandler is an http.Handler that serves the stream of data to the client
type StreamHandler struct {
	file           io.ReaderAt
	pointerAddr    int64
	inputEventsBus *pubsub.PubSub
	deltaThreshold float64

	// session guards deltaEncoder, which is per-connection state: the previous
	// frame it diffs against and its scratch buffers. ThrottlingMiddleware
	// admits one streamer at a time, but net/http may call any http.Handler
	// concurrently, so the handler has to hold on its own.
	session      sync.Mutex
	deltaEncoder *delta.Encoder
	encMode      encoderMode
	UseRLE       bool
	RM1RGB565    bool
}

// ensureEncoder selects the appropriate delta encoder for the current device and
// configuration, preserving allocated buffers across connections when the mode
// has not changed. Must be called with h.session held.
func (h *StreamHandler) ensureEncoder() {
	want := encoderModeBGRA
	if remarkable.Model == remarkable.Remarkable1 && !h.UseRLE {
		if h.RM1RGB565 {
			want = encoderModeRGB565
		} else {
			want = encoderModeColor8
		}
	}
	if h.deltaEncoder == nil || h.encMode != want {
		switch want {
		case encoderModeColor8:
			h.deltaEncoder = delta.NewColor8Encoder(h.deltaThreshold)
		case encoderModeRGB565:
			h.deltaEncoder = delta.NewRGB565Encoder(h.deltaThreshold)
		default:
			h.deltaEncoder = delta.NewEncoder(h.deltaThreshold)
		}
		h.encMode = want
	}
	h.deltaEncoder.Reset()
}

// ReleaseMemory releases large buffers held by the stream handler's delta encoder.
func (h *StreamHandler) ReleaseMemory() {
	// A live stream owns those buffers; freeing them mid-session would only
	// force an immediate reallocation, so skipping is the right answer.
	if !h.session.TryLock() {
		return
	}
	defer h.session.Unlock()
	if h.deltaEncoder != nil {
		h.deltaEncoder.ReleaseMemory()
	}
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug.Log("Stream: new connection from %s", r.RemoteAddr)

	// Parse query parameters - each client gets its own rate
	rate := defaultRate
	query := r.URL.Query()
	rateStr := query.Get("rate")
	if rateStr != "" {
		rateInt, err := strconv.Atoi(rateStr)
		if err != nil {
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

	// Configure and reset delta encoder to force a full frame for the new client
	h.ensureEncoder()

	// Subscribe only to EvAbs events (pen position and pressure)
	absType := uint16(events.EvAbs)
	eventC := h.inputEventsBus.SubscribeWithFilter("stream", pubsub.EventFilter{
		Type: &absType,
	})
	defer h.inputEventsBus.Unsubscribe(eventC)
	debug.Log("Stream: subscribed to EvAbs events")

	ticker := time.NewTicker(rate * time.Millisecond)
	defer ticker.Stop()

	isRM1 := remarkable.Model == remarkable.Remarkable1
	useRLE := isRM1 && h.UseRLE

	activePressureThreshold := pressureThreshold
	activeCooldown := penLiftCooldown
	if isRM1 {
		activePressureThreshold = rm1PressureThreshold
		activeCooldown = rm1PenLiftCooldown
	}

	// On rM1, prefer reading directly from mmap'd /dev/fb0 (DirectFramebuffer)
	// with zero syscalls and zero extra 5.27MB scratch buffer. Fall back to a
	// pooled buffer when ReadAt is required or when legacy RLE mutates in-place.
	var directBuf []byte
	if isRM1 && !useRLE {
		if df, ok := h.file.(remarkable.DirectFramebuffer); ok {
			if b := df.Bytes(); len(b) == remarkable.Config.SizeBytes && len(b) > 0 {
				directBuf = b
			}
		}
	}

	var rawData []uint8
	if isRM1 && (useRLE || directBuf == nil) {
		rawDataPtr, ok := rawFrameBuffer.Get().(*[]uint8)
		if !ok || rawDataPtr == nil {
			http.Error(w, "Internal error: buffer pool returned invalid value", http.StatusInternalServerError)
			return
		}
		rawData = *rawDataPtr
		defer rawFrameBuffer.Put(rawDataPtr)
	}

	var frameWriter io.Writer = w
	if useRLE {
		frameWriter = rle.NewRLE(w)
	}

	// Start async frame reader ONLY for dual/quad-core devices (RM2 / Paper Pro).
	// On the single-core rM1 (Cortex-A9), a background reader goroutine competes
	// with xochitl and the encoder on the same core and wastes 15.8MB of RAM.
	var asyncReader *AsyncFrameReader
	if !isRM1 {
		asyncCtx, asyncCancel := context.WithCancel(r.Context())
		defer asyncCancel()
		asyncReader = NewAsyncFrameReader(h.file, h.pointerAddr, remarkable.Config.SizeBytes)
		asyncReader.SetInterval(rate * time.Millisecond)
		go asyncReader.Run(asyncCtx)
		asyncReader.Resume()
	}

	writing := true
	var lastRLEHash uint64
	hasLastRLEHash := false
	stopWriting := time.NewTicker(2 * time.Second)
	defer stopWriting.Stop()

	cooldownTimer := time.NewTimer(0)
	if !cooldownTimer.Stop() {
		<-cooldownTimer.C
	}
	defer cooldownTimer.Stop()
	cooldownActive := false

	var currentPressure int32

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Send initial full frame immediately on rM1 so the client renders right away
	// without waiting an extra `rate` ms for the first ticker cycle.
	if isRM1 {
		if useRLE {
			h.fetchAndSendIfChanged(frameWriter, rawData, &lastRLEHash, &hasLastRLEHash)
		} else if frameSize := h.sendRM1DeltaActive(w, directBuf, rawData); frameSize > 0 {
			ticker.Reset(adaptRate(frameSize, rate*time.Millisecond))
		}
	}

	for {
		select {
		case <-r.Context().Done():
			debug.Log("Stream: client disconnected (%s)", r.RemoteAddr)
			return
		case event := <-eventC:
			if event.Code == 24 {
				currentPressure = event.Value
			}

			shouldWrite := false
			if event.Source == events.Touch {
				shouldWrite = true
			} else if event.Source == events.Pen && currentPressure > activePressureThreshold {
				shouldWrite = true
			}

			if shouldWrite {
				if !writing {
					debug.Log("Stream: writing resumed (source=%v, pressure=%d)", event.Source, currentPressure)
					if asyncReader != nil {
						asyncReader.Resume()
					}
					if isRM1 {
						// Wake immediately from 1 Hz idle poll so the first stroke
						// frame is captured with minimal latency.
						ticker.Reset(10 * time.Millisecond)
					}
				}
				writing = true
				stopWriting.Reset(2000 * time.Millisecond)
				if cooldownActive {
					cooldownTimer.Stop()
					cooldownActive = false
				}
			} else if writing && event.Source == events.Pen && currentPressure <= activePressureThreshold {
				if !cooldownActive {
					debug.Log("Stream: pen lifted, starting cooldown (pressure=%d)", currentPressure)
					cooldownTimer.Reset(activeCooldown)
					cooldownActive = true
				}
			}
		case <-stopWriting.C:
			if writing {
				debug.Log("Stream: writing paused (no input for 2s)")
				if asyncReader != nil {
					asyncReader.Pause()
				}
				if isRM1 {
					ticker.Reset(rm1IdlePollInterval)
				}
			}
			writing = false
			cooldownActive = false
		case <-cooldownTimer.C:
			if writing && cooldownActive {
				debug.Log("Stream: writing paused (pen lift cooldown expired)")
				writing = false
				if asyncReader != nil {
					asyncReader.Pause()
				}
				if isRM1 {
					ticker.Reset(rm1IdlePollInterval)
				}
			}
			cooldownActive = false
		case <-ticker.C:
			if useRLE {
				if h.fetchAndSendIfChanged(frameWriter, rawData, &lastRLEHash, &hasLastRLEHash) && !writing {
					writing = true
					stopWriting.Reset(2000 * time.Millisecond)
					ticker.Reset(rate * time.Millisecond)
				}
			} else if isRM1 {
				if writing {
					next := rate * time.Millisecond
					if frameSize := h.sendRM1DeltaActive(w, directBuf, rawData); frameSize > 0 {
						next = adaptRate(frameSize, rate*time.Millisecond)
					}
					ticker.Reset(next)
				} else {
					// Low-duty 1 Hz idle check: emits 0 bytes when unchanged, or
					// wakes into active adaptive-rate streaming if the screen changed
					// (e.g., physical Next/Prev page button press on rM1).
					if frameSize, changed := h.sendRM1DeltaIdle(w, directBuf, rawData); changed && frameSize > 0 {
						debug.Log("Stream: rM1 screen change detected during idle (%d bytes), resuming active rate", frameSize)
						writing = true
						stopWriting.Reset(2000 * time.Millisecond)
						ticker.Reset(adaptRate(frameSize, rate*time.Millisecond))
					}
				}
			} else if writing && asyncReader != nil {
				next := rate * time.Millisecond
				if frameSize := h.fetchAndSendDeltaAsync(w, asyncReader); frameSize > 0 {
					next = adaptRate(frameSize, rate*time.Millisecond)
				}
				ticker.Reset(next)
				asyncReader.SetInterval(next)
			}
		}
	}
}

// sendRM1DeltaActive encodes and flushes a frame during active rM1 drawing,
// reading directly from mmap'd /dev/fb0 when available.
func (h *StreamHandler) sendRM1DeltaActive(w io.Writer, directBuf, rawData []byte) int {
	if directBuf != nil {
		span := trace.BeginSpan("fetch_and_send")
		defer trace.EndSpan(span, nil)

		frameSize, err := h.deltaEncoder.EncodeWithSize(directBuf, w)
		if err != nil {
			log.Println("Error in delta encoding", err)
			return 0
		}
		debug.Log("Stream: sent rM1 mmap frame (%d bytes)", frameSize)
		flush(w)
		return frameSize
	}
	return h.fetchAndSendDelta(w, rawData)
}

// sendRM1DeltaIdle checks for framebuffer changes during rM1 idle mode and only
// writes to the HTTP stream when a change actually occurred (0 idle network bytes).
func (h *StreamHandler) sendRM1DeltaIdle(w io.Writer, directBuf, rawData []byte) (int, bool) {
	span := trace.BeginSpan("fetch_and_send_idle")
	defer trace.EndSpan(span, nil)

	buf := directBuf
	if buf == nil {
		n, err := h.file.ReadAt(rawData, h.pointerAddr)
		if err != nil {
			log.Println("Error reading framebuffer:", err)
			for i := range rawData[:n] {
				rawData[i] = 0
			}
			return 0, false
		}
		buf = rawData
	}

	frameSize, changed, err := h.deltaEncoder.EncodeIfChanged(buf, w)
	if err != nil {
		log.Println("Error in delta encoding", err)
		return 0, false
	}
	if changed {
		flush(w)
	}
	return frameSize, changed
}

// fetchAndSendIfChanged handles the optional legacy RLE path on rM1 when
// RK_RLE_COMPRESSION=true. Uses a position-dependent 64-bit hash and LUT
// conversion instead of byte-by-byte sum() and per-pixel integer division.
func (h *StreamHandler) fetchAndSendIfChanged(w io.Writer, rawData []uint8, lastHash *uint64, hasHash *bool) bool {
	_, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println(err)
		return false
	}
	curHash := fastFrameHash64(rawData)
	if *hasHash && curHash == *lastHash {
		return false
	}
	*lastHash = curHash
	*hasHash = true

	if remarkable.Model == remarkable.Remarkable1 {
		delta.ConvertRGB565InPlace(rawData)
	}
	_, err = w.Write(rawData)
	if err != nil {
		log.Println("Error in writing", err)
		return false
	}
	flush(w)
	return true
}

// fastFrameHash64 computes a fast position-dependent 64-bit XOR-rotate checksum
// in 8-byte chunks (avoiding permutation collisions of byte-sum).
func fastFrameHash64(data []byte) uint64 {
	var acc uint64
	i := 0
	n := len(data) &^ 7
	for ; i < n; i += 8 {
		w := binary.LittleEndian.Uint64(data[i : i+8])
		acc = ((acc << 7) | (acc >> 57)) ^ w
	}
	for ; i < len(data); i++ {
		acc = ((acc << 7) | (acc >> 57)) ^ uint64(data[i])
	}
	return acc
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
// Used by fallback paths, tests, and benchmarks that don't use the async reader.
func (h *StreamHandler) fetchAndSendDelta(w io.Writer, rawData []uint8) int {
	span := trace.BeginSpan("fetch_and_send")
	defer trace.EndSpan(span, nil)

	n, err := h.file.ReadAt(rawData, h.pointerAddr)
	if err != nil {
		log.Println("Error reading framebuffer:", err)
		for i := range rawData[:n] {
			rawData[i] = 0
		}
		return 0
	}
	if h.deltaEncoder == nil {
		h.deltaEncoder = delta.NewEncoder(h.deltaThreshold)
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
