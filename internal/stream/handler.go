package stream

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
	"github.com/owulveryck/goMarkableStream/internal/rle"
)

const (
	rate = 100
)

var imagePool = sync.Pool{
	New: func() any {
		return make([]uint8, remarkable.ScreenWidth*remarkable.ScreenHeight) // Adjust the initial capacity as needed
	},
}

func NewStreamHandler(file io.ReaderAt, pointerAddr int64) *StreamHandler {
	return &StreamHandler{
		ticker:       time.NewTicker(rate * time.Millisecond),
		waitingQueue: make(chan struct{}, 1),
		file:         file,
		pointerAddr:  pointerAddr,
	}
}

type StreamHandler struct {
	ticker       *time.Ticker
	waitingQueue chan struct{}
	file         io.ReaderAt
	pointerAddr  int64
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case h.waitingQueue <- struct{}{}:
		defer func() {
			<-h.waitingQueue
		}()
		//ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		defer cancel()
		h.ticker.Reset(rate * time.Millisecond)
		defer h.ticker.Stop()

		imageData := imagePool.Get().([]uint8)
		defer imagePool.Put(imageData) // Return the slice to the pool when done
		// the informations are int4, therefore store it in a uint8array to reduce data transfer
		rleWriter := rle.NewRLE(w)

		for {
			select {
			case <-ctx.Done():
				return
			case <-h.ticker.C:
				_, err := h.file.ReadAt(imageData, h.pointerAddr)
				if err != nil {
					log.Fatal(err)
				}
				rleWriter.Write(imageData)
			}
		}
	default:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
}
