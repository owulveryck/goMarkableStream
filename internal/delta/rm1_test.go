package delta

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/klauspost/compress/zstd"
)

// packRGB565LE packs 8-bit R, G, B into a little-endian RGB565 uint16.
func packRGB565LE(r, g, b uint8) (byte, byte) {
	v := (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
	return byte(v), byte(v >> 8)
}

func TestRGB565ToColor8_AllValues(t *testing.T) {
	// Build a buffer containing all 65,536 possible RGB565 values
	src := make([]byte, 65536*2)
	for px := range 65536 {
		src[px*2] = byte(px)
		src[px*2+1] = byte(px >> 8)
	}

	dst := make([]byte, 65536)
	ConvertRGB565ToColor8(dst, src)

	for px := range 65536 {
		lo, hi := byte(px), byte(px>>8)
		want := computeRGB565ColorByte(lo, hi)
		if dst[px] != want {
			t.Fatalf("pixel 0x%04x (lo=0x%02x hi=0x%02x): got %d, want %d", px, lo, hi, dst[px], want)
		}
		if RGB565ToColorByte(lo, hi) != want {
			t.Fatalf("RGB565ToColorByte(0x%02x, 0x%02x) = %d, want %d", lo, hi, RGB565ToColorByte(lo, hi), want)
		}
	}

	// Test tail lengths 1..31 pixels (exercises non-multiple-of-16 / non-multiple-of-4 tails)
	for tail := 1; tail <= 31; tail++ {
		subDst := make([]byte, tail)
		ConvertRGB565ToColor8(subDst, src[:tail*2])
		if !bytes.Equal(subDst, dst[:tail]) {
			t.Fatalf("tail=%d mismatch", tail)
		}
	}

	// Verify semantic RM1 palette landmarks
	checkColor := func(name string, r, g, b, want byte) {
		t.Helper()
		lo, hi := packRGB565LE(r, g, b)
		got := RGB565ToColorByte(lo, hi)
		if got != want {
			t.Errorf("%s (%d,%d,%d -> 0x%02x%02x): got %d, want %d", name, r, g, b, hi, lo, got, want)
		}
	}
	checkColor("white paper (transparent)", 255, 255, 255, 30)
	checkColor("near-white paper", 250, 250, 250, 30)
	checkColor("black ink", 0, 0, 0, 0)
	checkColor("red pen", 255, 0, 0, 6)
	checkColor("blue pen", 0, 0, 255, 12)
	checkColor("green pen", 125, 184, 86, 20)
	checkColor("yellow highlighter", 255, 253, 84, 24)

	// Pure neutral grays must never collide with reserved color codes {6,8,12,20,24}
	for g := range 245 {
		lo, hi := packRGB565LE(byte(g), byte(g), byte(g))
		code := RGB565ToColorByte(lo, hi)
		switch code {
		case 6, 8, 12, 20, 24:
			t.Errorf("neutral gray %d produced reserved color code %d", g, code)
		}
	}
}

// decodeStreamBPP mirrors client/worker_stream_processing.js for arbitrary wire BPP (1, 2, or 4).
type decodeStreamBPP struct {
	t     *testing.T
	bpp   int
	dec   *zstd.Decoder
	frame []byte
	types []byte
}

func newDecodeStreamBPP(t *testing.T, numPixels, bpp int) *decodeStreamBPP {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("zstd reader: %v", err)
	}
	return &decodeStreamBPP{
		t:     t,
		bpp:   bpp,
		dec:   dec,
		frame: make([]byte, numPixels*bpp),
	}
}

func (d *decodeStreamBPP) feed(buf []byte) {
	d.t.Helper()
	for len(buf) >= 4 {
		frameType := buf[0]
		payloadLen := int(buf[1]) | int(buf[2])<<8 | int(buf[3])<<16
		if len(buf) < 4+payloadLen {
			d.t.Fatalf("truncated frame: have %d, need %d", len(buf), 4+payloadLen)
		}
		payload := buf[4 : 4+payloadLen]
		buf = buf[4+payloadLen:]
		d.types = append(d.types, frameType)

		switch frameType {
		case FrameTypeFullZstd:
			full, err := d.dec.DecodeAll(payload, nil)
			if err != nil {
				d.t.Fatalf("full frame inflate: %v", err)
			}
			if len(full) != len(d.frame) {
				d.t.Fatalf("full frame size %d, want %d", len(full), len(d.frame))
			}
			copy(d.frame, full)
		case FrameTypeDelta:
			d.applyRuns(payload)
		case FrameTypeDeltaZstd:
			runs, err := d.dec.DecodeAll(payload, nil)
			if err != nil {
				d.t.Fatalf("delta inflate: %v", err)
			}
			d.applyRuns(runs)
		default:
			d.t.Fatalf("unknown frame type 0x%02x", frameType)
		}
	}
	if len(buf) != 0 {
		d.t.Fatalf("%d trailing bytes", len(buf))
	}
}

func (d *decodeStreamBPP) applyRuns(payload []byte) {
	d.t.Helper()
	pos, frameOffset := 0, 0
	for pos < len(payload) {
		lengthByte := payload[pos]
		var runLength, relativeOffset, dataStart int
		if lengthByte&0x80 == 0 {
			if pos+3 > len(payload) {
				d.t.Fatal("truncated short run header")
			}
			runLength = int(lengthByte)
			relativeOffset = int(binary.LittleEndian.Uint16(payload[pos+1 : pos+3]))
			dataStart = pos + 3
		} else {
			if pos+5 > len(payload) {
				d.t.Fatal("truncated long run header")
			}
			runLength = int(lengthByte&0x7F)<<8 | int(payload[pos+1])
			relativeOffset = int(payload[pos+2]) | int(payload[pos+3])<<8 | int(payload[pos+4])<<16
			dataStart = pos + 5
		}
		dataLen := runLength * d.bpp
		pos = dataStart + dataLen
		if pos > len(payload) {
			d.t.Fatalf("delta run exceeds payload: pos=%d len=%d", pos, len(payload))
		}
		frameOffset += relativeOffset
		if frameOffset+dataLen > len(d.frame) {
			d.t.Fatalf("delta run exceeds frame bounds: %d > %d", frameOffset+dataLen, len(d.frame))
		}
		copy(d.frame[frameOffset:frameOffset+dataLen], payload[dataStart:pos])
		frameOffset += dataLen
	}
}

func drawRGB565Stroke(frame []byte, width, row, col, length int, lo, hi byte) {
	for i := range length {
		off := (row*width + col + i) * 2
		if off+2 > len(frame) {
			return
		}
		frame[off] = lo
		frame[off+1] = hi
	}
}

func TestRoundTrip_RM1Color8(t *testing.T) {
	const (
		width     = 1408
		height    = 1872
		numPixels = width * height
		sizeBytes = numPixels * 2
	)

	enc := NewColor8Encoder(DefaultThreshold)
	client := newDecodeStreamBPP(t, numPixels, 1)
	wantColor8 := make([]byte, numPixels)

	// Start with white RGB565 paper (0xFFFF)
	fb := make([]byte, sizeBytes)
	for i := range fb {
		fb[i] = 0xFF
	}

	colors := [][2]byte{
		{0x00, 0x00}, // black
		{0x00, 0xF8}, // red (LE: lo=0x00, hi=0xF8)
		{0x1F, 0x00}, // blue (LE: lo=0x1F, hi=0x00)
	}

	// Initial full frame
	var buf bytes.Buffer
	if err := enc.Encode(fb, &buf); err != nil {
		t.Fatalf("initial encode: %v", err)
	}
	client.feed(buf.Bytes())
	ConvertRGB565ToColor8(wantColor8, fb)
	if !bytes.Equal(client.frame, wantColor8) {
		t.Fatal("initial full frame mismatch")
	}
	t.Logf("RM1 Color8 initial blank page wire size: %d bytes", buf.Len())

	// Progressive strokes in black, red, and blue
	var totalStrokeWireBytes int
	const steps = 20
	for step := range steps {
		c := colors[step%len(colors)]
		for rowOff := range 3 {
			drawRGB565Stroke(fb, width, 200+step*12+rowOff, 150+step*5, 320, c[0], c[1])
		}
		buf.Reset()
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatalf("step %d encode: %v", step, err)
		}
		totalStrokeWireBytes += buf.Len()
		client.feed(buf.Bytes())
		ConvertRGB565ToColor8(wantColor8, fb)
		if !bytes.Equal(client.frame, wantColor8) {
			t.Fatalf("step %d: decoded Color8 frame differs from expected", step)
		}
	}
	t.Logf("RM1 Color8 average stroke delta wire size: %d bytes/frame", totalStrokeWireBytes/steps)

	// Idle check via EncodeIfChanged: must write 0 bytes twice in a row (transition + steady idle)
	for idle := range 3 {
		buf.Reset()
		n, changed, err := enc.EncodeIfChanged(fb, &buf)
		if err != nil {
			t.Fatalf("idle %d: %v", idle, err)
		}
		if changed || n != 0 || buf.Len() != 0 {
			t.Fatalf("idle %d: got n=%d changed=%v buf=%d, want 0,false,0", idle, n, changed, buf.Len())
		}
	}

	// Simulate physical page-turn button during idle (full screen change)
	for i := 0; i < len(fb); i += 2 {
		if (i/2)%17 == 0 {
			fb[i], fb[i+1] = 0x00, 0x00
		} else {
			fb[i], fb[i+1] = 0xFF, 0xFF
		}
	}
	buf.Reset()
	n, changed, err := enc.EncodeIfChanged(fb, &buf)
	if err != nil {
		t.Fatalf("page turn encode: %v", err)
	}
	if !changed || n == 0 {
		t.Fatal("expected EncodeIfChanged to detect page turn")
	}
	client.feed(buf.Bytes())
	ConvertRGB565ToColor8(wantColor8, fb)
	if !bytes.Equal(client.frame, wantColor8) {
		t.Fatal("page turn decoded frame mismatch")
	}
	t.Logf("RM1 Color8 dense page-turn wire size: %d bytes", n)
}

func TestRoundTrip_RM1RGB565(t *testing.T) {
	const (
		width     = 1408
		height    = 1872
		numPixels = width * height
		sizeBytes = numPixels * 2
	)

	enc := NewRGB565Encoder(DefaultThreshold)
	client := newDecodeStreamBPP(t, numPixels, 2)

	fb := make([]byte, sizeBytes)
	for i := range fb {
		fb[i] = 0xFF
	}

	var buf bytes.Buffer
	if err := enc.Encode(fb, &buf); err != nil {
		t.Fatal(err)
	}
	client.feed(buf.Bytes())
	if !bytes.Equal(client.frame, fb) {
		t.Fatal("RGB565 initial frame mismatch")
	}

	for step := range 10 {
		drawRGB565Stroke(fb, width, 400+step*9, 100+step*4, 250, 0x24, 0x11)
		buf.Reset()
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		client.feed(buf.Bytes())
		if !bytes.Equal(client.frame, fb) {
			t.Fatalf("step %d: RGB565 frame mismatch", step)
		}
	}
}

// TestLongRunSplitting_Over32767Pixels verifies that a single contiguous changed
// region larger than 32,767 pixels (the 15-bit long-run header limit) but below
// the 30% full-frame threshold is split into valid <=32,767-pixel runs rather
// than silently truncating run.length & 0x7FFF.
func TestLongRunSplitting_Over32767Pixels(t *testing.T) {
	const (
		numPixels     = 1408 * 1872 // 2,635,776 pixels
		changedPixels = 75_000      // > 2 * 32,767 (~2.8% of screen < 30% threshold)
		startPixel    = 100_000
	)

	t.Run("Color8", func(t *testing.T) {
		enc := NewColor8Encoder(DefaultThreshold)
		client := newDecodeStreamBPP(t, numPixels, 1)
		fb := make([]byte, numPixels*2)
		for i := range fb {
			fb[i] = 0xFF
		}
		var buf bytes.Buffer
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatal(err)
		}
		client.feed(buf.Bytes())

		for p := startPixel; p < startPixel+changedPixels; p++ {
			fb[p*2] = byte(p & 0x1F)
			fb[p*2+1] = 0x00
		}
		buf.Reset()
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatal(err)
		}
		client.feed(buf.Bytes())

		want := make([]byte, numPixels)
		ConvertRGB565ToColor8(want, fb)
		if !bytes.Equal(client.frame, want) {
			t.Fatal("decoded Color8 frame corrupted on >32767-pixel contiguous run")
		}
	})

	t.Run("BGRA", func(t *testing.T) {
		enc := NewEncoder(DefaultThreshold)
		client := newDecodeStreamBPP(t, numPixels, 4)
		fb := make([]byte, numPixels*4)
		var buf bytes.Buffer
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatal(err)
		}
		client.feed(buf.Bytes())

		for p := startPixel; p < startPixel+changedPixels; p++ {
			fb[p*4] = 0x42
			fb[p*4+3] = 0xFF
		}
		buf.Reset()
		if err := enc.Encode(fb, &buf); err != nil {
			t.Fatal(err)
		}
		client.feed(buf.Bytes())
		if !bytes.Equal(client.frame, fb) {
			t.Fatal("decoded BGRA frame corrupted on >32767-pixel contiguous run")
		}
	})
}

func TestChecksumChanged_PositionDependence(t *testing.T) {
	buf := make([]byte, 4*blockSize)
	buf[0] = 0xAA
	buf[blockSize] = 0x55
	var prev [16]byte

	if !checksumChanged(unsafe.Pointer(&buf[0]), 4, unsafe.Pointer(&prev[0])) {
		t.Fatal("first checksum call must report changed")
	}
	if checksumChanged(unsafe.Pointer(&buf[0]), 4, unsafe.Pointer(&prev[0])) {
		t.Fatal("identical buffer must report unchanged")
	}

	// Swap block 0 and block 1 (a byte-sum would falsely report unchanged!)
	var tmp [blockSize]byte
	copy(tmp[:], buf[0:blockSize])
	copy(buf[0:blockSize], buf[blockSize:2*blockSize])
	copy(buf[blockSize:2*blockSize], tmp[:])

	if !checksumChanged(unsafe.Pointer(&buf[0]), 4, unsafe.Pointer(&prev[0])) {
		t.Fatal("swapped blocks must be detected as changed by rotated XOR-fold checksum")
	}
}

// legacyScalarConvertRGB565 reproduces the pre-optimization scalar conversion from commit 948638f.
func legacyScalarConvertRGB565(data []byte) {
	for i := 0; i < len(data)-1; i += 2 {
		lo, hi := data[i], data[i+1]
		r5 := (hi >> 3) & 0x1F
		g6 := uint16((hi&0x07)<<3) | uint16((lo>>5)&0x07)
		b5 := lo & 0x1F
		r := uint16(r5) * 255 / 31
		g := g6 * 255 / 63
		b := uint16(b5) * 255 / 31
		var out byte
		switch {
		case r > 180 && g < 80 && b < 80:
			out = 6
		case r < 80 && g < 80 && b > 180:
			out = 12
		case g > 120 && r < 160 && b < 120 && g > r && g > b:
			out = 20
		case r > 180 && g > 180 && b < 120:
			out = 24
		default:
			gray := (r + g + b) / 3
			if gray > 245 {
				out = 30
			} else {
				scaled := byte(gray * 25 / 245)
				switch scaled {
				case 6, 8, 12, 20, 24:
					scaled++
				}
				out = scaled
			}
		}
		data[i] = out
	}
}

func BenchmarkRM1_LegacyScalarInPlace_FullFrame(b *testing.B) {
	const size = 1408 * 1872 * 2
	src := make([]byte, size)
	for i := range src {
		src[i] = byte(i*31 + 17)
	}
	work := make([]byte, size)
	b.SetBytes(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(work, src)
		legacyScalarConvertRGB565(work)
	}
}

func BenchmarkRM1_ConvertRGB565ToColor8_FullFrame(b *testing.B) {
	const npixels = 1408 * 1872
	src := make([]byte, npixels*2)
	for i := range src {
		src[i] = byte(i*31 + 17)
	}
	dst := make([]byte, npixels)
	b.SetBytes(npixels * 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertRGB565ToColor8(dst, src)
	}
}

func BenchmarkRM1_Color8Delta_HandwritingStroke(b *testing.B) {
	const (
		width  = 1408
		height = 1872
		size   = width * height * 2
	)
	enc := NewColor8Encoder(DefaultThreshold)
	frameA := make([]byte, size)
	for i := range frameA {
		frameA[i] = 0xFF
	}
	// Populaterealistic existing notes on the page (~5% ink)
	for row := 100; row < 1600; row += 28 {
		drawRGB565Stroke(frameA, width, row, 120, 1000, 0x00, 0x00)
	}
	frameB := make([]byte, size)
	copy(frameB, frameA)
	// Add a new pen stroke (~600 pixels across 4 scanlines)
	for r := range 4 {
		drawRGB565Stroke(frameB, width, 910+r, 400, 150, 0x00, 0x00)
	}

	var buf bytes.Buffer
	_ = enc.Encode(frameA, &buf)

	b.SetBytes(size)
	b.ReportAllocs()
	b.ResetTimer()
	var totalBytes int64
	for i := 0; i < b.N; i++ {
		buf.Reset()
		var src []byte
		if i%2 == 0 {
			src = frameB
		} else {
			src = frameA
		}
		n, _ := enc.EncodeWithSize(src, &buf)
		totalBytes += int64(n)
	}
	b.ReportMetric(float64(totalBytes)/float64(b.N), "wire_bytes/op")
}

func BenchmarkRM1_Color8Delta_Idle(b *testing.B) {
	const size = 1408 * 1872 * 2
	enc := NewColor8Encoder(DefaultThreshold)
	frame := make([]byte, size)
	for i := range frame {
		frame[i] = 0xFF
	}
	var buf bytes.Buffer
	_ = enc.Encode(frame, &buf)
	_, _, _ = enc.EncodeIfChanged(frame, &buf) // transition to idle

	b.SetBytes(size)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_, _, _ = enc.EncodeIfChanged(frame, &buf)
	}
}
