package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

var imagePool = sync.Pool{
	New: func() any {
		return make([]uint8, ScreenWidth*ScreenHeight) // Adjust the initial capacity as needed
	},
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	select {
	case waitingQueue <- struct{}{}:
		defer func() {
			<-waitingQueue
		}()
		//ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		defer cancel()
		ticker := time.NewTicker(time.Duration(c.Rate) * time.Millisecond) // Create a tick channel that emits a value every 200 milliseconds
		defer ticker.Stop()

		imageData := imagePool.Get().([]uint8)
		defer imagePool.Put(imageData) // Return the slice to the pool when done
		// the informations are int4, therefore store it in a uint8array to reduce data transfer
		rlewriter := &rleWriter{bufio.NewWriter(w)}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := file.ReadAt(imageData, pointerAddr)
				if err != nil {
					log.Fatal(err)
				}
				rlewriter.Write(imageData)
			}
		}
	default:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
}
