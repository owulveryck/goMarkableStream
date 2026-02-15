package delta

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/klauspost/compress/zstd"
)

func TestNewEncoder(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		expected  float64
	}{
		{"default threshold", 0, DefaultThreshold},
		{"negative threshold", -0.5, DefaultThreshold},
		{"threshold too high", 1.5, DefaultThreshold},
		{"valid threshold", 0.5, 0.5},
		{"edge threshold 1.0", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewEncoder(tt.threshold)
			if enc.threshold != tt.expected {
				t.Errorf("expected threshold %f, got %f", tt.expected, enc.threshold)
			}
		})
	}
}

func TestEncode_FullFrame_NoPrevious(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frame := make([]byte, 160) // 40 pixels
	for i := range frame {
		frame[i] = byte(i)
	}

	var buf bytes.Buffer
	err := enc.Encode(frame, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf.Bytes()

	// Check header - now expecting zstd compressed full frame
	if result[0] != FrameTypeFullZstd {
		t.Errorf("expected zstd compressed full frame type (0x03), got %d", result[0])
	}

	// Check payload length (24-bit LE)
	payloadLen := int(result[1]) | int(result[2])<<8 | int(result[3])<<16

	// Decompress and verify payload
	compressedPayload := result[4 : 4+payloadLen]
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("failed to create zstd reader: %v", err)
	}
	defer dec.Close()
	decompressed, err := dec.DecodeAll(compressedPayload, nil)
	if err != nil {
		t.Fatalf("failed to decompress payload: %v", err)
	}

	if !bytes.Equal(decompressed, frame) {
		t.Error("decompressed payload does not match frame data")
	}
}

func TestEncode_FullFrame_SizeChange(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	// First frame
	frame1 := make([]byte, 160)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("unexpected error on first frame: %v", err)
	}

	// Second frame with different size - should send full frame
	frame2 := make([]byte, 320)
	for i := range frame2 {
		frame2[i] = byte(i)
	}
	var buf2 bytes.Buffer
	err := enc.Encode(frame2, &buf2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf2.Bytes()
	if result[0] != FrameTypeFullZstd {
		t.Errorf("expected compressed full frame for size change (0x02), got %d", result[0])
	}
}

func TestEncode_DeltaFrame_SmallChange(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1600 // 400 pixels

	// First frame - all zeros
	frame1 := make([]byte, frameSize)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("unexpected error on first frame: %v", err)
	}

	// Second frame - change only first 4 pixels (16 bytes = 1% change)
	frame2 := make([]byte, frameSize)
	for i := 0; i < 16; i++ {
		frame2[i] = 0xFF
	}

	var buf2 bytes.Buffer
	err := enc.Encode(frame2, &buf2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf2.Bytes()

	// Should be delta frame since change is 1% < 30% threshold
	if result[0] != FrameTypeDelta {
		t.Errorf("expected delta frame type, got %d", result[0])
	}

	// Verify payload length in header
	payloadLen := int(result[1]) | int(result[2])<<8 | int(result[3])<<16

	// Should be much smaller than full frame
	if payloadLen >= frameSize {
		t.Errorf("delta payload (%d) should be smaller than full frame (%d)", payloadLen, frameSize)
	}
}

func TestEncode_FullFrame_ExceedsThreshold(t *testing.T) {
	enc := NewEncoder(0.10) // 10% threshold
	frameSize := 400        // 100 pixels

	// First frame - all zeros
	frame1 := make([]byte, frameSize)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("unexpected error on first frame: %v", err)
	}

	// Second frame - change 50% of pixels (exceeds 10% threshold)
	frame2 := make([]byte, frameSize)
	for i := 0; i < frameSize/2; i++ {
		frame2[i] = 0xFF
	}

	var buf2 bytes.Buffer
	err := enc.Encode(frame2, &buf2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf2.Bytes()

	// Should be compressed full frame since change exceeds threshold
	if result[0] != FrameTypeFullZstd {
		t.Errorf("expected compressed full frame when threshold exceeded (0x02), got %d", result[0])
	}
}

func TestEncode_Reset(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 160

	// First frame
	frame1 := make([]byte, frameSize)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("unexpected error on first frame: %v", err)
	}

	// Second frame with small change (would normally be delta)
	frame2 := make([]byte, frameSize)
	frame2[0] = 0xFF

	// Reset encoder
	enc.Reset()

	var buf2 bytes.Buffer
	err := enc.Encode(frame2, &buf2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf2.Bytes()

	// Should be compressed full frame after reset
	if result[0] != FrameTypeFullZstd {
		t.Errorf("expected compressed full frame after reset (0x02), got %d", result[0])
	}
}

func TestCompareFrames_NoChanges(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 160

	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = byte(i)
	}

	enc.prevFrame = make([]byte, frameSize)
	copy(enc.prevFrame, frame)
	enc.hasPrev = true

	runs := enc.compareFrames(frame)

	if len(runs) != 0 {
		t.Errorf("expected 0 runs for identical frames, got %d", len(runs))
	}
}

func TestEncode_NoChanges_EmptyDelta(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1600 // 400 pixels

	// First frame - establish baseline
	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = byte(i % 256)
	}
	var buf1 bytes.Buffer
	if err := enc.Encode(frame, &buf1); err != nil {
		t.Fatalf("unexpected error on first frame: %v", err)
	}

	// Second frame - identical to first (no changes)
	var buf2 bytes.Buffer
	err := enc.Encode(frame, &buf2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf2.Bytes()

	// Should be empty delta frame (type 0x01, payload size 0)
	if result[0] != FrameTypeDelta {
		t.Errorf("expected delta frame type (0x01), got %d", result[0])
	}

	// Check payload length is 0
	payloadLen := int(result[1]) | int(result[2])<<8 | int(result[3])<<16
	if payloadLen != 0 {
		t.Errorf("expected empty payload (length 0), got %d", payloadLen)
	}

	// Total frame should be just the 4-byte header
	if len(result) != 4 {
		t.Errorf("expected 4 bytes (header only), got %d", len(result))
	}
}

func TestCompareFrames_SingleRun(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 160 // 40 pixels

	// Previous frame - all zeros
	enc.prevFrame = make([]byte, frameSize)
	enc.hasPrev = true

	// Current frame - change first pixel
	current := make([]byte, frameSize)
	current[0] = 0xFF
	current[1] = 0xFF
	current[2] = 0xFF
	current[3] = 0xFF

	runs := enc.compareFrames(current)

	if len(runs) == 0 {
		t.Fatal("expected at least one run")
	}

	// First run should start at offset 0
	if runs[0].offset != 0 {
		t.Errorf("expected first run offset 0, got %d", runs[0].offset)
	}
}

func TestShortRunEncoding(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	run := changeRun{
		offset: 100,
		length: 10,
		data:   make([]byte, 40), // 10 pixels
	}
	for i := range run.data {
		run.data[i] = byte(i)
	}

	var buf bytes.Buffer
	err := enc.writeShortRun(run, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf.Bytes()

	// Check length byte
	if result[0] != 10 {
		t.Errorf("expected length 10, got %d", result[0])
	}

	// Check offset (little-endian)
	offset := binary.LittleEndian.Uint16(result[1:3])
	if offset != 100 {
		t.Errorf("expected offset 100, got %d", offset)
	}

	// Check data
	if !bytes.Equal(result[3:], run.data) {
		t.Error("run data mismatch")
	}
}

func TestLongRunEncoding(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	run := changeRun{
		offset: 70000, // > 64KB
		length: 200,   // > 127
		data:   make([]byte, 800),
	}

	var buf bytes.Buffer
	err := enc.writeLongRun(run, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := buf.Bytes()

	// Check high bit is set on first byte
	if result[0]&0x80 == 0 {
		t.Error("expected high bit set on first byte")
	}

	// Check length (15-bit)
	length := int(result[0]&0x7F)<<8 | int(result[1])
	if length != 200 {
		t.Errorf("expected length 200, got %d", length)
	}

	// Check offset (24-bit little-endian)
	offset := int(result[2]) | int(result[3])<<8 | int(result[4])<<16
	if offset != 70000 {
		t.Errorf("expected offset 70000, got %d", offset)
	}
}

func TestCalculateDeltaSize(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	tests := []struct {
		name     string
		runs     []changeRun
		expected int
	}{
		{
			name:     "empty",
			runs:     nil,
			expected: 0,
		},
		{
			name: "single short run",
			runs: []changeRun{
				{offset: 100, length: 10, data: make([]byte, 40)},
			},
			expected: 1 + 2 + 40, // length + offset + data
		},
		{
			name: "single long run",
			runs: []changeRun{
				{offset: 70000, length: 200, data: make([]byte, 800)},
			},
			expected: 2 + 3 + 800, // length + offset + data
		},
		{
			name: "mixed runs",
			runs: []changeRun{
				{offset: 100, length: 10, data: make([]byte, 40)},
				{offset: 70000, length: 200, data: make([]byte, 800)},
			},
			expected: (1 + 2 + 40) + (2 + 3 + 800),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := enc.calculateDeltaSize(tt.runs)
			if size != tt.expected {
				t.Errorf("expected size %d, got %d", tt.expected, size)
			}
		})
	}
}

func BenchmarkCompareFrames(b *testing.B) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4 // Typical reMarkable screen size

	enc.prevFrame = make([]byte, frameSize)
	current := make([]byte, frameSize)

	// Simulate 2% change (typical handwriting)
	changeBytes := frameSize * 2 / 100
	for i := 0; i < changeBytes; i++ {
		current[i] = 0xFF
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.compareFrames(current)
	}
}

func BenchmarkEncode_Delta(b *testing.B) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4

	frame1 := make([]byte, frameSize)
	enc.prevFrame = make([]byte, frameSize)
	enc.hasPrev = true

	// Simulate 2% change
	changeBytes := frameSize * 2 / 100
	for i := 0; i < changeBytes; i++ {
		frame1[i] = 0xFF
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frame1, &buf)
	}
}

// TestDeltaEncoderEmptyBuffers tests that the encoder handles empty buffers gracefully.
// This tests Bug #5 fix: unsafe memory access without bounds checking.
func TestDeltaEncoderEmptyBuffers(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	tests := []struct {
		name    string
		frame   []byte
		wantErr bool
	}{
		{
			name:    "empty buffer",
			frame:   []byte{},
			wantErr: false,
		},
		{
			name:    "nil buffer",
			frame:   nil,
			wantErr: false,
		},
		{
			name:    "single byte",
			frame:   []byte{0x01},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := enc.Encode(tt.frame, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCompareFramesEmptyBuffers tests that compareFrames handles empty buffers without panic.
func TestCompareFramesEmptyBuffers(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	// Set up encoder with empty previous frame
	enc.prevFrame = []byte{}
	enc.hasPrev = true

	// Should not panic with empty current frame
	runs := enc.compareFrames([]byte{})
	if len(runs) != 0 {
		t.Errorf("expected 0 runs for empty frames, got %d", len(runs))
	}
}

// TestCompareFramesMismatchedLengths tests handling of mismatched buffer lengths.
// After the fix, this should be handled by the EncodeWithSize check.
func TestEncodeWithMismatchedLengths(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)

	// First encode with 160 bytes
	frame1 := make([]byte, 160)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("First encode failed: %v", err)
	}

	// Second encode with different length - should send full frame, not delta
	frame2 := make([]byte, 320)
	var buf2 bytes.Buffer
	if err := enc.Encode(frame2, &buf2); err != nil {
		t.Fatalf("Second encode failed: %v", err)
	}

	// Verify it sent a full frame, not a delta
	result := buf2.Bytes()
	if result[0] != FrameTypeFullZstd {
		t.Errorf("expected full frame for size change, got frame type %d", result[0])
	}
}

func TestZSTDEncoderReset(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4 // RM2 size

	// Create compressible test data - frame1
	frame1 := make([]byte, frameSize)
	for i := range frame1 {
		frame1[i] = byte(i % 256)
	}

	// Create different test data - frame2 (different pattern to force full frame)
	frame2 := make([]byte, frameSize)
	for i := range frame2 {
		frame2[i] = byte((i + 128) % 256)
	}

	var buf bytes.Buffer

	// Encode first frame (always full frame)
	enc.Encode(frame1, &buf)
	size1 := buf.Len()

	// Reset encoder to force second frame to be full frame too
	enc.Reset()
	buf.Reset()

	// Encode second frame - should produce similar size
	enc.Encode(frame2, &buf)
	size2 := buf.Len()

	// Sizes should be within 10% (encoder not accumulating state)
	diff := abs(size1 - size2)
	tolerance := size1 / 10

	if diff > tolerance {
		t.Errorf("Encoder state may be leaking: size1=%d, size2=%d, diff=%d (tolerance=%d)",
			size1, size2, diff, tolerance)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestZSTDMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4

	frame1 := make([]byte, frameSize)
	frame2 := make([]byte, frameSize)

	// Different data in frame2 to trigger full frame encoding
	for i := 0; i < frameSize/10; i++ {
		frame2[i] = 0xFF
	}

	var buf bytes.Buffer

	// Measure memory before
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Encode many frames (alternate to trigger full frame encoding)
	for i := 0; i < 1000; i++ {
		buf.Reset()
		if i%2 == 0 {
			enc.Encode(frame1, &buf)
		} else {
			enc.Encode(frame2, &buf)
		}
	}

	// Measure memory after
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memGrowth := int64(m2.HeapAlloc - m1.HeapAlloc)

	// Should not grow more than 2 MB
	maxGrowthMB := int64(2 * 1024 * 1024)
	if memGrowth > maxGrowthMB {
		t.Errorf("Memory grew by %d MB after 1000 frames, expected < %d MB",
			memGrowth/1024/1024, maxGrowthMB/1024/1024)
	}

	t.Logf("Memory growth: %.2f MB", float64(memGrowth)/1024/1024)
}

func BenchmarkZSTDEncoding(b *testing.B) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4

	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		enc.Encode(frame, &buf)
	}
}

func TestCompressedBufferReuse(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4 // RM2 size

	frame1 := make([]byte, frameSize)
	for i := range frame1 {
		frame1[i] = byte(i % 256)
	}

	frame2 := make([]byte, frameSize)
	for i := range frame2 {
		frame2[i] = byte((i + 128) % 256)
	}

	var buf bytes.Buffer

	// Encode first frame (allocates compressedBuf)
	enc.Encode(frame1, &buf)
	firstCap := cap(enc.compressedBuf)

	// Reset encoder to force full frame
	enc.Reset()
	buf.Reset()

	// Encode second frame (should reuse compressedBuf)
	enc.Encode(frame2, &buf)
	secondCap := cap(enc.compressedBuf)

	// Capacity should remain stable (buffer reused)
	if firstCap == 0 {
		t.Fatal("compressedBuf not allocated on first frame")
	}

	if secondCap < firstCap {
		t.Errorf("compressedBuf capacity decreased: %d -> %d", firstCap, secondCap)
	}

	// Allow for small growth but not reallocation
	maxGrowth := firstCap / 10 // 10% tolerance
	if secondCap > firstCap+maxGrowth {
		t.Errorf("compressedBuf capacity grew too much: %d -> %d (max allowed: %d)",
			firstCap, secondCap, firstCap+maxGrowth)
	}
}

func TestReleaseMemoryCompressedBuf(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4

	frame := make([]byte, frameSize)
	var buf bytes.Buffer

	// Encode to allocate buffers
	enc.Encode(frame, &buf)

	// Verify buffer is allocated
	if cap(enc.compressedBuf) == 0 {
		t.Fatal("compressedBuf not allocated")
	}

	// Release memory
	enc.ReleaseMemory()

	// Verify buffer is released
	if enc.compressedBuf != nil {
		t.Error("compressedBuf not released by ReleaseMemory()")
	}
	if enc.prevFrame != nil {
		t.Error("prevFrame not released by ReleaseMemory()")
	}
}

// TestHashBasedEarlyExit_UnchangedFrame tests that unchanged frames use hash early exit
func TestHashBasedEarlyExit_UnchangedFrame(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4 // Full RM2 frame size

	// Create a frame with some data
	frame := make([]byte, frameSize)
	for i := 0; i < frameSize; i++ {
		frame[i] = byte(i % 256)
	}

	// First encode to establish baseline
	var buf1 bytes.Buffer
	if err := enc.Encode(frame, &buf1); err != nil {
		t.Fatalf("First encode failed: %v", err)
	}

	// Verify hash was computed
	expectedHash := xxhash.Sum64(frame)
	if enc.prevFrameHash != expectedHash {
		t.Errorf("prevFrameHash not set correctly: got %d, want %d", enc.prevFrameHash, expectedHash)
	}

	// Second encode with identical frame (should use hash early exit)
	var buf2 bytes.Buffer
	if err := enc.Encode(frame, &buf2); err != nil {
		t.Fatalf("Second encode failed: %v", err)
	}

	// Should produce empty delta frame (type 0x01, payload size 0)
	result := buf2.Bytes()
	if result[0] != FrameTypeDelta {
		t.Errorf("expected delta frame type (0x01), got %d", result[0])
	}

	payloadLen := int(result[1]) | int(result[2])<<8 | int(result[3])<<16
	if payloadLen != 0 {
		t.Errorf("expected empty payload (length 0), got %d", payloadLen)
	}
}

// TestHashBasedEarlyExit_ChangedFrame tests that changed frames skip hash early exit
func TestHashBasedEarlyExit_ChangedFrame(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 160

	// First frame - all zeros
	frame1 := make([]byte, frameSize)
	var buf1 bytes.Buffer
	if err := enc.Encode(frame1, &buf1); err != nil {
		t.Fatalf("First encode failed: %v", err)
	}

	hash1 := enc.prevFrameHash

	// Second frame - change one pixel
	frame2 := make([]byte, frameSize)
	frame2[0] = 0xFF // Change first byte

	var buf2 bytes.Buffer
	if err := enc.Encode(frame2, &buf2); err != nil {
		t.Fatalf("Second encode failed: %v", err)
	}

	// Hash should be different
	hash2 := enc.prevFrameHash
	if hash1 == hash2 {
		t.Error("hash should differ for changed frames")
	}

	// Should produce delta frame with changes
	result := buf2.Bytes()
	if result[0] != FrameTypeDelta {
		t.Errorf("expected delta frame type (0x01), got %d", result[0])
	}

	payloadLen := int(result[1]) | int(result[2])<<8 | int(result[3])<<16
	if payloadLen == 0 {
		t.Error("expected non-empty payload for changed frame")
	}
}

// TestHashBasedEarlyExit_Sequence tests alternating unchanged and changed frames
func TestHashBasedEarlyExit_Sequence(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1600

	// Frame A - pattern
	frameA := make([]byte, frameSize)
	for i := range frameA {
		frameA[i] = byte(i % 256)
	}

	// Frame B - different pattern
	frameB := make([]byte, frameSize)
	for i := range frameB {
		frameB[i] = byte((i + 100) % 256)
	}

	var buf bytes.Buffer

	// Sequence: A, A, B, B, A, A
	// Should see: full, empty-delta, delta/full, empty-delta, delta/full, empty-delta
	frames := [][]byte{frameA, frameA, frameB, frameB, frameA, frameA}
	expectedTypes := []byte{
		FrameTypeFullZstd, // First frame always full
		FrameTypeDelta,    // A->A: empty delta (hash match)
		FrameTypeFullZstd, // A->B: large change, full frame
		FrameTypeDelta,    // B->B: empty delta (hash match)
		FrameTypeFullZstd, // B->A: large change, full frame
		FrameTypeDelta,    // A->A: empty delta (hash match)
	}

	for i, frame := range frames {
		buf.Reset()
		if err := enc.Encode(frame, &buf); err != nil {
			t.Fatalf("Frame %d encode failed: %v", i, err)
		}

		result := buf.Bytes()
		if result[0] != expectedTypes[i] {
			t.Errorf("Frame %d: expected type %d, got %d", i, expectedTypes[i], result[0])
		}
	}
}

// TestHashComputation_Deterministic verifies hash is deterministic
func TestHashComputation_Deterministic(t *testing.T) {
	frame := make([]byte, 1000)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	hash1 := xxhash.Sum64(frame)
	hash2 := xxhash.Sum64(frame)

	if hash1 != hash2 {
		t.Errorf("hash should be deterministic: got %d and %d", hash1, hash2)
	}
}

// TestHashComputation_Sensitivity verifies hash changes on single byte change
func TestHashComputation_Sensitivity(t *testing.T) {
	frame1 := make([]byte, 1000)
	for i := range frame1 {
		frame1[i] = byte(i % 256)
	}

	frame2 := make([]byte, 1000)
	copy(frame2, frame1)
	frame2[500] = 0xFF // Change single byte

	hash1 := xxhash.Sum64(frame1)
	hash2 := xxhash.Sum64(frame2)

	if hash1 == hash2 {
		t.Error("hash should change when frame changes")
	}
}

// TestCompareFrames_HashEarlyExit verifies compareFrames uses hash early exit
func TestCompareFrames_HashEarlyExit(t *testing.T) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 10000

	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	// Set up encoder with previous frame
	enc.prevFrame = make([]byte, frameSize)
	copy(enc.prevFrame, frame)
	enc.prevFrameHash = xxhash.Sum64(frame)
	enc.hasPrev = true

	// Compare with identical frame (should use hash early exit)
	runs := enc.compareFrames(frame)

	// Should return empty runs
	if len(runs) != 0 {
		t.Errorf("expected 0 runs with hash early exit, got %d", len(runs))
	}

	// Hash should remain the same
	expectedHash := xxhash.Sum64(frame)
	if enc.prevFrameHash != expectedHash {
		t.Error("hash should be updated even on early exit")
	}
}

// BenchmarkCompareFrames_Unchanged_WithHash benchmarks hash early exit performance
func BenchmarkCompareFrames_Unchanged_WithHash(b *testing.B) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4 // RM2 size

	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	// Set up with previous frame
	enc.prevFrame = make([]byte, frameSize)
	copy(enc.prevFrame, frame)
	enc.prevFrameHash = xxhash.Sum64(frame)
	enc.hasPrev = true

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		enc.compareFrames(frame)
	}
}

// BenchmarkCompareFrames_SmallChange_WithHash benchmarks small change performance
func BenchmarkCompareFrames_SmallChange_WithHash(b *testing.B) {
	enc := NewEncoder(DefaultThreshold)
	frameSize := 1872 * 1404 * 4

	prev := make([]byte, frameSize)
	current := make([]byte, frameSize)

	// 1% change
	changeBytes := frameSize / 100
	for i := 0; i < changeBytes; i++ {
		current[i] = 0xFF
	}

	enc.prevFrame = prev
	enc.prevFrameHash = xxhash.Sum64(prev)
	enc.hasPrev = true

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		enc.compareFrames(current)
		enc.prevFrameHash = xxhash.Sum64(prev) // Reset for next iteration
	}
}

// BenchmarkHashComputation benchmarks just the hash computation speed
func BenchmarkHashComputation(b *testing.B) {
	frameSize := 1872 * 1404 * 4
	frame := make([]byte, frameSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(frame)
	}
}
