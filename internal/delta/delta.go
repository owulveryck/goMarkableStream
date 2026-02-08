// Package delta provides delta compression for BGRA framebuffer streaming.
// It encodes only changed pixels between frames, significantly reducing bandwidth
// for e-ink usage where typically only 1-5% of pixels change between frames.
package delta

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"

	"github.com/klauspost/compress/zstd"
	"github.com/owulveryck/goMarkableStream/internal/debug"
)

// zstdEncoderPool reuses zstd encoders to avoid allocation overhead
var zstdEncoderPool = sync.Pool{
	New: func() any {
		enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		return enc
	},
}

const (
	// Frame type constants for wire protocol
	FrameTypeFull           = 0x00 // Deprecated: uncompressed full frame
	FrameTypeDelta          = 0x01
	FrameTypeFullCompressed = 0x02 // Gzip-compressed full frame (legacy)
	FrameTypeFullZstd       = 0x03 // Zstd-compressed full frame

	// DefaultThreshold is the default change ratio above which a full frame is sent
	DefaultThreshold = 0.30

	// Maximum values for short run encoding
	maxShortOffset = 0xFFFF   // 64KB - 1
	maxShortLength = 127      // 7 bits
	bytesPerPixel  = 4        // BGRA format
)

// Encoder holds the state for delta encoding between frames.
type Encoder struct {
	prevFrame []byte
	threshold float64
	hasPrev   bool
	// Reusable buffers to avoid allocations
	headerBuf   [5]byte       // Max header size (long run: 5 bytes)
	frameHeader [4]byte       // Frame header buffer
	runsBuf     []changeRun   // Reusable runs slice
}

// NewEncoder creates a new delta encoder with the given threshold.
// Threshold is the change ratio (0.0-1.0) above which a full frame is sent.
func NewEncoder(threshold float64) *Encoder {
	if threshold <= 0 || threshold > 1.0 {
		threshold = DefaultThreshold
	}
	return &Encoder{
		threshold: threshold,
	}
}

// changeRun represents a contiguous run of changed pixels.
// Uses slice reference to avoid copying data.
type changeRun struct {
	offset int    // byte offset from previous run end (or frame start)
	length int    // number of changed pixels
	data   []byte // slice into current frame (not a copy)
}

// Encode writes the current frame to w, using delta encoding if beneficial.
// Returns nil on success, or an error if writing fails.
func (e *Encoder) Encode(current []byte, w io.Writer) error {
	_, err := e.EncodeWithSize(current, w)
	return err
}

// EncodeWithSize writes the current frame to w, using delta encoding if beneficial.
// Returns the number of bytes written and nil on success, or 0 and an error if writing fails.
func (e *Encoder) EncodeWithSize(current []byte, w io.Writer) (int, error) {
	frameSize := len(current)

	// First frame or no previous: send full frame
	if !e.hasPrev || len(e.prevFrame) != frameSize {
		e.prevFrame = make([]byte, frameSize)
		copy(e.prevFrame, current)
		e.hasPrev = true
		debug.Log("Delta: first frame, sending full")
		return e.writeFullFrame(current, w)
	}

	// Compare frames and build change runs
	runs := e.compareFrames(current)

	// Calculate total changed bytes
	changedBytes := 0
	for _, run := range runs {
		changedBytes += len(run.data)
	}

	// No changes - send empty delta frame without copying
	if changedBytes == 0 {
		debug.Log("Delta: no changes, sending empty delta")
		return e.writeDeltaFrame(runs, 0, w)
	}

	// Calculate change ratio
	changeRatio := float64(changedBytes) / float64(frameSize)

	// Calculate delta payload size
	deltaSize := e.calculateDeltaSize(runs)

	// If change ratio exceeds threshold OR delta is larger than full frame, send full frame
	if changeRatio > e.threshold || deltaSize >= frameSize {
		copy(e.prevFrame, current)
		debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending full", changeRatio*100, len(runs))
		return e.writeFullFrame(current, w)
	}

	// Send delta frame - only copy prevFrame when we actually have changes
	copy(e.prevFrame, current)
	debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending delta", changeRatio*100, len(runs))
	return e.writeDeltaFrame(runs, deltaSize, w)
}

// compareFrames compares current frame with previous and returns change runs.
// Uses uint64 comparisons for speed. Reuses internal buffer to minimize allocations.
// The returned slice references data in 'current', so current must not be modified
// until after the runs are processed.
func (e *Encoder) compareFrames(current []byte) []changeRun {
	// Reuse runs buffer, reset length but keep capacity
	e.runsBuf = e.runsBuf[:0]

	prev := e.prevFrame
	frameLen := len(current)

	// Compare 8 bytes at a time using unsafe pointer casting
	numQwords := frameLen / 8

	var runStart int = -1
	var lastDiffEnd int = 0

	// Get pointers for fast comparison
	currPtr := unsafe.Pointer(&current[0])
	prevPtr := unsafe.Pointer(&prev[0])

	for i := range numQwords {
		offset := i * 8
		currQword := *(*uint64)(unsafe.Add(currPtr, offset))
		prevQword := *(*uint64)(unsafe.Add(prevPtr, offset))

		if currQword != prevQword {
			if runStart == -1 {
				runStart = offset
			}
			lastDiffEnd = offset + 8
		} else {
			if runStart != -1 {
				// End of a change run - align to pixel boundaries
				alignedStart := (runStart / bytesPerPixel) * bytesPerPixel
				alignedEnd := ((lastDiffEnd + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
				alignedEnd = min(alignedEnd, frameLen)

				// Use slice reference into current frame (no copy)
				e.runsBuf = append(e.runsBuf, changeRun{
					offset: alignedStart,
					length: (alignedEnd - alignedStart) / bytesPerPixel,
					data:   current[alignedStart:alignedEnd],
				})
				runStart = -1
			}
		}
	}

	// Handle remainder bytes
	for i := numQwords * 8; i < frameLen; i++ {
		if current[i] != prev[i] {
			if runStart == -1 {
				runStart = i
			}
			lastDiffEnd = i + 1
		}
	}

	// Finalize any remaining run
	if runStart != -1 {
		alignedStart := (runStart / bytesPerPixel) * bytesPerPixel
		alignedEnd := ((lastDiffEnd + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
		alignedEnd = min(alignedEnd, frameLen)

		// Use slice reference into current frame (no copy)
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: alignedStart,
			length: (alignedEnd - alignedStart) / bytesPerPixel,
			data:   current[alignedStart:alignedEnd],
		})
	}

	// Convert absolute offsets to relative offsets
	lastEnd := 0
	for i := range e.runsBuf {
		absOffset := e.runsBuf[i].offset
		e.runsBuf[i].offset = absOffset - lastEnd
		lastEnd = absOffset + len(e.runsBuf[i].data)
	}

	return e.runsBuf
}

// calculateDeltaSize calculates the size of the delta payload.
func (e *Encoder) calculateDeltaSize(runs []changeRun) int {
	size := 0
	for _, run := range runs {
		if run.offset <= maxShortOffset && run.length <= maxShortLength {
			// Short run: 1 byte length + 2 bytes offset + pixel data
			size += 1 + 2 + len(run.data)
		} else {
			// Long run: 2 bytes length + 3 bytes offset + pixel data
			size += 2 + 3 + len(run.data)
		}
	}
	return size
}

// writeFullFrame writes a zstd-compressed full frame with header.
// Returns the total number of bytes written.
func (e *Encoder) writeFullFrame(data []byte, w io.Writer) (int, error) {
	// Compress data with zstd using pooled encoder
	enc := zstdEncoderPool.Get().(*zstd.Encoder)
	compressed := enc.EncodeAll(data, nil)
	zstdEncoderPool.Put(enc)

	// Write header with zstd compressed type
	e.frameHeader[0] = FrameTypeFullZstd
	// Payload length in 24-bit little-endian
	payloadLen := len(compressed)
	e.frameHeader[1] = byte(payloadLen & 0xFF)
	e.frameHeader[2] = byte((payloadLen >> 8) & 0xFF)
	e.frameHeader[3] = byte((payloadLen >> 16) & 0xFF)

	if _, err := w.Write(e.frameHeader[:]); err != nil {
		return 0, err
	}
	n, err := w.Write(compressed)
	if err != nil {
		return 4, err
	}
	return 4 + n, nil
}

// writeDeltaFrame writes a delta frame with header and change runs.
// Returns the total number of bytes written.
func (e *Encoder) writeDeltaFrame(runs []changeRun, payloadSize int, w io.Writer) (int, error) {
	e.frameHeader[0] = FrameTypeDelta
	// Payload length in 24-bit little-endian
	e.frameHeader[1] = byte(payloadSize & 0xFF)
	e.frameHeader[2] = byte((payloadSize >> 8) & 0xFF)
	e.frameHeader[3] = byte((payloadSize >> 16) & 0xFF)

	if _, err := w.Write(e.frameHeader[:]); err != nil {
		return 0, err
	}

	// Write each change run
	for _, run := range runs {
		if run.offset <= maxShortOffset && run.length <= maxShortLength {
			// Short run format
			if err := e.writeShortRun(run, w); err != nil {
				return 0, err
			}
		} else {
			// Long run format
			if err := e.writeLongRun(run, w); err != nil {
				return 0, err
			}
		}
	}

	return 4 + payloadSize, nil
}

// writeShortRun writes a short run (offset < 64KB, length <= 127 pixels).
// Format: [1 byte: length] [2 bytes: relative offset LE] [N bytes: pixel data]
func (e *Encoder) writeShortRun(run changeRun, w io.Writer) error {
	e.headerBuf[0] = byte(run.length)
	binary.LittleEndian.PutUint16(e.headerBuf[1:3], uint16(run.offset))

	if _, err := w.Write(e.headerBuf[:3]); err != nil {
		return err
	}
	_, err := w.Write(run.data)
	return err
}

// writeLongRun writes a long run (larger offsets/lengths).
// Format: [1 byte: 0x80 | length_high] [1 byte: length_low] [3 bytes: offset LE] [N bytes: pixel data]
func (e *Encoder) writeLongRun(run changeRun, w io.Writer) error {
	// Length as 15-bit value with high bit set on first byte
	e.headerBuf[0] = 0x80 | byte((run.length>>8)&0x7F)
	e.headerBuf[1] = byte(run.length & 0xFF)
	// Offset as 24-bit little-endian
	e.headerBuf[2] = byte(run.offset & 0xFF)
	e.headerBuf[3] = byte((run.offset >> 8) & 0xFF)
	e.headerBuf[4] = byte((run.offset >> 16) & 0xFF)

	if _, err := w.Write(e.headerBuf[:5]); err != nil {
		return err
	}
	_, err := w.Write(run.data)
	return err
}

// Reset clears the encoder state, forcing the next frame to be a full frame.
func (e *Encoder) Reset() {
	e.hasPrev = false
}
