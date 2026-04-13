package stream

import (
	"image"
	"math/rand"
	"sync/atomic"
	"testing"
)

// Benchmark frame dimensions — always BGRA, matching delta encoder's bytesPerPixel=4.
// Defined independently of remarkable.Config which defaults to Gray16 on non-arm64 dev machines.
const (
	benchWidth     = 1872
	benchHeight    = 1404
	benchBPP       = 4
	benchFrameSize = benchWidth * benchHeight * benchBPP // 10_501_632
)

// --- BGRA pixel helpers ---

func bgraWhite() [4]byte { return [4]byte{0xFF, 0xFF, 0xFF, 0xFF} }
func bgraBlack() [4]byte { return [4]byte{0x00, 0x00, 0x00, 0xFF} }

func bgraGray(v byte) [4]byte { return [4]byte{v, v, v, 0xFF} }

func setPixel(frame []byte, x, y int, pixel [4]byte) {
	if x < 0 || x >= benchWidth || y < 0 || y >= benchHeight {
		return
	}
	off := (y*benchWidth + x) * benchBPP
	frame[off+0] = pixel[0]
	frame[off+1] = pixel[1]
	frame[off+2] = pixel[2]
	frame[off+3] = pixel[3]
}

func fillWhite(frame []byte) {
	for i := 0; i < len(frame); i += 4 {
		frame[i+0] = 0xFF
		frame[i+1] = 0xFF
		frame[i+2] = 0xFF
		frame[i+3] = 0xFF
	}
}

func fillBlack(frame []byte) {
	for i := 0; i < len(frame); i += 4 {
		frame[i+0] = 0x00
		frame[i+1] = 0x00
		frame[i+2] = 0x00
		frame[i+3] = 0xFF
	}
}

// --- Bresenham line drawing ---

func drawLine(frame []byte, x0, y0, x1, y1 int, pixel [4]byte) {
	dx := x1 - x0
	dy := y1 - y0
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}

	err := dx - dy

	for {
		setPixel(frame, x0, y0, pixel)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawThickLine(frame []byte, x0, y0, x1, y1, thickness int, pixel [4]byte) {
	r := thickness / 2
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				drawLine(frame, x0+dx, y0+dy, x1+dx, y1+dy, pixel)
			}
		}
	}
}

// --- Handwriting stroke generation ---

// Stroke represents a sequence of connected points forming a pen stroke.
type Stroke struct {
	Points    []image.Point
	Thickness int
	Color     [4]byte
}

// drawStroke renders a stroke onto a frame by connecting successive points.
func drawStroke(frame []byte, s Stroke) {
	for i := 1; i < len(s.Points); i++ {
		drawThickLine(frame, s.Points[i-1].X, s.Points[i-1].Y,
			s.Points[i].X, s.Points[i].Y, s.Thickness, s.Color)
	}
}

// generateHandwritingStrokes creates strokes that mimic cursive handwriting
// within the given bounding box.
func generateHandwritingStrokes(rng *rand.Rand, bounds image.Rectangle, numStrokes int) []Stroke {
	strokes := make([]Stroke, 0, numStrokes)

	cursorX := bounds.Min.X + 50
	cursorY := bounds.Min.Y + 40
	lineHeight := 50

	for i := 0; i < numStrokes; i++ {
		// Each stroke is a short polyline (5-15 points) simulating a word fragment
		numPoints := 5 + rng.Intn(11)
		points := make([]image.Point, numPoints)

		x := cursorX
		y := cursorY
		points[0] = image.Pt(x, y)

		for j := 1; j < numPoints; j++ {
			// Advance horizontally 5-15 px, oscillate vertically ±10 px
			x += 5 + rng.Intn(11)
			y = cursorY + rng.Intn(21) - 10
			// Clamp within bounds
			if x >= bounds.Max.X-20 {
				x = bounds.Max.X - 20
			}
			if y < bounds.Min.Y+5 {
				y = bounds.Min.Y + 5
			}
			if y >= bounds.Max.Y-5 {
				y = bounds.Max.Y - 5
			}
			points[j] = image.Pt(x, y)
		}

		strokes = append(strokes, Stroke{
			Points:    points,
			Thickness: 2 + rng.Intn(2), // 2-3 px
			Color:     bgraBlack(),
		})

		// Advance cursor: word gap or line break
		cursorX = x + 15 + rng.Intn(20) // space between words
		if cursorX >= bounds.Max.X-100 {
			cursorX = bounds.Min.X + 50
			cursorY += lineHeight
			if cursorY >= bounds.Max.Y-40 {
				cursorY = bounds.Min.Y + 40 // wrap to top
			}
		}
	}

	return strokes
}

// --- Text block generator ---

// drawTextBlock fills a rectangular region with horizontal line segments simulating text rows.
func drawTextBlock(frame []byte, bounds image.Rectangle, lineHeight, lineThickness int) {
	black := bgraBlack()
	y := bounds.Min.Y

	for y+lineHeight <= bounds.Max.Y {
		// Draw a "text line": series of short horizontal segments with small gaps
		x := bounds.Min.X
		for x < bounds.Max.X-10 {
			// Word: 20-80 px wide
			wordLen := 20 + rand.Intn(61)
			endX := x + wordLen
			if endX > bounds.Max.X {
				endX = bounds.Max.X
			}
			for t := 0; t < lineThickness; t++ {
				drawLine(frame, x, y+t, endX, y+t, black)
			}
			x = endX + 5 + rand.Intn(10) // word gap
		}
		y += lineHeight
	}
}

// --- SequenceReaderAt ---

// SequenceReaderAt implements io.ReaderAt and cycles through a pre-built sequence of frames.
// Thread-safe via atomic counter.
type SequenceReaderAt struct {
	frames    [][]byte
	callCount int64
}

// NewSequenceReaderAt creates a SequenceReaderAt from the given frame sequence.
func NewSequenceReaderAt(frames [][]byte) *SequenceReaderAt {
	return &SequenceReaderAt{frames: frames}
}

// ReadAt copies the current frame into p and advances to the next frame.
func (s *SequenceReaderAt) ReadAt(p []byte, _ int64) (n int, err error) {
	idx := atomic.AddInt64(&s.callCount, 1) - 1
	frame := s.frames[idx%int64(len(s.frames))]
	n = copy(p, frame)
	return n, nil
}

// --- Sequence builders ---

// buildIdleSequence returns n identical white frames.
func buildIdleSequence(n int) [][]byte {
	frame := make([]byte, benchFrameSize)
	fillWhite(frame)
	frames := make([][]byte, n)
	for i := range frames {
		f := make([]byte, benchFrameSize)
		copy(f, frame)
		frames[i] = f
	}
	return frames
}

// buildProgressiveDrawingSequence returns numFrames frames where each adds one stroke.
func buildProgressiveDrawingSequence(rng *rand.Rand, numFrames int) [][]byte {
	bounds := image.Rect(100, 100, benchWidth-100, benchHeight-100)
	strokes := generateHandwritingStrokes(rng, bounds, numFrames)

	frames := make([][]byte, numFrames)
	base := make([]byte, benchFrameSize)
	fillWhite(base)

	for i := 0; i < numFrames; i++ {
		drawStroke(base, strokes[i])
		frame := make([]byte, benchFrameSize)
		copy(frame, base)
		frames[i] = frame
	}
	return frames
}

// buildPageTurnSequence returns numPages pairs of frames, each pair being a distinct page.
func buildPageTurnSequence(rng *rand.Rand, numPages int) [][]byte {
	frames := make([][]byte, numPages)
	for i := 0; i < numPages; i++ {
		frame := make([]byte, benchFrameSize)
		fillWhite(frame)
		// Each page gets a unique block of handwriting strokes in different positions
		yOffset := 80 + rng.Intn(200)
		bounds := image.Rect(80, yOffset, benchWidth-80, benchHeight-80)
		strokes := generateHandwritingStrokes(rng, bounds, 30+rng.Intn(20))
		for _, s := range strokes {
			drawStroke(frame, s)
		}
		// Also add text blocks at varying positions to maximize visual difference
		drawTextBlock(frame, image.Rect(100, 50+rng.Intn(100), benchWidth/2, 200+rng.Intn(200)), 18, 2)
		frames[i] = frame
	}
	return frames
}

// buildHeavyDrawingSequence returns numFrames frames where each adds many strokes (~5-10% change).
func buildHeavyDrawingSequence(rng *rand.Rand, numFrames, strokesPerFrame int) [][]byte {
	bounds := image.Rect(50, 50, benchWidth-50, benchHeight-50)

	frames := make([][]byte, numFrames)
	base := make([]byte, benchFrameSize)
	fillWhite(base)

	for i := 0; i < numFrames; i++ {
		strokes := generateHandwritingStrokes(rng, bounds, strokesPerFrame)
		for _, s := range strokes {
			drawStroke(base, s)
		}
		frame := make([]byte, benchFrameSize)
		copy(frame, base)
		frames[i] = frame
	}
	return frames
}

// buildHandwritingSessionSequence simulates a realistic writing session:
// write a few strokes, pause (repeat same frame), write more, then page turn.
func buildHandwritingSessionSequence(rng *rand.Rand) [][]byte {
	var frames [][]byte
	bounds := image.Rect(100, 100, benchWidth-100, benchHeight-100)

	base := make([]byte, benchFrameSize)
	fillWhite(base)

	// Phase 1: Write 5 strokes with pauses between
	strokes := generateHandwritingStrokes(rng, bounds, 5)
	for _, s := range strokes {
		drawStroke(base, s)
		frame := make([]byte, benchFrameSize)
		copy(frame, base)
		frames = append(frames, frame)
		// Add 2 idle frames (pause while thinking)
		for j := 0; j < 2; j++ {
			idle := make([]byte, benchFrameSize)
			copy(idle, base)
			frames = append(frames, idle)
		}
	}

	// Phase 2: Page turn — completely new content
	newPage := make([]byte, benchFrameSize)
	fillWhite(newPage)
	newStrokes := generateHandwritingStrokes(rng, bounds, 8)
	for _, s := range newStrokes {
		drawStroke(newPage, s)
	}
	drawTextBlock(newPage, image.Rect(100, 50, benchWidth-100, 250), 20, 2)
	frames = append(frames, newPage)

	// Phase 3: Continue writing on new page
	base2 := make([]byte, benchFrameSize)
	copy(base2, newPage)
	moreStrokes := generateHandwritingStrokes(rng, image.Rect(100, 300, benchWidth-100, benchHeight-100), 5)
	for _, s := range moreStrokes {
		drawStroke(base2, s)
		frame := make([]byte, benchFrameSize)
		copy(frame, base2)
		frames = append(frames, frame)
	}

	return frames
}

// --- Unit tests for the frame generator ---

func TestFillWhite_AllBytes(t *testing.T) {
	frame := make([]byte, benchFrameSize)
	fillWhite(frame)
	for i := 0; i < len(frame); i += 4 {
		if frame[i] != 0xFF || frame[i+1] != 0xFF || frame[i+2] != 0xFF || frame[i+3] != 0xFF {
			t.Fatalf("pixel at byte %d is not white: [%x, %x, %x, %x]",
				i, frame[i], frame[i+1], frame[i+2], frame[i+3])
		}
	}
}

func TestFillBlack_AllBytes(t *testing.T) {
	frame := make([]byte, benchFrameSize)
	fillBlack(frame)
	for i := 0; i < len(frame); i += 4 {
		if frame[i] != 0x00 || frame[i+1] != 0x00 || frame[i+2] != 0x00 || frame[i+3] != 0xFF {
			t.Fatalf("pixel at byte %d is not black: [%x, %x, %x, %x]",
				i, frame[i], frame[i+1], frame[i+2], frame[i+3])
		}
	}
}

func TestDrawLine_Horizontal(t *testing.T) {
	frame := make([]byte, benchFrameSize)
	fillWhite(frame)
	black := bgraBlack()
	drawLine(frame, 10, 50, 100, 50, black)

	// Verify pixels along the line are black
	for x := 10; x <= 100; x++ {
		off := (50*benchWidth + x) * benchBPP
		if frame[off] != 0x00 || frame[off+1] != 0x00 || frame[off+2] != 0x00 {
			t.Fatalf("pixel at (%d, 50) should be black, got [%x, %x, %x]",
				x, frame[off], frame[off+1], frame[off+2])
		}
	}

	// Verify a pixel outside the line is still white
	off := (50*benchWidth + 5) * benchBPP
	if frame[off] != 0xFF {
		t.Fatalf("pixel at (5, 50) should be white, got %x", frame[off])
	}
}

func TestDrawLine_Diagonal(t *testing.T) {
	frame := make([]byte, benchFrameSize)
	fillWhite(frame)
	black := bgraBlack()
	drawLine(frame, 0, 0, 50, 50, black)

	// The diagonal line from (0,0) to (50,50) should have black pixels at (0,0) and (50,50)
	for _, pt := range [][2]int{{0, 0}, {25, 25}, {50, 50}} {
		off := (pt[1]*benchWidth + pt[0]) * benchBPP
		if frame[off] != 0x00 || frame[off+1] != 0x00 || frame[off+2] != 0x00 {
			t.Fatalf("pixel at (%d, %d) should be black", pt[0], pt[1])
		}
	}
}

func TestSequenceReaderAt_Cycles(t *testing.T) {
	// Create 3 small distinct frames
	frames := make([][]byte, 3)
	for i := range frames {
		f := make([]byte, 16)
		for j := range f {
			f[j] = byte(i)
		}
		frames[i] = f
	}

	seq := NewSequenceReaderAt(frames)
	buf := make([]byte, 16)

	// Should cycle through 0, 1, 2, 0, 1, 2
	for cycle := 0; cycle < 2; cycle++ {
		for i := 0; i < 3; i++ {
			seq.ReadAt(buf, 0)
			expected := byte(i)
			if buf[0] != expected {
				t.Fatalf("cycle %d, read %d: expected %d, got %d", cycle, i, expected, buf[0])
			}
		}
	}
}

func TestGenerateHandwritingStrokes(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bounds := image.Rect(100, 100, 500, 400)
	strokes := generateHandwritingStrokes(rng, bounds, 5)

	if len(strokes) != 5 {
		t.Fatalf("expected 5 strokes, got %d", len(strokes))
	}
	for i, s := range strokes {
		if len(s.Points) < 5 {
			t.Errorf("stroke %d has only %d points, expected >= 5", i, len(s.Points))
		}
		if s.Thickness < 2 || s.Thickness > 3 {
			t.Errorf("stroke %d thickness %d out of range [2,3]", i, s.Thickness)
		}
	}
}

func TestBuildProgressiveDrawingSequence(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	frames := buildProgressiveDrawingSequence(rng, 5)

	if len(frames) != 5 {
		t.Fatalf("expected 5 frames, got %d", len(frames))
	}
	for i, f := range frames {
		if len(f) != benchFrameSize {
			t.Fatalf("frame %d size %d, expected %d", i, len(f), benchFrameSize)
		}
	}

	// Each successive frame should differ from the previous
	for i := 1; i < len(frames); i++ {
		same := true
		for j := 0; j < benchFrameSize; j++ {
			if frames[i][j] != frames[i-1][j] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("frame %d is identical to frame %d", i, i-1)
		}
	}
}
