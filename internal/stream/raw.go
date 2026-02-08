package stream

import (
	"io"
	"log"
	"net/http"
)

// NewRawHandler creates a new stream handler reading from file @pointerAddr
func NewRawHandler(file io.ReaderAt, pointerAddr int64) *RawHandler {
	return &RawHandler{
		waitingQueue: make(chan struct{}, 1),
		file:         file,
		pointerAddr:  pointerAddr,
	}
}

// RawHandler is an http.Handler that serves the stream of data to the client
type RawHandler struct {
	waitingQueue chan struct{}
	file         io.ReaderAt
	pointerAddr  int64
}

// ServeHTTP implements http.Handler
func (h *RawHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	imageDataPtr := rawFrameBuffer.Get().(*[]uint8)
	imageData := *imageDataPtr
	defer rawFrameBuffer.Put(imageDataPtr)
	_, err := h.file.ReadAt(imageData, h.pointerAddr)
	if err != nil {
		log.Printf("failed to read framebuffer: %v", err)
		http.Error(w, "failed to read framebuffer", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(imageData); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
