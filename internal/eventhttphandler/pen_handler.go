package eventhttphandler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// NewEventHandler creates an event habdler that subscribes from the inputEvents
func NewEventHandler(inputEvents *pubsub.PubSub) *EventHandler {
	return &EventHandler{
		inputEventBus: inputEvents,
	}
}

// EventHandler is a http.Handler that servers the input events over http via wabsockets
type EventHandler struct {
	inputEventBus *pubsub.PubSub
}

// ServeHTTP implements http.Handler
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventC := h.inputEventBus.Subscribe("eventListener")
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()
	// Set necessary headers to indicate a stream
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-eventC:
			// Serialize the structure as JSON
			if event.Source != events.Pen {
				continue
			}
			if event.Type != events.EvAbs {
				continue
			}
			jsonMessage, err := json.Marshal(event)
			if err != nil {
				http.Error(w, "cannot send json encode the message "+err.Error(), http.StatusInternalServerError)
				return
			}
			// Send the event
			fmt.Fprintf(w, "data: %s\n\n", jsonMessage)
			if f, ok := w.(http.Flusher); ok {
				f.Flush() // Ensure client receives the message immediately
			}

		}
	}
}
