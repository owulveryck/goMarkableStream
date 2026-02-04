// Package delta provides delta compression for BGRA framebuffer streaming.
// It encodes only changed pixels between frames, significantly reducing bandwidth
// for e-ink usage where typically only 1-5% of pixels change between frames.
package delta

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"unsafe"
)

const (
	// Frame type constants for wire protocol
	FrameTypeFull           = 0x00 // Deprecated: uncompressed full frame
	FrameTypeDelta          = 0x01
	FrameTypeFullCompressed = 0x02 // Gzip-compressed full frame

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
type changeRun struct {
	offset int // byte offset from previous run end (or frame start)
	length int // number of changed pixels
	data   []byte
}

// Encode writes the current frame to w, using delta encoding if beneficial.
// Returns nil on success, or an error if writing fails.
func (e *Encoder) Encode(current []byte, w io.Writer) error {
	frameSize := len(current)

	// First frame or no previous: send full frame
	if !e.hasPrev || len(e.prevFrame) != frameSize {
		e.prevFrame = make([]byte, frameSize)
		copy(e.prevFrame, current)
		e.hasPrev = true
		return e.writeFullFrame(current, w)
	}

	// Compare frames and build change runs
	runs := e.compareFrames(current)

	// Calculate total changed bytes
	changedBytes := 0
	for _, run := range runs {
		changedBytes += len(run.data)
	}

	// Calculate change ratio
	changeRatio := float64(changedBytes) / float64(frameSize)

	// Calculate delta payload size
	deltaSize := e.calculateDeltaSize(runs)

	// If change ratio exceeds threshold OR delta is larger than full frame, send full frame
	if changeRatio > e.threshold || deltaSize >= frameSize {
		copy(e.prevFrame, current)
		return e.writeFullFrame(current, w)
	}

	// Send delta frame
	copy(e.prevFrame, current)
	return e.writeDeltaFrame(runs, deltaSize, w)
}

// compareFrames compares current frame with previous and returns change runs.
// Uses uint64 comparisons for speed.
func (e *Encoder) compareFrames(current []byte) []changeRun {
	var runs []changeRun
	prev := e.prevFrame
	frameLen := len(current)

	// Compare 8 bytes at a time using unsafe pointer casting
	numQwords := frameLen / 8

	var runStart int = -1
	var lastDiffEnd int = 0

	// Get pointers for fast comparison
	currPtr := unsafe.Pointer(&current[0])
	prevPtr := unsafe.Pointer(&prev[0])

	for i := 0; i < numQwords; i++ {
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
				if alignedEnd > frameLen {
					alignedEnd = frameLen
				}

				run := changeRun{
					offset: alignedStart,
					length: (alignedEnd - alignedStart) / bytesPerPixel,
					data:   make([]byte, alignedEnd-alignedStart),
				}
				copy(run.data, current[alignedStart:alignedEnd])
				runs = append(runs, run)
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
		if alignedEnd > frameLen {
			alignedEnd = frameLen
		}

		run := changeRun{
			offset: alignedStart,
			length: (alignedEnd - alignedStart) / bytesPerPixel,
			data:   make([]byte, alignedEnd-alignedStart),
		}
		copy(run.data, current[alignedStart:alignedEnd])
		runs = append(runs, run)
	}

	// Convert absolute offsets to relative offsets
	lastEnd := 0
	for i := range runs {
		absOffset := runs[i].offset
		runs[i].offset = absOffset - lastEnd
		lastEnd = absOffset + len(runs[i].data)
	}

	return runs
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

// writeFullFrame writes a gzip-compressed full frame with header.
func (e *Encoder) writeFullFrame(data []byte, w io.Writer) error {
	// Compress data with gzip (BestSpeed for minimal CPU overhead)
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	gz.Write(data)
	gz.Close()
	compressed := buf.Bytes()

	// Write header with compressed type
	header := make([]byte, 4)
	header[0] = FrameTypeFullCompressed
	// Payload length in 24-bit little-endian
	payloadLen := len(compressed)
	header[1] = byte(payloadLen & 0xFF)
	header[2] = byte((payloadLen >> 8) & 0xFF)
	header[3] = byte((payloadLen >> 16) & 0xFF)

	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(compressed)
	return err
}

// writeDeltaFrame writes a delta frame with header and change runs.
func (e *Encoder) writeDeltaFrame(runs []changeRun, payloadSize int, w io.Writer) error {
	header := make([]byte, 4)
	header[0] = FrameTypeDelta
	// Payload length in 24-bit little-endian
	header[1] = byte(payloadSize & 0xFF)
	header[2] = byte((payloadSize >> 8) & 0xFF)
	header[3] = byte((payloadSize >> 16) & 0xFF)

	if _, err := w.Write(header); err != nil {
		return err
	}

	// Write each change run
	for _, run := range runs {
		if run.offset <= maxShortOffset && run.length <= maxShortLength {
			// Short run format
			if err := e.writeShortRun(run, w); err != nil {
				return err
			}
		} else {
			// Long run format
			if err := e.writeLongRun(run, w); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeShortRun writes a short run (offset < 64KB, length <= 127 pixels).
// Format: [1 byte: length] [2 bytes: relative offset LE] [N bytes: pixel data]
func (e *Encoder) writeShortRun(run changeRun, w io.Writer) error {
	buf := make([]byte, 3)
	buf[0] = byte(run.length)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(run.offset))

	if _, err := w.Write(buf); err != nil {
		return err
	}
	_, err := w.Write(run.data)
	return err
}

// writeLongRun writes a long run (larger offsets/lengths).
// Format: [1 byte: 0x80 | length_high] [1 byte: length_low] [3 bytes: offset LE] [N bytes: pixel data]
func (e *Encoder) writeLongRun(run changeRun, w io.Writer) error {
	buf := make([]byte, 5)
	// Length as 15-bit value with high bit set on first byte
	buf[0] = 0x80 | byte((run.length>>8)&0x7F)
	buf[1] = byte(run.length & 0xFF)
	// Offset as 24-bit little-endian
	buf[2] = byte(run.offset & 0xFF)
	buf[3] = byte((run.offset >> 8) & 0xFF)
	buf[4] = byte((run.offset >> 16) & 0xFF)

	if _, err := w.Write(buf); err != nil {
		return err
	}
	_, err := w.Write(run.data)
	return err
}

// Reset clears the encoder state, forcing the next frame to be a full frame.
func (e *Encoder) Reset() {
	e.hasPrev = false
}
