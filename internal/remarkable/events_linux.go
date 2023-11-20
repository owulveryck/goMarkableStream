//go:build linux

package remarkable

import (
	"log"
	"os"
	"unsafe"

	"context"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// EventScanner ...
type EventScanner struct {
	pen, touch *os.File
}

// NewEventScanner ...
func NewEventScanner() *EventScanner {
	pen, err := os.OpenFile("/dev/input/event1", os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	touch, err := os.OpenFile("/dev/input/event2", os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	return &EventScanner{
		pen:   pen,
		touch: touch,
	}
}

// StartAndPublish ...
func (e *EventScanner) StartAndPublish(ctx context.Context, pubsub *pubsub.PubSub) {
	// Start a goroutine to read events and send them on the channel
	go func(_ context.Context) {
		for {
			ev, err := readEvent(e.pen)
			if err != nil {
				log.Println(err)
				continue
			}
			pubsub.Publish(events.InputEventFromSource{
				Source:     events.Pen,
				InputEvent: ev,
			})
		}
	}(ctx)
	// Start a goroutine to read events and send them on the channel
	go func(_ context.Context) {
		for {
			ev, err := readEvent(e.touch)
			if err != nil {
				log.Println(err)
				continue
			}
			pubsub.Publish(events.InputEventFromSource{
				Source:     events.Touch,
				InputEvent: ev,
			})
		}
	}(ctx)
}

func readEvent(inputDevice *os.File) (events.InputEvent, error) {
	var ev events.InputEvent
	_, err := inputDevice.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
	return ev, err

}
