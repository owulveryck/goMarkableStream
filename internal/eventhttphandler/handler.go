package eventhttphandler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

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
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "cannot upgrade connection "+err.Error(), http.StatusInternalServerError)
		return
	}
	eventC := h.inputEventBus.Subscribe("eventListener")
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()

	for event := range eventC {
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
		// Send the JSON message to the WebSocket client
		err = wsutil.WriteServerText(conn, jsonMessage)
		if err != nil {
			log.Println(err)
			http.Error(w, "cannot send message "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
