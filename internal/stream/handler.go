package stream

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/rle"
)

const (
	rate = 200
)

var rawFrameBuffer = sync.Pool{
	New: func() any {
		return make([]uint8, remarkable.ScreenWidth*remarkable.ScreenHeight*2) // Adjust the initial capacity as needed
	},
}

// NewStreamHandler creates a new stream handler reading from file @pointerAddr
func NewStreamHandler(file io.ReaderAt, pointerAddr int64, inputEvents *pubsub.PubSub) *StreamHandler {
	return &StreamHandler{
		ticker:         time.NewTicker(rate * time.Millisecond),
		waitingQueue:   make(chan struct{}, 2),
		file:           file,
		pointerAddr:    pointerAddr,
		inputEventsBus: inputEvents,
	}
}

// StreamHandler is an http.Handler that serves the stream of data to the client
type StreamHandler struct {
	ticker         *time.Ticker
	waitingQueue   chan struct{}
	file           io.ReaderAt
	pointerAddr    int64
	inputEventsBus *pubsub.PubSub
}

// ServeHTTP implements http.Handler
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case h.waitingQueue <- struct{}{}:
		eventC := h.inputEventsBus.Subscribe("stream")
		defer func() {
			<-h.waitingQueue
			h.inputEventsBus.Unsubscribe(eventC)
		}()
		//ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		defer cancel()
		h.ticker.Reset(rate * time.Millisecond)
		defer h.ticker.Stop()

		rawData := rawFrameBuffer.Get().([]uint8)
		defer rawFrameBuffer.Put(rawData) // Return the slice to the pool when done
		// the informations are int4, therefore store it in a uint8array to reduce data transfer
		rleWriter := rle.NewRLE(w)
		extractor := &oneOutOfTwo{rleWriter}
		writing := true
		stopWriting := time.NewTicker(2 * time.Second)
		defer stopWriting.Stop()

		w.Header().Set("Content-Type", "application/octet-stream")

		for {
			select {
			case <-r.Context().Done():
				log.Println("disconnected")
				return
			case <-ctx.Done():
				log.Println("disconnected")
				return
			case <-eventC:
				writing = true
				stopWriting.Reset(2 * time.Second)
			case <-stopWriting.C:
				writing = false
			case <-h.ticker.C:
				if writing {
					_, err := h.file.ReadAt(rawData, h.pointerAddr)
					if err != nil {
						log.Fatal(err)
					}
					extractor.Write(rawData)
				}
			}
		}
	default:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
}
