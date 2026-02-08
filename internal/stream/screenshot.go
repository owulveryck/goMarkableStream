package stream

import (
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

// NewScreenshotHandler creates a new screenshot handler reading from file @pointerAddr
func NewScreenshotHandler(file io.ReaderAt, pointerAddr int64) *ScreenshotHandler {
	return &ScreenshotHandler{
		file:        file,
		pointerAddr: pointerAddr,
	}
}

// ScreenshotHandler is an http.Handler that serves PNG screenshots of the framebuffer
type ScreenshotHandler struct {
	file        io.ReaderAt
	pointerAddr int64
}

// ServeHTTP implements http.Handler
func (h *ScreenshotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	imageDataPtr := rawFrameBuffer.Get().(*[]uint8)
	imageData := *imageDataPtr
	defer rawFrameBuffer.Put(imageDataPtr)

	_, err := h.file.ReadAt(imageData, h.pointerAddr)
	if err != nil {
		log.Printf("failed to read framebuffer: %v", err)
		http.Error(w, "failed to read framebuffer", http.StatusInternalServerError)
		return
	}

	width := remarkable.Config.Width
	height := remarkable.Config.Height

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Convert framebuffer to RGBA
	// All devices use BGRA format (4 bytes per pixel)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcIdx := (y*width + x) * 4
			dstIdx := (y*width + x) * 4

			// BGRA to RGBA: swap B and R channels
			img.Pix[dstIdx+0] = imageData[srcIdx+2] // R <- B
			img.Pix[dstIdx+1] = imageData[srcIdx+1] // G <- G
			img.Pix[dstIdx+2] = imageData[srcIdx+0] // B <- R
			img.Pix[dstIdx+3] = 255                 // A (fully opaque)
		}
	}

	// Generate filename with timestamp
	filename := "remarkable_" + time.Now().Format("20060102_150405") + ".png"

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-cache")

	if err := png.Encode(w, img); err != nil {
		log.Printf("failed to encode PNG: %v", err)
	}
}
