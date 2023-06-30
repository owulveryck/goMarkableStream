package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	select {
	case waitingQueue <- struct{}{}:
		defer func() {
			<-waitingQueue
		}()
		// Generate a random integer between 0 and 100
		tick := time.Tick(200 * time.Millisecond) // Create a tick channel that emits a value every 200 milliseconds
		if c.Dev {
			tick = time.Tick(2000 * time.Millisecond) // Create a tick channel that emits a value every 200 milliseconds
		}
		timeout := time.Tick(1 * time.Hour)

		// Create a context with a cancellation function
		options := &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		}
		if c.Compression {
			options = &websocket.AcceptOptions{
				CompressionMode: websocket.CompressionContextTakeover,
			}
		}

		conn, err := websocket.Accept(w, r, options)
		//conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "Internal Server Error")

		// Simulated pixel data

		imageData := make([]byte, ScreenWidth*ScreenHeight)
		// the informations are int4, therefore store it in a uint8array to reduce data transfer

		for {
			select {
			case <-timeout:
				conn.Close(websocket.StatusNormalClosure, "timeout")
				return
			case <-r.Context().Done():
				return
			case <-tick:
				ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
				defer cancel()
				_, err := file.ReadAt(imageData, pointerAddr)
				if err != nil {
					log.Fatal(err)
				}
				uint8Array := encodeRLE(imageData)

				err = conn.Write(ctx, websocket.MessageBinary, uint8Array)
				if err != nil {
					//					log.Println(err)
					return
				}
			}
		}
	default:
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
}
