package stream

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

// AsyncFrameReader reads the framebuffer continuously in a background goroutine,
// using triple buffering to overlap I/O with delta encoding.
//
// Three buffers rotate between three roles:
//   - writing: background goroutine is filling this via ReadAt (no lock held)
//   - ready:   latest complete frame, waiting to be consumed
//   - reading: handler is encoding this (stable, safe from overwrites)
//
// This allows the Cortex-A9's second core to read the next frame while
// the first core encodes the current one.
type AsyncFrameReader struct {
	file        io.ReaderAt
	pointerAddr int64

	mu      sync.Mutex
	writing []byte // owned by background goroutine during ReadAt
	ready   []byte // latest complete frame
	reading []byte // owned by handler during encode
	hasNew  bool

	paused int32         // atomic: 1 = paused, 0 = active
	wake   chan struct{} // signal to resume from paused state
}

// NewAsyncFrameReader creates a reader with three pre-allocated frame buffers.
func NewAsyncFrameReader(file io.ReaderAt, pointerAddr int64, frameSize int) *AsyncFrameReader {
	return &AsyncFrameReader{
		file:        file,
		pointerAddr: pointerAddr,
		writing:     make([]byte, frameSize),
		ready:       make([]byte, frameSize),
		reading:     make([]byte, frameSize),
		paused:      1, // start paused — resume when writing begins
		wake:        make(chan struct{}, 1),
	}
}

// Pause tells the reader to stop reading the framebuffer.
// The background goroutine blocks until Resume is called.
func (r *AsyncFrameReader) Pause() {
	atomic.StoreInt32(&r.paused, 1)
}

// Resume wakes the reader if it was paused.
func (r *AsyncFrameReader) Resume() {
	if atomic.CompareAndSwapInt32(&r.paused, 1, 0) {
		select {
		case r.wake <- struct{}{}:
		default:
		}
	}
}

// Run reads frames continuously until ctx is cancelled.
// Should be called in a goroutine: go reader.Run(ctx)
func (r *AsyncFrameReader) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// When paused, block until resumed or cancelled.
		if atomic.LoadInt32(&r.paused) == 1 {
			select {
			case <-ctx.Done():
				return
			case <-r.wake:
			}
			continue
		}

		// ReadAt into writing buffer — no lock held, we own this buffer.
		r.file.ReadAt(r.writing, r.pointerAddr)

		// Swap writing and ready under lock (O(1) pointer swap).
		r.mu.Lock()
		r.writing, r.ready = r.ready, r.writing
		r.hasNew = true
		r.mu.Unlock()
	}
}

// Latest returns the latest complete frame, or nil if no new frame
// is available since the last call. The returned slice is stable
// until the next call to Latest.
func (r *AsyncFrameReader) Latest() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasNew {
		return nil
	}
	r.hasNew = false
	r.reading, r.ready = r.ready, r.reading
	return r.reading
}
