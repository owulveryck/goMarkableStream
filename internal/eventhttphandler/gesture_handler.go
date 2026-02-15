package eventhttphandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// SwipeDirection ...
type SwipeDirection string

const (
	// SwipeLeft ...
	SwipeLeft SwipeDirection = "Swipe Left"
	// SwipeRight ...
	SwipeRight SwipeDirection = "Swipe Right"
)

// NewGestureHandler creates an event habdler that subscribes from the inputEvents
func NewGestureHandler(inputEvents *pubsub.PubSub) *GestureHandler {
	return &GestureHandler{
		inputEventBus: inputEvents,
	}
}

// GestureHandler is a http.Handler that detect touch gestures
type GestureHandler struct {
	inputEventBus *pubsub.PubSub
}

type gesture struct {
	leftDistance, rightDistance, upDistance, downDistance int64
}

func (g *gesture) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{ "left": %v, "right": %v, "up": %v, "down": %v}`+"\n", g.leftDistance, g.rightDistance, g.upDistance, g.downDistance)), nil
}

func (g *gesture) String() string {
	return fmt.Sprintf("Left: %v, Right: %v, Up: %v, Down: %v", g.leftDistance, g.rightDistance, g.upDistance, g.downDistance)
}

func (g *gesture) sum() int64 {
	return g.leftDistance + g.rightDistance + g.upDistance + g.downDistance
}

func (g *gesture) reset() {
	g.leftDistance = 0
	g.rightDistance = 0
	g.upDistance = 0
	g.downDistance = 0
}

// ServeHTTP implements http.Handler
func (h *GestureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Subscribe only to Touch events of type EvAbs
	touchSource := events.Touch
	absType := uint16(events.EvAbs)
	eventC := h.inputEventBus.SubscribeWithFilter("eventListener", pubsub.EventFilter{
		Source: &touchSource,
		Type:   &absType,
	})
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()
	const (
		codeXAxis   uint16 = 54
		codeYAxis   uint16 = 53
		maxStepDist int32  = 150
		// a gesture in a set of event separated by 100 millisecond
		gestureMaxInterval = 150 * time.Millisecond
	)

	tick := time.NewTicker(gestureMaxInterval)
	defer tick.Stop()
	currentGesture := &gesture{}
	lastEventX := events.InputEventFromSource{}
	lastEventY := events.InputEventFromSource{}

	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/x-ndjson")

	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			// TODO send last event
			if currentGesture.sum() != 0 {
				err := enc.Encode(currentGesture)
				if err != nil {
					http.Error(w, "cannot send json encode the message "+err.Error(), http.StatusInternalServerError)
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
			currentGesture.reset()
			lastEventX = events.InputEventFromSource{}
			lastEventY = events.InputEventFromSource{}
		case event := <-eventC:
			switch event.Code {
			case codeXAxis:
				// This is the initial event, do not compute the distance
				if lastEventX.Value == 0 {
					lastEventX = event
					continue
				}
				distance := event.Value - lastEventX.Value
				if distance < 0 {
					currentGesture.rightDistance += -int64(distance)
				} else {
					currentGesture.leftDistance += int64(distance)
				}
				lastEventX = event
			case codeYAxis:
				// This is the initial event, do not compute the distance
				if lastEventY.Value == 0 {
					lastEventY = event
					continue
				}
				distance := event.Value - lastEventY.Value
				if distance < 0 {
					currentGesture.upDistance += -int64(distance)
				} else {
					currentGesture.downDistance += int64(distance)
				}
				lastEventY = event
			}
			tick.Reset(gestureMaxInterval)
		}
	}
}
