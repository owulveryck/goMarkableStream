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

	var img *image.RGBA
	switch {
	case remarkable.Model == remarkable.Remarkable1 || remarkable.Config.BytesPerPixel == 2 && !remarkable.Config.UseBGRA && width == 1408:
		// reMarkable 1: RGB565 little-endian, 1408 row stride with 1404 visible columns
		visWidth := width
		if width == 1408 {
			visWidth = 1404
		}
		img = image.NewRGBA(image.Rect(0, 0, visWidth, height))
		for y := 0; y < height; y++ {
			rowSrc := y * width * 2
			rowDst := y * visWidth * 4
			for x := 0; x < visWidth; x++ {
				srcIdx := rowSrc + x*2
				if srcIdx+1 >= len(imageData) {
					break
				}
				lo := imageData[srcIdx]
				hi := imageData[srcIdx+1]
				r8 := (hi & 0xF8) | (hi >> 5)
				g6 := ((hi & 0x07) << 3) | (lo >> 5)
				g8 := (g6 << 2) | (g6 >> 4)
				b5 := lo & 0x1F
				b8 := (b5 << 3) | (b5 >> 2)

				dstIdx := rowDst + x*4
				img.Pix[dstIdx+0] = r8
				img.Pix[dstIdx+1] = g8
				img.Pix[dstIdx+2] = b8
				img.Pix[dstIdx+3] = 255
			}
		}
	case remarkable.Config.BytesPerPixel == remarkable.BytesPerPixelGray16:
		// Legacy RM2 pre-3.24: 16-bit grayscale (2 bytes/pixel)
		img = image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcIdx := (y*width + x) * 2
				if srcIdx >= len(imageData) {
					break
				}
				v := imageData[srcIdx] * 17 // 4-bit gray (0..15) → 0..255
				if imageData[srcIdx] > 15 {
					v = imageData[srcIdx]
				}
				dstIdx := (y*width + x) * 4
				img.Pix[dstIdx+0] = v
				img.Pix[dstIdx+1] = v
				img.Pix[dstIdx+2] = v
				img.Pix[dstIdx+3] = 255
			}
		}
	default:
		// RM2 (3.24+) and Paper Pro: BGRA32 (4 bytes/pixel)
		img = image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcIdx := (y*width + x) * 4
				if srcIdx+3 >= len(imageData) {
					break
				}
				dstIdx := (y*width + x) * 4
				img.Pix[dstIdx+0] = imageData[srcIdx+2] // R <- B
				img.Pix[dstIdx+1] = imageData[srcIdx+1] // G <- G
				img.Pix[dstIdx+2] = imageData[srcIdx+0] // B <- R
				img.Pix[dstIdx+3] = 255                 // A (fully opaque)
			}
		}
	}

	filename := "remarkable_" + time.Now().Format("20060102_150405") + ".png"

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-cache")

	if err := png.Encode(w, img); err != nil {
		log.Printf("failed to encode PNG: %v", err)
	}
}
