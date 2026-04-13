// Package delta provides delta compression for BGRA framebuffer streaming.
// It encodes only changed pixels between frames, significantly reducing bandwidth
// for e-ink usage where typically only 1-5% of pixels change between frames.
package delta

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"

	"github.com/cespare/xxhash/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/trace"
)

// zstdEncoderPool reuses zstd encoders to avoid allocation overhead
var zstdEncoderPool = sync.Pool{
	New: func() any {
		enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		return enc
	},
}

// ResetEncoderPool replaces the zstd encoder pool with a fresh one,
// allowing old encoders to be garbage collected.
func ResetEncoderPool() {
	zstdEncoderPool = sync.Pool{
		New: func() any {
			enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
			return enc
		},
	}
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
	maxShortOffset = 0xFFFF // 64KB - 1
	maxShortLength = 127    // 7 bits

	// bytesPerPixel defines the pixel format for delta encoding.
	// Hardcoded to 4 for BGRA32 format (matches remarkable.BytesPerPixelBGRA).
	// All current reMarkable devices use BGRA in streaming mode:
	// - RM2 firmware 3.24+ uses BGRA
	// - RMPP uses BGRA
	// - RM2 legacy firmware uses gray16 but is converted to BGRA server-side
	bytesPerPixel = 4
)

// Encoder holds the state for delta encoding between frames.
type Encoder struct {
	prevFrame     []byte
	prevFrameHash uint64 // XXHash64 of previous frame for fast equality check
	threshold     float64
	hasPrev       bool
	lastChanged   bool // true when previous frame had changes (hybrid hash strategy)
	// Reusable buffers to avoid allocations
	headerBuf     [5]byte     // Max header size (long run: 5 bytes)
	frameHeader   [4]byte     // Frame header buffer
	runsBuf       []changeRun // Reusable runs slice
	compressedBuf []byte      // Reusable buffer for ZSTD compression output
	maskBuf       []byte      // Reusable buffer for block comparison mask
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
func (e *Encoder) EncodeWithSize(current []byte, w io.Writer) (n int, err error) {
	span := trace.BeginSpan("delta_encode")
	defer func() {
		frameType := "delta"
		if !e.hasPrev {
			frameType = "full"
		}
		trace.EndSpan(span, map[string]any{
			"bytes_written": n,
			"frame_type":    frameType,
		})
	}()

	frameSize := len(current)

	// First frame or no previous: send full frame
	if !e.hasPrev || len(e.prevFrame) != frameSize {
		e.prevFrame = make([]byte, frameSize)
		copy(e.prevFrame, current)
		e.prevFrameHash = xxhash.Sum64(current)
		e.hasPrev = true
		debug.Log("Delta: first frame, sending full")
		return e.writeFullFrame(current, w)
	}

	// Compare frames and copy current → prev in a single pass.
	// This merges two memory scans into one, reducing bandwidth pressure.
	runs := e.compareAndCopyFrames(current)

	// Calculate total changed bytes
	changedBytes := 0
	for _, run := range runs {
		changedBytes += len(run.data)
	}

	// No changes - send empty delta frame (copy already skipped via hash early exit)
	if changedBytes == 0 {
		debug.Log("Delta: no changes, sending empty delta")
		return e.writeDeltaFrame(runs, 0, w)
	}

	// Calculate change ratio
	changeRatio := float64(changedBytes) / float64(frameSize)

	// Calculate delta payload size
	deltaSize := e.calculateDeltaSize(runs)

	// If change ratio exceeds threshold OR delta is larger than full frame, send full frame
	// prevFrame was already updated during compareAndCopyFrames
	if changeRatio > e.threshold || deltaSize >= frameSize {
		debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending full", changeRatio*100, len(runs))
		return e.writeFullFrame(current, w)
	}

	// Send delta frame - prevFrame already updated during compareAndCopyFrames
	debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending delta", changeRatio*100, len(runs))
	return e.writeDeltaFrame(runs, deltaSize, w)
}

// compareAndCopyFrames compares current frame with previous, copies current → prev
// during the same pass, and returns change runs.
// This merges two memory-bandwidth-heavy operations (compare + copy) into one,
// reducing total memory traffic by ~25% for changed frames.
//
// Uses a hybrid hash strategy: hash is only computed when the previous frame was
// unchanged (idle mode). During active drawing, hash is skipped entirely (~41% CPU
// savings for changed frames). When drawing stops and compare+copy finds no changes,
// hash is recomputed once to re-enable fast early exit for subsequent idle frames.
//
// Reuses internal buffer to minimize allocations.
// The returned slice references data in 'current', so current must not be modified
// until after the runs are processed.
func (e *Encoder) compareAndCopyFrames(current []byte) []changeRun {
	span := trace.BeginSpan("delta_compare")
	earlyExit := false
	hashSkipped := false
	defer func() {
		changedBytes := 0
		for _, run := range e.runsBuf {
			changedBytes += len(run.data)
		}
		changeRatio := 0.0
		if len(current) > 0 {
			changeRatio = float64(changedBytes) / float64(len(current))
		}
		trace.EndSpan(span, map[string]any{
			"frame_size":    len(current),
			"runs":          len(e.runsBuf),
			"changed_bytes": changedBytes,
			"change_ratio":  changeRatio,
			"early_exit":    earlyExit,
			"hash_skipped":  hashSkipped,
		})
	}()

	// Reuse runs buffer, reset length but keep capacity
	e.runsBuf = e.runsBuf[:0]

	prev := e.prevFrame
	frameLen := len(current)

	// Handle empty buffers - no comparison needed
	if frameLen == 0 || len(prev) == 0 {
		return e.runsBuf
	}

	// Safety check: buffers should have same length (caller's responsibility)
	// but prevent panic if they don't
	if len(prev) != frameLen {
		return e.runsBuf
	}

	// Hybrid hash strategy: only compute hash when likely idle.
	// During active drawing (lastChanged=true), skip the expensive hash
	// and go straight to compare+copy. When frames stop changing,
	// compute hash once to re-enable fast early exit for idle frames.
	if !e.lastChanged {
		currentHash := xxhash.Sum64(current)
		if currentHash == e.prevFrameHash {
			earlyExit = true
			return e.runsBuf
		}
		e.prevFrameHash = currentHash
	} else {
		hashSkipped = true
	}

	// SIMD-accelerated compare+copy in 64-byte blocks.
	// On arm64 this uses NEON vector instructions; on other platforms a scalar fallback.
	// The mask records which blocks contain any changed bytes.
	nblocks := frameLen / blockSize
	remainder := frameLen - nblocks*blockSize

	// Ensure mask buffer is large enough (reuse across calls)
	if cap(e.maskBuf) < nblocks {
		e.maskBuf = make([]byte, nblocks)
	}
	mask := e.maskBuf[:nblocks]

	// Compare and copy all full blocks
	compareAndCopyBlocks(unsafe.Pointer(&prev[0]), unsafe.Pointer(&current[0]), mask, nblocks)

	// Build change runs from block mask.
	// blockSize (128) is a multiple of bytesPerPixel (4), so block boundaries
	// are always pixel-aligned — no alignment fixup needed.
	runStart := -1 // block index of current run start, or -1

	for i, changed := range mask {
		if changed != 0 {
			if runStart == -1 {
				runStart = i
			}
		} else if runStart != -1 {
			startByte := runStart * blockSize
			endByte := i * blockSize
			e.runsBuf = append(e.runsBuf, changeRun{
				offset: startByte,
				length: (endByte - startByte) / bytesPerPixel,
				data:   current[startByte:endByte],
			})
			runStart = -1
		}
	}

	// Handle remainder bytes after last full block (compare and copy)
	remStart := nblocks * blockSize
	remChanged := false
	for i := remStart; i < frameLen; i++ {
		c := current[i]
		p := prev[i]
		prev[i] = c
		if c != p {
			remChanged = true
		}
	}

	// Finalize any open run
	if runStart != -1 {
		startByte := runStart * blockSize
		endByte := nblocks * blockSize
		if remChanged {
			endByte = frameLen
		}
		alignedEnd := ((endByte + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
		alignedEnd = min(alignedEnd, frameLen)
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: startByte,
			length: (alignedEnd - startByte) / bytesPerPixel,
			data:   current[startByte:alignedEnd],
		})
	} else if remChanged && remainder > 0 {
		alignedStart := (remStart / bytesPerPixel) * bytesPerPixel
		alignedEnd := ((frameLen + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
		alignedEnd = min(alignedEnd, frameLen)
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: alignedStart,
			length: (alignedEnd - alignedStart) / bytesPerPixel,
			data:   current[alignedStart:alignedEnd],
		})
	}

	// Update hybrid hash state
	if len(e.runsBuf) == 0 {
		// No changes — transition to idle mode.
		// Compute hash so the next identical frame can early-exit.
		if e.lastChanged {
			e.prevFrameHash = xxhash.Sum64(current)
		}
		e.lastChanged = false
	} else {
		e.lastChanged = true
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

// compareFrames compares current frame with previous and returns change runs.
// Unlike compareAndCopyFrames, it does NOT copy current → prev.
// This is used by tests and benchmarks that manage prevFrame directly.
func (e *Encoder) compareFrames(current []byte) []changeRun {
	e.runsBuf = e.runsBuf[:0]

	prev := e.prevFrame
	frameLen := len(current)

	if frameLen == 0 || len(prev) == 0 || len(prev) != frameLen {
		return e.runsBuf
	}

	currentHash := xxhash.Sum64(current)
	if e.hasPrev && currentHash == e.prevFrameHash {
		return e.runsBuf
	}
	e.prevFrameHash = currentHash

	numQwords := frameLen / 8
	var runStart int = -1
	var lastDiffEnd int
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
		} else if runStart != -1 {
			alignedStart := (runStart / bytesPerPixel) * bytesPerPixel
			alignedEnd := ((lastDiffEnd + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
			alignedEnd = min(alignedEnd, frameLen)
			e.runsBuf = append(e.runsBuf, changeRun{
				offset: alignedStart,
				length: (alignedEnd - alignedStart) / bytesPerPixel,
				data:   current[alignedStart:alignedEnd],
			})
			runStart = -1
		}
	}

	for i := numQwords * 8; i < frameLen; i++ {
		if current[i] != prev[i] {
			if runStart == -1 {
				runStart = i
			}
			lastDiffEnd = i + 1
		}
	}

	if runStart != -1 {
		alignedStart := (runStart / bytesPerPixel) * bytesPerPixel
		alignedEnd := ((lastDiffEnd + bytesPerPixel - 1) / bytesPerPixel) * bytesPerPixel
		alignedEnd = min(alignedEnd, frameLen)
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: alignedStart,
			length: (alignedEnd - alignedStart) / bytesPerPixel,
			data:   current[alignedStart:alignedEnd],
		})
	}

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
	span := trace.BeginSpan("zstd_compress")
	inputSize := len(data)

	// Compress data with zstd using pooled encoder
	enc := zstdEncoderPool.Get().(*zstd.Encoder)
	defer func() {
		enc.Reset(nil) // Clear internal buffers before returning to pool
		zstdEncoderPool.Put(enc)
	}()

	// Reuse compressed buffer (reset length, keep capacity)
	e.compressedBuf = e.compressedBuf[:0]
	e.compressedBuf = enc.EncodeAll(data, e.compressedBuf)

	compressionRatio := 0.0
	if inputSize > 0 {
		compressionRatio = float64(len(e.compressedBuf)) / float64(inputSize)
	}

	trace.EndSpan(span, map[string]any{
		"input_size":        inputSize,
		"output_size":       len(e.compressedBuf),
		"compression_ratio": compressionRatio,
	})

	// Write header with zstd compressed type
	e.frameHeader[0] = FrameTypeFullZstd
	// Payload length in 24-bit little-endian
	payloadLen := len(e.compressedBuf)
	e.frameHeader[1] = byte(payloadLen & 0xFF)
	e.frameHeader[2] = byte((payloadLen >> 8) & 0xFF)
	e.frameHeader[3] = byte((payloadLen >> 16) & 0xFF)

	if _, err := w.Write(e.frameHeader[:]); err != nil {
		return 0, err
	}
	n, err := w.Write(e.compressedBuf)
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
	e.lastChanged = false
}

// ReleaseMemory releases large buffers held by the encoder to reduce memory usage.
// After calling this, the encoder remains usable but will reallocate buffers as needed.
func (e *Encoder) ReleaseMemory() {
	e.hasPrev = false
	e.lastChanged = false
	e.prevFrame = nil
	e.runsBuf = nil
	e.compressedBuf = nil
	e.maskBuf = nil
}
