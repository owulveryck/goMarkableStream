package eventhttphandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

type SwipeDirection string

const (
	SwipeLeft  SwipeDirection = "Swipe Left"
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
	leftDistance, rightDistance, upDistance, downDistance int
}

func (g *gesture) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{ "left": %v, "right": %v, "up": %v, "down": %v}`+"\n", g.leftDistance, g.rightDistance, g.upDistance, g.downDistance)), nil
}

func (g *gesture) String() string {
	return fmt.Sprintf("Left: %v, Right: %v, Up: %v, Down: %v", g.leftDistance, g.rightDistance, g.upDistance, g.downDistance)
}

func (g *gesture) sum() int {
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
	eventC := h.inputEventBus.Subscribe("eventListener")
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()
	const (
		codeXAxis   uint16 = 54
		codeYAxis   uint16 = 53
		maxStepDist int32  = 150
		// a gesture in a set of event separated by 100 millisecond
		gestureMaxInterval = 50 * time.Millisecond
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
			if event.Source != events.Touch {
				continue
			}
			if event.Type != events.EvAbs {
				continue
			}
			switch event.Code {
			case codeXAxis:
				// This is the initial event, do not compute the distance
				if lastEventX.Value == 0 {
					lastEventX = event
					continue
				}
				distance := event.Value - lastEventX.Value
				if distance < 0 {
					currentGesture.rightDistance += -int(distance)
				} else {
					currentGesture.leftDistance += int(distance)
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
					currentGesture.upDistance += -int(distance)
				} else {
					currentGesture.downDistance += int(distance)
				}
				lastEventY = event
			}
			tick.Reset(gestureMaxInterval)
		}
	}
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// timevalToTime converts syscall.Timeval to time.Time
func timevalToTime(tv syscall.Timeval) time.Time {
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)
}
