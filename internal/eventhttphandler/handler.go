package eventhttphandler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

func NewEventHandler(inputEvents *pubsub.PubSub) *EventHandler {
	return &EventHandler{
		inputEventBus: inputEvents,
	}
}

type EventHandler struct {
	inputEventBus *pubsub.PubSub
}

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
	writer := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(writer)

	for event := range eventC {
		err := encoder.Encode(event)
		if err != nil {
			log.Println(err)
			http.Error(w, "cannot send message "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
