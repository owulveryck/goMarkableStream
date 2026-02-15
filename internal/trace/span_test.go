//go:build trace

package trace

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBeginEndSpan(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Create a span
	span := BeginSpan("test_operation")
	if span == nil {
		t.Fatal("BeginSpan returned nil")
	}
	if span.Operation != "test_operation" {
		t.Errorf("Expected operation 'test_operation', got '%s'", span.Operation)
	}
	if span.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	time.Sleep(10 * time.Millisecond)

	EndSpan(span, map[string]any{
		"test_key": "test_value",
		"count":    42,
	})

	files, err := Stop()
	if err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	// Read and verify JSONL file
	if len(files) == 0 {
		t.Fatal("No trace files created")
	}

	spanFile := files[0]
	content, err := os.ReadFile(spanFile)
	if err != nil {
		t.Fatalf("Failed to read span file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Span file is empty")
	}

	// Parse JSONL
	var record SpanRecord
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatalf("Failed to parse span record: %v", err)
	}

	if record.Operation != "test_operation" {
		t.Errorf("Expected operation 'test_operation', got '%s'", record.Operation)
	}
	if record.DurationMS < 10 {
		t.Errorf("Expected duration >= 10ms, got %.2fms", record.DurationMS)
	}
	if record.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if record.Metadata["test_key"] != "test_value" {
		t.Errorf("Expected metadata test_key='test_value', got %v", record.Metadata["test_key"])
	}
}

func TestSpanWithNilMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	span := BeginSpan("test")
	EndSpan(span, nil) // nil metadata should be ok

	Stop()
}

func TestMultipleSpans(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Record multiple spans
	operations := []string{"op1", "op2", "op3", "op4", "op5"}
	for _, op := range operations {
		span := BeginSpan(op)
		time.Sleep(1 * time.Millisecond)
		EndSpan(span, map[string]any{"operation": op})
	}

	files, err := Stop()
	if err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	// Read and verify all spans
	spanFile := files[0]
	file, err := os.Open(spanFile)
	if err != nil {
		t.Fatalf("Failed to open span file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		var record SpanRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("Failed to parse span record: %v", err)
		}
		count++
	}

	if count != len(operations) {
		t.Errorf("Expected %d spans, got %d", len(operations), count)
	}
}

func TestSpanWhenDisabled(t *testing.T) {
	// Test that spans are no-ops when disabled
	cfg := Config{
		Enabled:   false,
		Mode:      "spans",
		Dir:       t.TempDir(),
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	span := BeginSpan("test")
	if span != nil {
		t.Error("BeginSpan should return nil when disabled")
	}

	// Should not panic
	EndSpan(span, nil)
}

func TestEndSpanWithNilSpan(t *testing.T) {
	// EndSpan should handle nil span gracefully
	EndSpan(nil, nil)
	EndSpan(nil, map[string]any{"key": "value"})
}

// Benchmarks

func BenchmarkBeginEndSpanDisabled(b *testing.B) {
	// Benchmark overhead when tracing is disabled
	cfg := Config{
		Enabled:   false,
		Mode:      "spans",
		Dir:       b.TempDir(),
		MaxSizeMB: 10,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := BeginSpan("benchmark_op")
		EndSpan(span, nil)
	}
}

func BenchmarkBeginEndSpanEnabled(b *testing.B) {
	// Benchmark overhead when tracing is enabled
	tmpDir := b.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 100,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := BeginSpan("benchmark_op")
		EndSpan(span, nil)
	}
	b.StopTimer()

	Stop()
}

func BenchmarkBeginEndSpanWithMetadata(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 100,
		MaxFiles:  3,
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	if err := Start("spans"); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	metadata := map[string]any{
		"frame_size":    1234567,
		"changed_bytes": 12345,
		"change_ratio":  0.01,
		"runs":          15,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := BeginSpan("delta_compare")
		EndSpan(span, metadata)
	}
	b.StopTimer()

	Stop()
}

func BenchmarkSpanRecordWriting(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a span recorder
	filename := filepath.Join(tmpDir, "test.jsonl")
	file, err := os.Create(filename)
	if err != nil {
		b.Fatalf("Failed to create file: %v", err)
	}

	rec := &spanRecorder{
		enabled:    true,
		file:       file,
		filename:   filename,
		encoder:    json.NewEncoder(file),
		spanChan:   make(chan SpanRecord, 10000),
		flushDone:  make(chan struct{}),
		bufferSize: 10000,
	}

	go rec.flusher()

	record := SpanRecord{
		Operation:  "test_operation",
		Start:      float64(time.Now().UnixNano()) / 1e9,
		DurationMS: 5.5,
		Metadata: map[string]any{
			"key1": "value1",
			"key2": 123,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.recordSpan(record)
	}
	b.StopTimer()

	rec.close()
}

func TestFileRotationByCount(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		Mode:      "spans",
		Dir:       tmpDir,
		MaxSizeMB: 100,
		MaxFiles:  2, // Keep only 2 files
		AutoStart: false,
	}

	if err := Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create 3 trace files
	for i := 0; i < 3; i++ {
		if err := Start("spans"); err != nil {
			t.Fatalf("Failed to start trace %d: %v", i, err)
		}

		span := BeginSpan("test")
		EndSpan(span, map[string]any{"iteration": i})

		if _, err := Stop(); err != nil {
			t.Fatalf("Failed to stop trace %d: %v", i, err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Should have only 2 files (oldest one deleted)
	files, err := ListFiles()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) > 2 {
		t.Errorf("Expected max 2 files, got %d", len(files))
	}
}
