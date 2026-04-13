package stream

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/owulveryck/goMarkableStream/internal/delta"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// =============================================================================
// IDLE SCREEN — tests xxhash early exit (identical frames)
// =============================================================================

func BenchmarkRealistic_IdleScreen_DeltaOnly(b *testing.B) {
	frames := buildIdleSequence(10)
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	// Prime the encoder with the first frame
	_ = enc.Encode(frames[0], &buf)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%len(frames)], &buf)
	}
}

func BenchmarkRealistic_IdleScreen_Pipeline(b *testing.B) {
	frames := buildIdleSequence(10)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// SINGLE STROKE — small delta (~0.5% change)
// =============================================================================

func BenchmarkRealistic_SingleStroke_DeltaOnly(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildProgressiveDrawingSequence(rng, 2) // blank + 1 stroke
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	// Prime with first frame
	_ = enc.Encode(frames[0], &buf)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[1], &buf)
	}
}

func BenchmarkRealistic_SingleStroke_Pipeline(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildProgressiveDrawingSequence(rng, 2)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// PROGRESSIVE DRAWING — incremental delta (~1-2% per frame)
// =============================================================================

func BenchmarkRealistic_ProgressiveDrawing_DeltaOnly(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildProgressiveDrawingSequence(rng, 20)
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%len(frames)], &buf)
	}
}

func BenchmarkRealistic_ProgressiveDrawing_Pipeline(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildProgressiveDrawingSequence(rng, 20)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// PAGE TURN — full refresh (>30% change between distinct pages)
// =============================================================================

func BenchmarkRealistic_PageTurn_DeltaOnly(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildPageTurnSequence(rng, 4) // 4 very different pages
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%len(frames)], &buf)
	}
}

func BenchmarkRealistic_PageTurn_Pipeline(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildPageTurnSequence(rng, 4)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// WRITING SESSION — realistic mixed workload (delta + idle + full refresh)
// =============================================================================

func BenchmarkRealistic_WritingSession_DeltaOnly(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildHandwritingSessionSequence(rng)
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%len(frames)], &buf)
	}
}

func BenchmarkRealistic_WritingSession_Pipeline(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildHandwritingSessionSequence(rng)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// HEAVY DRAWING — moderate delta (~5-10% per frame, 10 strokes each)
// =============================================================================

func BenchmarkRealistic_HeavyDrawing_DeltaOnly(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildHeavyDrawingSequence(rng, 10, 10)
	enc := delta.NewEncoder(0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%len(frames)], &buf)
	}
}

func BenchmarkRealistic_HeavyDrawing_Pipeline(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	frames := buildHeavyDrawingSequence(rng, 10, 10)
	seqReader := NewSequenceReaderAt(frames)
	eventPublisher := pubsub.NewPubSub()
	handler := NewStreamHandler(seqReader, 0, eventPublisher, 0.30)

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)
	rawData := make([]byte, benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		handler.fetchAndSendDelta(&buf, rawData)
	}
}

// =============================================================================
// FULL BLACK TO WHITE — maximum compression scenario (100% change)
// =============================================================================

func BenchmarkRealistic_FullBlackToWhite_DeltaOnly(b *testing.B) {
	black := make([]byte, benchFrameSize)
	fillBlack(black)
	white := make([]byte, benchFrameSize)
	fillWhite(white)

	enc := delta.NewEncoder(0.30)
	frames := [][]byte{black, white}

	var buf bytes.Buffer
	buf.Grow(benchFrameSize)

	b.SetBytes(int64(benchFrameSize))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Encode(frames[i%2], &buf)
	}
}

// =============================================================================
// PARAMETRIC: STROKE COUNT — varies number of strokes per frame
// =============================================================================

func BenchmarkRealistic_StrokeCount(b *testing.B) {
	for _, numStrokes := range []int{1, 5, 10, 20, 50} {
		b.Run(fmt.Sprintf("strokes_%d", numStrokes), func(b *testing.B) {
			rng := rand.New(rand.NewSource(42))
			frames := buildHeavyDrawingSequence(rng, 10, numStrokes)
			enc := delta.NewEncoder(0.30)

			var buf bytes.Buffer
			buf.Grow(benchFrameSize)

			b.SetBytes(int64(benchFrameSize))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = enc.Encode(frames[i%len(frames)], &buf)
			}
		})
	}
}

// =============================================================================
// PARAMETRIC: THRESHOLD — varies delta encoder threshold
// =============================================================================

func BenchmarkRealistic_Threshold(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	// Build frames with moderate change (~5-10%)
	frames := buildHeavyDrawingSequence(rng, 10, 10)

	for _, threshold := range []float64{0.10, 0.20, 0.30, 0.50} {
		b.Run(fmt.Sprintf("threshold_%.0f%%", threshold*100), func(b *testing.B) {
			enc := delta.NewEncoder(threshold)

			var buf bytes.Buffer
			buf.Grow(benchFrameSize)

			b.SetBytes(int64(benchFrameSize))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = enc.Encode(frames[i%len(frames)], &buf)
			}
		})
	}
}
