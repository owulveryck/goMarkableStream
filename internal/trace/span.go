//go:build trace

package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Span represents a single operation timing measurement
type Span struct {
	Operation string
	StartTime time.Time
	started   bool // internal flag to track if span is valid
}

// SpanRecord is the serialized span data written to JSONL
type SpanRecord struct {
	Operation  string                 `json:"operation"`
	Start      float64                `json:"start"`      // Unix timestamp with fractional seconds
	DurationMS float64                `json:"duration_ms"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// spanRecorder manages span collection and writing
type spanRecorder struct {
	mu          sync.Mutex
	enabled     bool
	file        *os.File
	filename    string
	encoder     *json.Encoder
	spanChan    chan SpanRecord
	flushDone   chan struct{}
	bufferSize  int
	closeOnce   sync.Once
}

var recorder *spanRecorder

// spanPool reduces allocations for span objects
var spanPool = sync.Pool{
	New: func() any {
		return &Span{}
	},
}

// initSpanRecorder initializes the span recording system
func initSpanRecorder() error {
	if recorder != nil {
		return fmt.Errorf("span recorder already initialized")
	}

	filename := generateFilename("spans")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create span file: %w", err)
	}

	recorder = &spanRecorder{
		enabled:    true,
		file:       file,
		filename:   filename,
		encoder:    json.NewEncoder(file),
		spanChan:   make(chan SpanRecord, 10000), // Buffer up to 10k spans
		flushDone:  make(chan struct{}),
		bufferSize: 10000,
	}

	// Start background flusher
	go recorder.flusher()

	return nil
}

// flusher runs in the background and writes spans to the JSONL file
func (sr *spanRecorder) flusher() {
	defer close(sr.flushDone)

	for span := range sr.spanChan {
		if err := sr.encoder.Encode(span); err != nil {
			// Log error but continue - we don't want to crash the app
			fmt.Fprintf(os.Stderr, "trace: failed to write span: %v\n", err)
		}
	}

	// Flush any remaining buffered data
	if sr.file != nil {
		sr.file.Sync()
	}
}

// recordSpan sends a span to the recorder channel (non-blocking)
func (sr *spanRecorder) recordSpan(span SpanRecord) {
	if !sr.enabled {
		return
	}

	select {
	case sr.spanChan <- span:
		// Span queued successfully
	default:
		// Channel full, drop the span (avoid blocking)
		// This is acceptable for performance tracing
	}
}

// close stops the span recorder and flushes all pending spans
func (sr *spanRecorder) close() error {
	sr.closeOnce.Do(func() {
		sr.mu.Lock()
		sr.enabled = false
		sr.mu.Unlock()

		// Close the channel to signal flusher to stop
		close(sr.spanChan)

		// Wait for flusher to finish
		<-sr.flushDone

		// Close the file
		if sr.file != nil {
			sr.file.Close()
		}
	})

	return nil
}

// BeginSpan starts a new span for the given operation
// Returns nil if tracing is disabled (zero-cost when disabled)
func BeginSpan(operation string) *Span {
	if !Enabled || recorder == nil || !recorder.enabled {
		return nil
	}

	span := spanPool.Get().(*Span)
	span.Operation = operation
	span.StartTime = time.Now()
	span.started = true
	return span
}

// EndSpan completes a span and records it with optional metadata
// This is a no-op if span is nil (safe to call even when tracing is disabled)
func EndSpan(span *Span, metadata map[string]any) {
	if span == nil || !span.started {
		return
	}

	if recorder == nil || !recorder.enabled {
		// Return to pool even if recorder disabled
		span.started = false
		spanPool.Put(span)
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(span.StartTime)

	record := SpanRecord{
		Operation:  span.Operation,
		Start:      float64(span.StartTime.UnixNano()) / 1e9,
		DurationMS: float64(duration.Microseconds()) / 1000.0,
		Metadata:   metadata,
	}

	recorder.recordSpan(record)

	// Return span to pool for reuse
	span.started = false
	spanPool.Put(span)
}

// getSpanFilename returns the current span file name
func getSpanFilename() string {
	if recorder == nil {
		return ""
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.filename
}

// getSpanFileSize returns the current span file size
func getSpanFileSize() int64 {
	if recorder == nil {
		return 0
	}
	recorder.mu.Lock()
	filename := recorder.filename
	recorder.mu.Unlock()

	if filename == "" {
		return 0
	}

	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}
