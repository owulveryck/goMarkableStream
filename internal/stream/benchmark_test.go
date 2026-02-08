package stream

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	"github.com/owulveryck/goMarkableStream/internal/delta"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

// FileReaderAt wraps a file for benchmarking real file I/O
type FileReaderAt struct {
	file *os.File
}

func NewFileReaderAt(path string) (*FileReaderAt, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &FileReaderAt{file: f}, nil
}

func (f *FileReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	return f.file.ReadAt(p, off)
}

func (f *FileReaderAt) Close() error {
	return f.file.Close()
}

// =============================================================================
// BUFFER POOL BENCHMARKS
// =============================================================================

// BenchmarkBufferPool_GetPut measures sync.Pool overhead
func BenchmarkBufferPool_GetPut(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := rawFrameBuffer.Get().(*[]uint8)
		rawFrameBuffer.Put(ptr)
	}
}

// BenchmarkBufferPool_GetPut_Parallel measures pool contention
func BenchmarkBufferPool_GetPut_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ptr := rawFrameBuffer.Get().(*[]uint8)
			rawFrameBuffer.Put(ptr)
		}
	})
}

// BenchmarkBufferAllocation compares pool vs direct allocation
func BenchmarkBufferAllocation(b *testing.B) {
	size := remarkable.Config.SizeBytes

	b.Run("DirectAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := make([]uint8, size)
			_ = buf
		}
	})

	b.Run("PoolAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := rawFrameBuffer.Get().(*[]uint8)
			rawFrameBuffer.Put(ptr)
		}
	})
}

// =============================================================================
// FRAMEBUFFER READ BENCHMARKS
// =============================================================================

// BenchmarkFramebufferRead_Mock measures in-memory read performance
func BenchmarkFramebufferRead_Mock(b *testing.B) {
	width := remarkable.Config.Width
	height := remarkable.Config.Height
	size := remarkable.Config.SizeBytes
	mock := NewMockReaderAt(width, height)
	buf := make([]byte, size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mock.ReadAt(buf, 0)
	}
}

// BenchmarkFramebufferRead_File measures real file I/O performance
func BenchmarkFramebufferRead_File(b *testing.B) {
	reader, err := NewFileReaderAt("../../testdata/full_memory_region.raw")
	if err != nil {
		b.Skip("testdata not available:", err)
	}
	defer reader.Close()

	size := remarkable.Config.SizeBytes
	buf := make([]byte, size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadAt(buf, 0)
	}
}

// =============================================================================
// DELTA ENCODING BENCHMARKS WITH VARIOUS CHANGE RATIOS
// =============================================================================

// benchmarkDeltaEncode runs delta encoding benchmark with specified change ratio
func benchmarkDeltaEncode(b *testing.B, changeRatio float64) {
	size := remarkable.Config.SizeBytes
	enc := delta.NewEncoder(0.30)

	// Initialize previous frame
	prevFrame := make([]byte, size)
	for i := range prevFrame {
		prevFrame[i] = byte(rand.Intn(256))
	}

	// First encode to set previous frame
	var initBuf bytes.Buffer
	_ = enc.Encode(prevFrame, &initBuf)

	// Create current frame with specified change ratio
	currentFrame := make([]byte, size)
	copy(currentFrame, prevFrame)

	changeBytes := int(float64(size) * changeRatio)
	for i := 0; i < changeBytes; i++ {
		currentFrame[i] = byte(rand.Intn(256))
	}

	var buf bytes.Buffer
	buf.Grow(size) // Pre-allocate to reduce allocations

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(currentFrame, &buf)
	}
}

func BenchmarkDeltaEncode_NoChange(b *testing.B) {
	benchmarkDeltaEncode(b, 0.0)
}

func BenchmarkDeltaEncode_1Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.01)
}

func BenchmarkDeltaEncode_2Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.02)
}

func BenchmarkDeltaEncode_5Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.05)
}

func BenchmarkDeltaEncode_10Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.10)
}

func BenchmarkDeltaEncode_30Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.30)
}

func BenchmarkDeltaEncode_50Percent(b *testing.B) {
	benchmarkDeltaEncode(b, 0.50)
}

func BenchmarkDeltaEncode_FullFrame(b *testing.B) {
	size := remarkable.Config.SizeBytes
	enc := delta.NewEncoder(0.30)

	frame := make([]byte, size)
	for i := range frame {
		frame[i] = byte(rand.Intn(256))
	}

	var buf bytes.Buffer
	buf.Grow(size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		enc.Reset() // Force full frame each time
		_ = enc.Encode(frame, &buf)
	}
}

// =============================================================================
// FULL PIPELINE BENCHMARKS
// =============================================================================

// BenchmarkFullPipeline_FetchAndEncode measures the complete fetch+encode cycle
func BenchmarkFullPipeline_FetchAndEncode(b *testing.B) {
	width := remarkable.Config.Width
	height := remarkable.Config.Height
	size := remarkable.Config.SizeBytes
	mock := NewMockReaderAt(width, height)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(mock, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(size)

	// Get a buffer from the pool
	rawDataPtr := rawFrameBuffer.Get().(*[]uint8)
	rawData := *rawDataPtr
	defer rawFrameBuffer.Put(rawDataPtr)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// BenchmarkFullPipeline_WithFileRead uses real file I/O
func BenchmarkFullPipeline_WithFileRead(b *testing.B) {
	reader, err := NewFileReaderAt("../../testdata/full_memory_region.raw")
	if err != nil {
		b.Skip("testdata not available:", err)
	}
	defer reader.Close()

	size := remarkable.Config.SizeBytes
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(reader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(size)

	rawDataPtr := rawFrameBuffer.Get().(*[]uint8)
	rawData := *rawDataPtr
	defer rawFrameBuffer.Put(rawDataPtr)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// PARALLEL PIPELINE BENCHMARKS
// =============================================================================

// BenchmarkFullPipeline_Parallel measures concurrent encoding performance
func BenchmarkFullPipeline_Parallel(b *testing.B) {
	width := remarkable.Config.Width
	height := remarkable.Config.Height
	size := remarkable.Config.SizeBytes
	mock := NewMockReaderAt(width, height)
	eventPublisher := pubsub.NewPubSub()

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets its own handler and buffer
		handler := NewStreamHandler(mock, 0, eventPublisher, 0.30)
		var buf bytes.Buffer
		buf.Grow(size)

		rawDataPtr := rawFrameBuffer.Get().(*[]uint8)
		rawData := *rawDataPtr
		defer rawFrameBuffer.Put(rawDataPtr)

		for pb.Next() {
			buf.Reset()
			handler.fetchAndSendDelta(&buf, rawData)
		}
	})
}

// =============================================================================
// COMPRESSION BENCHMARKS
// =============================================================================

// BenchmarkGzipCompression measures gzip overhead for full frames (e-ink pattern)
func BenchmarkGzipCompression_EinkPattern(b *testing.B) {
	size := remarkable.Config.SizeBytes
	enc := delta.NewEncoder(0.30)

	frame := make([]byte, size)
	// Simulate realistic e-ink content (mostly white with some black)
	for i := range frame {
		if rand.Float32() < 0.95 {
			frame[i] = 0xFF // white
		} else {
			frame[i] = 0x00 // black
		}
	}

	var buf bytes.Buffer
	buf.Grow(size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		enc.Reset()
		_ = enc.Encode(frame, &buf)
	}
}

// =============================================================================
// MEMORY COPY BENCHMARKS
// =============================================================================

// BenchmarkMemoryCopy measures raw memory copy performance (baseline)
func BenchmarkMemoryCopy(b *testing.B) {
	size := remarkable.Config.SizeBytes
	src := make([]byte, size)
	dst := make([]byte, size)
	for i := range src {
		src[i] = byte(rand.Intn(256))
	}

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		copy(dst, src)
	}
}

// =============================================================================
// COMPARISON BENCHMARKS
// =============================================================================

// BenchmarkFrameComparison measures frame comparison overhead
func BenchmarkFrameComparison(b *testing.B) {
	size := remarkable.Config.SizeBytes
	frame1 := make([]byte, size)
	frame2 := make([]byte, size)
	for i := range frame1 {
		frame1[i] = byte(rand.Intn(256))
	}
	copy(frame2, frame1)

	// Introduce 2% difference
	changeBytes := size * 2 / 100
	for i := 0; i < changeBytes; i++ {
		frame2[i] = byte(rand.Intn(256))
	}

	b.Run("bytes.Equal", func(b *testing.B) {
		b.SetBytes(int64(size))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = bytes.Equal(frame1, frame2)
		}
	})

	b.Run("LoopCompare", func(b *testing.B) {
		b.SetBytes(int64(size))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			equal := true
			for j := 0; j < size; j++ {
				if frame1[j] != frame2[j] {
					equal = false
					break
				}
			}
			_ = equal
		}
	})
}

// =============================================================================
// THRESHOLD IMPACT BENCHMARKS
// =============================================================================

// BenchmarkThresholdImpact measures how threshold affects performance
func BenchmarkThresholdImpact(b *testing.B) {
	thresholds := []struct {
		name  string
		value float64
	}{
		{"10%", 0.10},
		{"20%", 0.20},
		{"30%", 0.30},
		{"40%", 0.40},
		{"50%", 0.50},
	}
	size := remarkable.Config.SizeBytes

	for _, threshold := range thresholds {
		b.Run(threshold.name, func(b *testing.B) {
			enc := delta.NewEncoder(threshold.value)

			prevFrame := make([]byte, size)
			for i := range prevFrame {
				prevFrame[i] = byte(rand.Intn(256))
			}

			var initBuf bytes.Buffer
			_ = enc.Encode(prevFrame, &initBuf)

			// 15% change - near middle threshold
			currentFrame := make([]byte, size)
			copy(currentFrame, prevFrame)
			changeBytes := size * 15 / 100
			for i := 0; i < changeBytes; i++ {
				currentFrame[i] = byte(rand.Intn(256))
			}

			var buf bytes.Buffer
			buf.Grow(size)

			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = enc.Encode(currentFrame, &buf)
			}
		})
	}
}

// =============================================================================
// COMPONENT ISOLATION BENCHMARKS
// =============================================================================

// BenchmarkReadAt_Only isolates the ReadAt cost
func BenchmarkReadAt_Only(b *testing.B) {
	width := remarkable.Config.Width
	height := remarkable.Config.Height
	size := remarkable.Config.SizeBytes
	mock := NewMockReaderAt(width, height)

	buf := make([]byte, size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mock.ReadAt(buf, 0)
	}
}

// BenchmarkEncode_Only isolates the encoding cost (no ReadAt)
func BenchmarkEncode_Only(b *testing.B) {
	size := remarkable.Config.SizeBytes
	enc := delta.NewEncoder(0.30)

	// Pre-fill frame data
	frameData := make([]byte, size)
	for i := range frameData {
		frameData[i] = byte(rand.Intn(256))
	}

	var initBuf bytes.Buffer
	_ = enc.Encode(frameData, &initBuf)

	// Slightly modify frame (2% change)
	for i := 0; i < size*2/100; i++ {
		frameData[i] = byte(rand.Intn(256))
	}

	var buf bytes.Buffer
	buf.Grow(size)

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frameData, &buf)
	}
}

// BenchmarkPoolGetPut_Only isolates pool overhead
func BenchmarkPoolGetPut_Only(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ptr := rawFrameBuffer.Get().(*[]uint8)
		_ = *ptr // use it
		rawFrameBuffer.Put(ptr)
	}
}
