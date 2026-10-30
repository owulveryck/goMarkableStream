// Package delta provides delta compression for BGRA, RGB565, and Color8 framebuffer streaming.
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

// newFullFrameEncoder builds an encoder for whole-framebuffer compression.
//
// Concurrency is pinned to 1 on purpose: by default zstd fans out to GOMAXPROCS
// goroutines and allocates per-goroutine state, which costs ~20MB of retained
// buffers per encoder. On a dual-core device that fan-out also steals the core
// the framebuffer reader needs, so it buys nothing.
func newFullFrameEncoder() *zstd.Encoder {
	enc, _ := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderConcurrency(1),
	)
	return enc
}

// newDeltaEncoder builds an encoder for delta run streams. Delta payloads are
// small and highly local (long repeats of untouched background), so a bounded
// window keeps nearly all of the compression ratio while shrinking the retained
// encoder state by roughly 4x compared to the full-frame encoder.
func newDeltaEncoder() *zstd.Encoder {
	enc, _ := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderConcurrency(1),
		zstd.WithWindowSize(deltaWindowSize),
	)
	return enc
}

// zstdEncoderPool reuses zstd encoders to avoid allocation overhead.
// Full frames and deltas use separate pools so that an encoder sized for a 10MB
// framebuffer is never recycled for a few-KB delta.
var (
	zstdEncoderPool      = sync.Pool{New: func() any { return newFullFrameEncoder() }}
	zstdDeltaEncoderPool = sync.Pool{New: func() any { return newDeltaEncoder() }}
)

// ResetEncoderPool replaces the zstd encoder pools with fresh ones,
// allowing old encoders to be garbage collected.
func ResetEncoderPool() {
	zstdEncoderPool = sync.Pool{New: func() any { return newFullFrameEncoder() }}
	zstdDeltaEncoderPool = sync.Pool{New: func() any { return newDeltaEncoder() }}
}

const (
	// Frame type constants for wire protocol
	FrameTypeFull           = 0x00 // Deprecated: uncompressed full frame
	FrameTypeDelta          = 0x01
	FrameTypeFullCompressed = 0x02 // Gzip-compressed full frame (legacy)
	FrameTypeFullZstd       = 0x03 // Zstd-compressed full frame
	FrameTypeDeltaZstd      = 0x04 // Zstd-compressed delta frame

	// DefaultThreshold is the default change ratio above which a full frame is sent
	DefaultThreshold = 0.30

	// minDeltaCompressSize is the delta payload size below which compression is
	// skipped: zstd framing overhead plus CPU cost outweighs the saving, and such
	// frames are negligible on the wire anyway.
	minDeltaCompressSize = 1024

	// Falling back to a full frame means pushing the whole framebuffer through
	// zstd (~10MB, i.e. ~10x the CPU of a typical delta on a Cortex-A9), so it is
	// only worth it when the saving is substantial in both relative and absolute
	// terms. A delta must be at least fullFrameAdvantage times bigger than the
	// last full frame *and* above minFullSwitchSize on the wire before we switch.
	fullFrameAdvantage = 2
	minFullSwitchSize  = 32 * 1024

	// deltaWindowSize bounds the zstd match window used for delta payloads.
	deltaWindowSize = 256 * 1024

	// Maximum values for run header encoding
	maxShortOffset = 0xFFFF // 64KB - 1
	maxShortLength = 127    // 7 bits
	maxLongLength  = 0x7FFF // 15 bits (32,767 pixels max per long run header)

	// bytesPerPixel is the default pixel stride (BGRA32).
	bytesPerPixel = 4
)

// Encoder holds the state for delta encoding between frames.
type Encoder struct {
	prevFrame     []byte
	prevFrameHash uint64 // XXHash64 of previous frame for fast equality check
	threshold     float64
	hasPrev       bool
	lastChanged   bool  // true when previous frame had changes (hybrid hash strategy)
	inBPP         uint8 // input framebuffer bytes per pixel (0/4 = BGRA32, 2 = RGB565)
	outBPP        uint8 // wire output bytes per pixel (0/4 = BGRA32, 2 = RGB565, 1 = Color8)
	convertColor8 bool  // true: convert changed RGB565 spans to 1-byte Color8 on output
	// Reusable buffers to avoid allocations
	headerBuf     [5]byte     // Max header size (long run: 5 bytes)
	frameHeader   [4]byte     // Frame header buffer
	runsBuf       []changeRun // Reusable runs slice
	compressedBuf []byte      // Reusable buffer for ZSTD compression output
	maskBuf       []byte      // Reusable buffer for block comparison mask
	writeBuf      []byte      // Reusable buffer for coalesced delta frame writes
	deltaZstdBuf  []byte      // Reusable buffer for ZSTD-compressed delta frames
	colorBuf      []byte      // Reusable buffer for full-frame RGB565→Color8 conversion
	prevChecksum  [16]byte    // XOR-fold checksum of previous frame (ARM32 idle detection)

	// lastFullSize is the total wire size (header included) of the most recent
	// full frame sent. It is the reference used to decide whether a delta is
	// actually cheaper than resending the whole screen, without having to
	// compress 10MB on every frame just to answer the question.
	lastFullSize int
}

// NewEncoder creates a new 4-bytes-per-pixel (BGRA32) delta encoder with the given threshold.
// Threshold is the change ratio (0.0-1.0) above which a full frame is sent.
func NewEncoder(threshold float64) *Encoder {
	if threshold <= 0 || threshold > 1.0 {
		threshold = DefaultThreshold
	}
	return &Encoder{
		threshold: threshold,
		inBPP:     bytesPerPixel,
		outBPP:    bytesPerPixel,
	}
}

// NewColor8Encoder creates a delta encoder optimized for the reMarkable 1:
// it compares raw 2-byte-per-pixel RGB565 frames in 128-byte NEON blocks first
// ("compare-before-convert") and converts ONLY the changed runs into 1-byte
// Color8 palette values (transparent paper + 4 pen colors + 21 grayscale levels)
// before Zstd compression.
func NewColor8Encoder(threshold float64) *Encoder {
	if threshold <= 0 || threshold > 1.0 {
		threshold = DefaultThreshold
	}
	return &Encoder{
		threshold:     threshold,
		inBPP:         2,
		outBPP:        1,
		convertColor8: true,
	}
}

// NewRGB565Encoder creates a lossless 2-byte-per-pixel RGB565 delta encoder for
// the reMarkable 1 (zero pixel format conversion on the server, Zstd-compressed
// RGB565 full and delta frames).
func NewRGB565Encoder(threshold float64) *Encoder {
	if threshold <= 0 || threshold > 1.0 {
		threshold = DefaultThreshold
	}
	return &Encoder{
		threshold: threshold,
		inBPP:     2,
		outBPP:    2,
	}
}

func (e *Encoder) inputBPP() int {
	if e.inBPP == 0 {
		return bytesPerPixel
	}
	return int(e.inBPP)
}

func (e *Encoder) outputBPP() int {
	if e.outBPP == 0 {
		return bytesPerPixel
	}
	return int(e.outBPP)
}

// changeRun represents a contiguous run of changed pixels.
// Uses slice reference to avoid copying data.
type changeRun struct {
	offset int    // wire-format byte offset from previous run end (or frame start)
	length int    // number of changed pixels (<= maxLongLength when emitted by appendRun)
	data   []byte // slice into cached prevFrame (or test input)
}

// appendRun appends a changed span to e.runsBuf, automatically splitting any
// contiguous span exceeding the 15-bit long-run header limit (32,767 pixels)
// into contiguous chunks with offset=0 on continuation chunks.
func (e *Encoder) appendRun(inOffsetBytes int, data []byte, inBPP, outBPP int) {
	wireOffset := (inOffsetBytes / inBPP) * outBPP
	totalPixels := len(data) / inBPP
	maxChunkBytes := maxLongLength * inBPP
	for totalPixels > maxLongLength {
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: wireOffset,
			length: maxLongLength,
			data:   data[:maxChunkBytes],
		})
		data = data[maxChunkBytes:]
		totalPixels -= maxLongLength
		wireOffset = 0
	}
	if totalPixels > 0 {
		e.runsBuf = append(e.runsBuf, changeRun{
			offset: wireOffset,
			length: totalPixels,
			data:   data,
		})
	}
}

// Encode writes the current frame to w, using delta encoding if beneficial.
// Returns nil on success, or an error if writing fails.
func (e *Encoder) Encode(current []byte, w io.Writer) error {
	_, err := e.EncodeWithSize(current, w)
	return err
}

// EncodeWithSize writes the current frame to w, using delta encoding if beneficial.
// When the frame is unchanged, emits a 4-byte empty delta keepalive.
// Returns the number of bytes written and nil on success, or 0 and an error if writing fails.
func (e *Encoder) EncodeWithSize(current []byte, w io.Writer) (n int, err error) {
	n, _, err = e.encodeInternal(current, w, true)
	return n, err
}

// EncodeIfChanged encodes and writes the frame only when it differs from the
// previous frame (or on the initial full frame). When unchanged, writes 0 bytes
// and returns (0, false, nil) — used by the reMarkable 1 low-duty idle poller
// to detect hardware button page turns with zero idle network traffic.
func (e *Encoder) EncodeIfChanged(current []byte, w io.Writer) (n int, changed bool, err error) {
	return e.encodeInternal(current, w, false)
}

func (e *Encoder) encodeInternal(current []byte, w io.Writer, sendEmpty bool) (n int, changed bool, err error) {
	span := trace.BeginSpan("delta_encode")
	frameType := "delta"
	defer func() {
		trace.EndSpan(span, map[string]any{
			"bytes_written": n,
			"frame_type":    frameType,
		})
	}()

	frameSize := len(current)

	// First frame or size change: snapshot into cached prevFrame via burst-friendly
	// CopyFrame (critical when current is Write-Combine mmap'd /dev/fb0 on RM1)
	// and send full frame from prevFrame.
	if !e.hasPrev || len(e.prevFrame) != frameSize {
		frameType = "full"
		e.prevFrame = make([]byte, frameSize)
		CopyFrame(e.prevFrame, current)
		if useHashEarlyExit {
			e.prevFrameHash = xxhash.Sum64(e.prevFrame)
		} else if nblocks := frameSize / blockSize; nblocks > 0 {
			checksumChanged(unsafe.Pointer(&e.prevFrame[0]), nblocks, unsafe.Pointer(&e.prevChecksum[0]))
		}
		e.hasPrev = true
		e.lastChanged = false
		debug.Log("Delta: first frame, sending full")
		n, err = e.writeFullFrame(e.prevFrame, w)
		return n, true, err
	}

	// Compare frames and copy changed blocks current → prevFrame in a single pass.
	runs := e.compareAndCopyFrames(current)

	// Calculate total changed input bytes
	changedBytes := 0
	for _, run := range runs {
		changedBytes += len(run.data)
	}

	// No changes
	if changedBytes == 0 {
		if !sendEmpty {
			frameType = "idle_skip"
			return 0, false, nil
		}
		debug.Log("Delta: no changes, sending empty delta")
		n, err = e.writeDeltaFrame(runs, 0, w)
		return n, false, err
	}

	// Calculate change ratio
	changeRatio := float64(changedBytes) / float64(frameSize)

	// Cheap guard before doing any work: a delta covering more than `threshold`
	// of the screen is expensive to assemble and compress, and is very unlikely
	// to beat a compressed full frame.
	// prevFrame was already updated during compareAndCopyFrames.
	if changeRatio > e.threshold {
		frameType = "full"
		debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending full", changeRatio*100, len(runs))
		n, err = e.writeFullFrame(e.prevFrame, w)
		return n, true, err
	}

	// Assemble the delta frame (converting changed RGB565 spans to Color8 inline
	// when convertColor8 is enabled), then Zstd-compress the run stream.
	deltaSize := e.calculateDeltaSize(runs)
	frame := e.assembleDeltaFrame(runs, deltaSize)
	if compressed := e.compressDeltaFrame(frame[4:]); compressed != nil {
		frame = compressed
	}

	// Compare actual compressed delta wire size against resending a full frame.
	if e.lastFullSize > 0 && len(frame) > minFullSwitchSize && len(frame) >= fullFrameAdvantage*e.lastFullSize {
		frameType = "full"
		debug.Log("Delta: changeRatio=%.2f%%, delta=%d >= full=%d, sending full",
			changeRatio*100, len(frame), e.lastFullSize)
		n, err = e.writeFullFrame(e.prevFrame, w)
		return n, true, err
	}

	debug.Log("Delta: changeRatio=%.2f%%, runs=%d, sending delta (%d bytes)",
		changeRatio*100, len(runs), len(frame))
	n, err = w.Write(frame)
	return n, true, err
}

// compareAndCopyFrames compares current frame with previous, copies current → prev
// during the same pass, and returns change runs referencing the cached prevFrame.
// This merges two memory-bandwidth-heavy operations (compare + copy) into one,
// reducing total memory traffic by ~25% for changed frames, and ensures current
// (which may be uncached mmap'd /dev/fb0) is read strictly once via NEON bursts.
//
// Uses a hybrid hash strategy: hash/checksum is only computed when the previous
// frame was unchanged (idle mode). During active drawing, it is skipped entirely.
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

	if frameLen == 0 || len(prev) == 0 || len(prev) != frameLen {
		return e.runsBuf
	}

	inBPP := e.inputBPP()
	outBPP := e.outputBPP()

	nblocks := frameLen / blockSize
	remainder := frameLen - nblocks*blockSize

	if !e.lastChanged {
		if useHashEarlyExit {
			currentHash := xxhash.Sum64(current)
			if currentHash == e.prevFrameHash {
				earlyExit = true
				return e.runsBuf
			}
			e.prevFrameHash = currentHash
		} else if nblocks > 0 {
			// Single-buffer NEON 128-bit XOR-fold checksum early exit (ARM32).
			// Reads only current (5.27MB on RM1, 10.54MB on RM2) via 32-byte
			// VLD1.8 bursts instead of reading both buffers.
			if !checksumChanged(unsafe.Pointer(&current[0]), nblocks, unsafe.Pointer(&e.prevChecksum[0])) {
				// Also verify any sub-128B tail bytes
				tailChanged := false
				for i := nblocks * blockSize; i < frameLen; i++ {
					if current[i] != prev[i] {
						tailChanged = true
						break
					}
				}
				if !tailChanged {
					earlyExit = true
					return e.runsBuf
				}
			}
		}
	} else {
		hashSkipped = true
	}

	// Ensure mask buffer is large enough (reuse across calls)
	if cap(e.maskBuf) < nblocks {
		e.maskBuf = make([]byte, nblocks)
	}
	mask := e.maskBuf[:nblocks]

	// Compare and copy all full 128-byte blocks (prev updated in-place for changed halves)
	if nblocks > 0 {
		compareAndCopyBlocks(unsafe.Pointer(&prev[0]), unsafe.Pointer(&current[0]), mask, nblocks)
	}

	// Build change runs from block mask, computing relative wire offsets inline.
	// blockSize (128) is a multiple of all supported inBPP values (4 and 2).
	runStart := -1
	lastEnd := 0

	nwords := nblocks / 8
	var maskPtr unsafe.Pointer
	if nblocks > 0 {
		maskPtr = unsafe.Pointer(&mask[0])
	}

	for w := range nwords {
		word := *(*uint64)(unsafe.Add(maskPtr, w*8))
		if word == 0 {
			if runStart != -1 {
				endBlock := w * 8
				startByte := runStart * blockSize
				endByte := endBlock * blockSize
				e.appendRun(startByte-lastEnd, prev[startByte:endByte], inBPP, outBPP)
				lastEnd = endByte
				runStart = -1
			}
		} else {
			base := w * 8
			for j := range 8 {
				i := base + j
				if mask[i] != 0 {
					if runStart == -1 {
						runStart = i
					}
				} else if runStart != -1 {
					startByte := runStart * blockSize
					endByte := i * blockSize
					e.appendRun(startByte-lastEnd, prev[startByte:endByte], inBPP, outBPP)
					lastEnd = endByte
					runStart = -1
				}
			}
		}
	}

	// Handle remaining mask bytes (nblocks % 8)
	for i := nwords * 8; i < nblocks; i++ {
		if mask[i] != 0 {
			if runStart == -1 {
				runStart = i
			}
		} else if runStart != -1 {
			startByte := runStart * blockSize
			endByte := i * blockSize
			e.appendRun(startByte-lastEnd, prev[startByte:endByte], inBPP, outBPP)
			lastEnd = endByte
			runStart = -1
		}
	}

	// Handle remainder bytes after last full block (compare and copy into prev)
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
		alignedEnd := ((endByte + inBPP - 1) / inBPP) * inBPP
		alignedEnd = min(alignedEnd, frameLen)
		e.appendRun(startByte-lastEnd, prev[startByte:alignedEnd], inBPP, outBPP)
	} else if remChanged && remainder > 0 {
		alignedStart := (remStart / inBPP) * inBPP
		alignedEnd := ((frameLen + inBPP - 1) / inBPP) * inBPP
		alignedEnd = min(alignedEnd, frameLen)
		e.appendRun(alignedStart-lastEnd, prev[alignedStart:alignedEnd], inBPP, outBPP)
	}

	// Update hybrid hash/checksum state using cached prev (never re-reading current)
	if len(e.runsBuf) == 0 {
		if e.lastChanged {
			if useHashEarlyExit {
				e.prevFrameHash = xxhash.Sum64(prev)
			} else if nblocks > 0 {
				checksumChanged(unsafe.Pointer(&prev[0]), nblocks, unsafe.Pointer(&e.prevChecksum[0]))
			}
		}
		e.lastChanged = false
	} else {
		e.lastChanged = true
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

	inBPP := e.inputBPP()
	outBPP := e.outputBPP()

	numQwords := frameLen / 8
	var runStart int = -1
	var lastDiffEnd int
	var lastEnd int
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
			alignedStart := (runStart / inBPP) * inBPP
			alignedEnd := ((lastDiffEnd + inBPP - 1) / inBPP) * inBPP
			alignedEnd = min(alignedEnd, frameLen)
			e.appendRun(alignedStart-lastEnd, current[alignedStart:alignedEnd], inBPP, outBPP)
			lastEnd = alignedEnd
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
		alignedStart := (runStart / inBPP) * inBPP
		alignedEnd := ((lastDiffEnd + inBPP - 1) / inBPP) * inBPP
		alignedEnd = min(alignedEnd, frameLen)
		e.appendRun(alignedStart-lastEnd, current[alignedStart:alignedEnd], inBPP, outBPP)
	}

	return e.runsBuf
}

// calculateDeltaSize calculates the wire byte size of the delta payload.
func (e *Encoder) calculateDeltaSize(runs []changeRun) int {
	size := 0
	for _, run := range runs {
		runBytes := len(run.data)
		if e.convertColor8 {
			runBytes = run.length
		}
		if run.offset <= maxShortOffset && run.length <= maxShortLength {
			// Short run: 1 byte length + 2 bytes offset + pixel data
			size += 1 + 2 + runBytes
		} else {
			// Long run: 2 bytes length + 3 bytes offset + pixel data
			size += 2 + 3 + runBytes
		}
	}
	return size
}

// writeFullFrame writes a zstd-compressed full frame with header.
// When convertColor8 is enabled, converts RGB565 → Color8 into reusable colorBuf
// before Zstd compression (halving Zstd input size and producing ~3-5x smaller output).
func (e *Encoder) writeFullFrame(data []byte, w io.Writer) (int, error) {
	if e.convertColor8 {
		npixels := len(data) / 2
		if cap(e.colorBuf) < npixels {
			e.colorBuf = make([]byte, npixels)
		}
		converted := e.colorBuf[:npixels]
		ConvertRGB565ToColor8(converted, data)
		data = converted
	}

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
	payloadLen := len(e.compressedBuf)
	e.frameHeader[1] = byte(payloadLen & 0xFF)
	e.frameHeader[2] = byte((payloadLen >> 8) & 0xFF)
	e.frameHeader[3] = byte((payloadLen >> 16) & 0xFF)

	// Record the wire cost of resending the whole screen
	e.lastFullSize = 4 + payloadLen

	if _, err := w.Write(e.frameHeader[:]); err != nil {
		return 0, err
	}
	n, err := w.Write(e.compressedBuf)
	if err != nil {
		return 4, err
	}
	return 4 + n, nil
}

// writeDeltaFrame assembles an uncompressed delta frame and writes it with a
// single w.Write() call. Used for the empty-delta keepalive.
func (e *Encoder) writeDeltaFrame(runs []changeRun, payloadSize int, w io.Writer) (int, error) {
	return w.Write(e.assembleDeltaFrame(runs, payloadSize))
}

// compressDeltaFrame zstd-compresses a delta run stream and returns a complete
// frame (4-byte FrameTypeDeltaZstd header + compressed payload), or nil when
// compression is not worth it and the caller should send the payload as-is.
func (e *Encoder) compressDeltaFrame(payload []byte) []byte {
	if len(payload) < minDeltaCompressSize {
		return nil
	}

	span := trace.BeginSpan("zstd_compress_delta")

	enc := zstdDeltaEncoderPool.Get().(*zstd.Encoder)
	defer func() {
		enc.Reset(nil) // Clear internal buffers before returning to pool
		zstdDeltaEncoderPool.Put(enc)
	}()

	// Reserve the 4 header bytes up front so EncodeAll appends straight after
	// them: the frame is emitted with a single Write and no extra copy.
	e.deltaZstdBuf = append(e.deltaZstdBuf[:0], 0, 0, 0, 0)
	e.deltaZstdBuf = enc.EncodeAll(payload, e.deltaZstdBuf)

	compressedLen := len(e.deltaZstdBuf) - 4
	trace.EndSpan(span, map[string]any{
		"input_size":  len(payload),
		"output_size": compressedLen,
	})

	// Incompressible payload: sending it raw avoids the client-side inflate.
	if compressedLen >= len(payload) {
		return nil
	}

	e.deltaZstdBuf[0] = FrameTypeDeltaZstd
	e.deltaZstdBuf[1] = byte(compressedLen & 0xFF)
	e.deltaZstdBuf[2] = byte((compressedLen >> 8) & 0xFF)
	e.deltaZstdBuf[3] = byte((compressedLen >> 16) & 0xFF)
	return e.deltaZstdBuf
}

// assembleDeltaFrame builds the entire delta frame (4-byte header + all runs)
// into a single reusable buffer. When convertColor8 is enabled, changed RGB565
// spans are converted directly into buf via NEON/LUT with zero extra copies.
func (e *Encoder) assembleDeltaFrame(runs []changeRun, payloadSize int) []byte {
	totalSize := 4 + payloadSize

	// Reuse write buffer, grow if needed
	if cap(e.writeBuf) < totalSize {
		e.writeBuf = make([]byte, totalSize)
	}
	buf := e.writeBuf[:totalSize]

	// Frame header
	buf[0] = FrameTypeDelta
	buf[1] = byte(payloadSize & 0xFF)
	buf[2] = byte((payloadSize >> 8) & 0xFF)
	buf[3] = byte((payloadSize >> 16) & 0xFF)

	pos := 4
	for _, run := range runs {
		if run.offset <= maxShortOffset && run.length <= maxShortLength {
			// Short run: 1 byte length + 2 bytes offset LE + pixel data
			buf[pos] = byte(run.length)
			binary.LittleEndian.PutUint16(buf[pos+1:pos+3], uint16(run.offset))
			pos += 3
		} else {
			// Long run: 2 bytes length (0x80|high, low) + 3 bytes offset LE + pixel data
			buf[pos] = 0x80 | byte((run.length>>8)&0x7F)
			buf[pos+1] = byte(run.length & 0xFF)
			buf[pos+2] = byte(run.offset & 0xFF)
			buf[pos+3] = byte((run.offset >> 8) & 0xFF)
			buf[pos+4] = byte((run.offset >> 16) & 0xFF)
			pos += 5
		}
		if e.convertColor8 {
			ConvertRGB565ToColor8(buf[pos:pos+run.length], run.data)
			pos += run.length
		} else {
			copy(buf[pos:], run.data)
			pos += len(run.data)
		}
	}

	return buf[:pos]
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
	e.writeBuf = nil
	e.deltaZstdBuf = nil
	e.colorBuf = nil
	e.lastFullSize = 0
}
