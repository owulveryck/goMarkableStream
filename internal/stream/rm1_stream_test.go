package stream

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"syscall"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/delta"
	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/rle"
)

// mockMmapFB simulates MmapFramebufferReader via remarkable.DirectFramebuffer
// and counts any fallback ReadAt calls.
type mockMmapFB struct {
	mu          sync.Mutex
	buf         []byte
	readAtCalls atomic.Int64
}

func (m *mockMmapFB) Bytes() []byte {
	return m.buf
}

func (m *mockMmapFB) ReadAt(p []byte, off int64) (int, error) {
	m.readAtCalls.Add(1)
	if off < 0 || int(off) >= len(m.buf) {
		return 0, errors.New("out of range")
	}
	return copy(p, m.buf[off:]), nil
}

func withRM1Config(t testing.TB, width, height int) {
	t.Helper()
	origModel := remarkable.Model
	origCfg := remarkable.Config
	origScreenW := remarkable.ScreenWidth
	origScreenH := remarkable.ScreenHeight
	origSize := remarkable.ScreenSizeBytes

	remarkable.Model = remarkable.Remarkable1
	remarkable.ScreenWidth = width
	remarkable.ScreenHeight = height
	remarkable.ScreenSizeBytes = width * height * 2
	remarkable.Config = remarkable.FramebufferConfig{
		Width:          width,
		Height:         height,
		BytesPerPixel:  2,
		SizeBytes:      width * height * 2,
		UseBGRA:        false,
		TextureFlipped: false,
	}
	ResetFrameBufferPool()

	t.Cleanup(func() {
		remarkable.Model = origModel
		remarkable.Config = origCfg
		remarkable.ScreenWidth = origScreenW
		remarkable.ScreenHeight = origScreenH
		remarkable.ScreenSizeBytes = origSize
		ResetFrameBufferPool()
	})
}

type syncedResponseWriter struct {
	hdr     http.Header
	mu      sync.Mutex
	buf     bytes.Buffer
	writes  int
	notifyC chan struct{}
}

func newSyncedResponseWriter() *syncedResponseWriter {
	return &syncedResponseWriter{
		hdr:     http.Header{},
		notifyC: make(chan struct{}, 32),
	}
}

func (s *syncedResponseWriter) Header() http.Header { return s.hdr }
func (s *syncedResponseWriter) WriteHeader(int)     {}
func (s *syncedResponseWriter) Flush()              {}
func (s *syncedResponseWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	n, err := s.buf.Write(p)
	s.writes++
	s.mu.Unlock()
	select {
	case s.notifyC <- struct{}{}:
	default:
	}
	return n, err
}

func (s *syncedResponseWriter) snapshot() ([]byte, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, s.buf.Len())
	copy(out, s.buf.Bytes())
	return out, s.writes
}

func TestRM1_StreamHandler_DirectMmapAndZeroIdleTraffic(t *testing.T) {
	// Use compact 128B-multiple dimensions so test runs in milliseconds
	const width, height = 128, 64
	withRM1Config(t, width, height)

	origIdle := rm1IdlePollInterval
	origCooldown := rm1PenLiftCooldown
	rm1IdlePollInterval = 40 * time.Millisecond
	rm1PenLiftCooldown = 30 * time.Millisecond
	defer func() {
		rm1IdlePollInterval = origIdle
		rm1PenLiftCooldown = origCooldown
	}()

	mmapBuf, mmapErr := syscall.Mmap(-1, 0, width*height*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_SHARED)
	if mmapErr != nil {
		t.Fatalf("syscall.Mmap: %v", mmapErr)
	}
	defer syscall.Munmap(mmapBuf)
	fb := &mockMmapFB{buf: mmapBuf}
	for i := range fb.buf {
		fb.buf[i] = 0xFF
	}

	bus := pubsub.NewPubSub()
	h := NewStreamHandler(fb, 0, bus, delta.DefaultThreshold)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := newSyncedResponseWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := httptest.NewRequest(http.MethodGet, "/stream?rate=25", nil).WithContext(ctx)
		h.ServeHTTP(w, req)
	}()

	// 1. Initial full frame must arrive immediately using DirectFramebuffer (0 ReadAt syscalls)
	select {
	case <-w.notifyC:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial RM1 full frame")
	}
	data, _ := w.snapshot()
	if len(data) < 4 || data[0] != delta.FrameTypeFullZstd {
		t.Fatalf("expected FrameTypeFullZstd (0x03), got %v", data)
	}
	if calls := fb.readAtCalls.Load(); calls != 0 {
		t.Fatalf("expected 0 ReadAt calls with DirectFramebuffer mmap, got %d", calls)
	}

	// 2. Simulate pen touch + lift so stream transitions to idle quickly via cooldown
	bus.Publish(events.InputEventFromSource{
		Source:     events.Pen,
		InputEvent: events.InputEvent{Type: events.EvAbs, Code: 24, Value: 1800}, // > 1000
	})
	bus.Publish(events.InputEventFromSource{
		Source:     events.Pen,
		InputEvent: events.InputEvent{Type: events.EvAbs, Code: 24, Value: 200}, // hover <= 1000 -> cooldown
	})
	time.Sleep(90 * time.Millisecond) // allow cooldown (30ms) + at least one idle poll (40ms)

	// Drain notifications and verify 0 network bytes are emitted across multiple idle polls
	_, writesBeforeIdle := w.snapshot()
	// Hovering laser pointer (pressure 400 <= 1000) must NOT wake frame streaming
	bus.Publish(events.InputEventFromSource{
		Source:     events.Pen,
		InputEvent: events.InputEvent{Type: events.EvAbs, Code: 24, Value: 400},
	})
	time.Sleep(100 * time.Millisecond)
	_, writesAfterIdle := w.snapshot()
	if writesAfterIdle != writesBeforeIdle {
		t.Fatalf("expected 0 network writes during RM1 idle/hover, got %d new writes", writesAfterIdle-writesBeforeIdle)
	}

	// 3. Mutate mmap buffer without publishing any EvAbs event (simulating RM1 hardware page-turn button)
	fb.buf[0] = 0x00
	fb.buf[1] = 0xF8 // red pixel
	deadline := time.After(2 * time.Second)
	for {
		_, curWrites := w.snapshot()
		if curWrites > writesAfterIdle {
			break
		}
		select {
		case <-w.notifyC:
		case <-deadline:
			t.Fatal("RM1 idle poll did not detect hardware-button screen change")
		}
	}

	if calls := fb.readAtCalls.Load(); calls != 0 {
		t.Fatalf("expected 0 ReadAt calls throughout mmap session, got %d", calls)
	}

	cancel()
	<-done
}

func TestRM1_ScreenshotHandler_NoPanicAndCrops1404(t *testing.T) {
	// Full RM1 dimensions: 1408 stride width (1404 visible + 4 padding), 1872 height, 2 B/px RGB565
	withRM1Config(t, 1408, 1872)

	fb := &mockMmapFB{buf: make([]byte, 1408*1872*2)}
	for i := range fb.buf {
		fb.buf[i] = 0xFF
	}
	// Paint pixel (0, 0) pure red in RGB565 LE (0xF800 -> lo=0x00, hi=0xF8)
	fb.buf[0] = 0x00
	fb.buf[1] = 0xF8

	sh := NewScreenshotHandler(fb, 0)
	rec := httptest.NewRecorder()
	sh.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/screenshot", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("png decode failed: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 1404 || bounds.Dy() != 1872 {
		t.Fatalf("screenshot bounds = %dx%d, want 1404x1872", bounds.Dx(), bounds.Dy())
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Fatalf("pixel (0,0) = (%d,%d,%d,%d), want (255,0,0,255)", r>>8, g>>8, b>>8, a>>8)
	}
}

// BenchmarkRM1_LegacyRLE_HandwritingStroke measures the pre-optimization RM1 pipeline
// (ReadAt copy + byte sum + full-frame scalar RGB565 conversion + full-frame RLE).
func BenchmarkRM1_LegacyRLE_HandwritingStroke(b *testing.B) {
	withRM1Config(b, 1408, 1872)
	const size = 1408 * 1872 * 2

	pageA := make([]byte, size)
	for i := range pageA {
		pageA[i] = 0xFF
	}
	// Realistic page of handwritten notes (~6% dark pixels with anti-aliased edges)
	for y := 80; y < 1780; y += 24 {
		for x := 100; x < 1300; x++ {
			if (x*7+y*3)%11 < 4 {
				idx := (y*1408 + x) * 2
				g := byte(((x + y) % 16) << 3)
				v := (uint16(g>>3) << 11) | (uint16(g>>2) << 5) | uint16(g>>3)
				pageA[idx] = byte(v)
				pageA[idx+1] = byte(v >> 8)
			}
		}
	}
	pageB := make([]byte, size)
	copy(pageB, pageA)
	for y := 900; y < 904; y++ {
		for x := 400; x < 550; x++ {
			idx := (y*1408 + x) * 2
			pageB[idx], pageB[idx+1] = 0x00, 0x00
		}
	}

	rawBuf := make([]byte, size)
	var out bytes.Buffer
	rleWriter := rle.NewRLE(&out)

	b.SetBytes(size)
	b.ReportAllocs()
	b.ResetTimer()
	var wireBytes int64
	for i := 0; i < b.N; i++ {
		out.Reset()
		src := pageA
		if i%2 == 1 {
			src = pageB
		}
		copy(rawBuf, src)
		// Legacy byte-sum + scalar per-pixel conversion + full-screen RLE
		s := 0
		for _, v := range rawBuf {
			s += int(v)
		}
		_ = s
		for p := 0; p < len(rawBuf)-1; p += 2 {
			lo, hi := rawBuf[p], rawBuf[p+1]
			r := uint16((hi>>3)&0x1F) * 255 / 31
			g := (uint16((hi&0x07)<<3) | uint16((lo>>5)&0x07)) * 255 / 63
			bl := uint16(lo&0x1F) * 255 / 31
			gray := (r + g + bl) / 3
			if gray > 245 {
				rawBuf[p] = 30
			} else {
				sc := byte(gray * 25 / 245)
				switch sc {
				case 6, 8, 12, 20, 24:
					sc++
				}
				rawBuf[p] = sc
			}
		}
		n, _ := rleWriter.Write(rawBuf)
		wireBytes += int64(n)
	}
	b.ReportMetric(float64(wireBytes)/float64(b.N), "wire_bytes/op")
}

// BenchmarkRM1_OptimColor8Delta_HandwritingStroke measures the new rm1-optim pipeline
// (zero-copy mmap + Compare-Before-Convert + Color8 Delta + Zstd) on the exact same page.
func BenchmarkRM1_OptimColor8Delta_HandwritingStroke(b *testing.B) {
	withRM1Config(b, 1408, 1872)
	const size = 1408 * 1872 * 2

	pageA := make([]byte, size)
	for i := range pageA {
		pageA[i] = 0xFF
	}
	for y := 80; y < 1780; y += 24 {
		for x := 100; x < 1300; x++ {
			if (x*7+y*3)%11 < 4 {
				idx := (y*1408 + x) * 2
				g := byte(((x + y) % 16) << 3)
				v := (uint16(g>>3) << 11) | (uint16(g>>2) << 5) | uint16(g>>3)
				pageA[idx] = byte(v)
				pageA[idx+1] = byte(v >> 8)
			}
		}
	}
	pageB := make([]byte, size)
	copy(pageB, pageA)
	for y := 900; y < 904; y++ {
		for x := 400; x < 550; x++ {
			idx := (y*1408 + x) * 2
			pageB[idx], pageB[idx+1] = 0x00, 0x00
		}
	}

	enc := delta.NewColor8Encoder(delta.DefaultThreshold)
	var out bytes.Buffer
	_ = enc.Encode(pageA, &out)

	b.SetBytes(size)
	b.ReportAllocs()
	b.ResetTimer()
	var wireBytes int64
	for i := 0; i < b.N; i++ {
		out.Reset()
		src := pageB
		if i%2 == 1 {
			src = pageA
		}
		n, _ := enc.EncodeWithSize(src, &out)
		wireBytes += int64(n)
	}
	b.ReportMetric(float64(wireBytes)/float64(b.N), "wire_bytes/op")
}
