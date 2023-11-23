package eventhttphandler

import (
	"encoding/json"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

type SwipeDirection string

const (
	SwipeLeft  SwipeDirection = "Swipe Left"
	SwipeRight SwipeDirection = "Swipe Right"
)

// NewGestureHandler creates an event habdler that subscribes from the inputEvents
func NewGestureHandler(inputEvents *pubsub.PubSub) *EventHandler {
	return &EventHandler{
		inputEventBus: inputEvents,
	}
}

// GestureHandler is a http.Handler that detect touch gestures
type GestureHandler struct {
	inputEventBus *pubsub.PubSub
}

// ServeHTTP implements http.Handler
func (h *GestureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "cannot upgrade connection "+err.Error(), http.StatusInternalServerError)
		return
	}
	eventC := h.inputEventBus.Subscribe("eventListener")
	defer func() {
		h.inputEventBus.Unsubscribe(eventC)
	}()
	const (
		evAbs        uint16 = 3
		codeXAxis    uint16 = 54
		minSwipeDist int32  = 250
		maxSwipeTime        = 800 * time.Millisecond
		maxStepDist  int32  = 25
	)

	var (
		startTime  time.Time
		startValue int32
		lastValue  int32
		isSwiping  bool
		swipeRight bool
	)

	for event := range eventC {
		if event.Type == evAbs && event.Code == codeXAxis {
			currentTime := timevalToTime(event.Time)
			if !isSwiping {
				startTime = currentTime
				startValue = event.Value
				lastValue = event.Value
				isSwiping = true
			} else {
				if abs(event.Value-lastValue) > maxStepDist {
					isSwiping = false
					continue
				}

				if event.Value > lastValue {
					swipeRight = true
				} else if event.Value < lastValue {
					swipeRight = false
				}

				if abs(event.Value-startValue) >= minSwipeDist && currentTime.Sub(startTime) <= maxSwipeTime {
					if swipeRight {
						jsonMessage, err := json.Marshal(SwipeRight)
						if err != nil {
							http.Error(w, "cannot send json encode the message "+err.Error(), http.StatusInternalServerError)
							return
						}
						// Send the JSON message to the WebSocket client
						err = wsutil.WriteServerText(conn, jsonMessage)
						if err != nil {
							log.Println(err)
							return
						}
					} else {
						jsonMessage, err := json.Marshal(SwipeLeft)
						if err != nil {
							http.Error(w, "cannot send json encode the message "+err.Error(), http.StatusInternalServerError)
							return
						}
						// Send the JSON message to the WebSocket client
						err = wsutil.WriteServerText(conn, jsonMessage)
						if err != nil {
							log.Println(err)
							return
						}
					}
					isSwiping = false
				}
				lastValue = event.Value
			}
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
