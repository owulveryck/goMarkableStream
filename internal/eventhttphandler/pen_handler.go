package eventhttphandler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

const (
	// pressureThreshold defines the minimum pressure value to consider the pen as "touching"
	// Values below this are considered "hovering" and should generate SSE events
	// Values above this mean the pen is drawing, so frame stream handles visualization
	pressureThreshold int32 = 100
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
	// Subscribe only to Pen events of type EvAbs
	penSource := events.Pen
	absType := uint16(events.EvAbs)
	eventC := h.inputEventBus.SubscribeWithFilter("eventListener", pubsub.EventFilter{
		Source: &penSource,
		Type:   &absType,
	})
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()
	// Set necessary headers to indicate a stream
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Track current pressure to determine if pen is hovering or drawing
	var currentPressure int32

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-eventC:
			// Update pressure tracking from ABS_PRESSURE events (code 24)
			if event.Code == 24 {
				currentPressure = event.Value
			}

			// Only send SSE events when pen is hovering (not touching)
			// When pen is down (drawing), the frame stream provides visual feedback
			// so individual coordinate events are redundant
			if currentPressure <= pressureThreshold {
				// Serialize the structure as JSON
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
}
