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
	imageData := rawFrameBuffer.Get().([]uint8)
	defer rawFrameBuffer.Put(imageData) // Return the slice to the pool when done
	_, err := h.file.ReadAt(imageData, h.pointerAddr)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(imageData)
}
