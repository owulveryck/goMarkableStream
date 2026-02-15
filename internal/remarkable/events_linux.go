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
	closed     bool
}

// NewEventScanner ...
func NewEventScanner() *EventScanner {
	pen, err := os.OpenFile(PenInputDevice, os.O_RDONLY, 0o644)
	if err != nil {
		log.Fatalf("failed to read pen position: %v", err)
	}
	touch, err := os.OpenFile(TouchInputDevice, os.O_RDONLY, 0o644)
	if err != nil {
		log.Fatalf("failed to read touch position: %v", err)
	}
	return &EventScanner{
		pen:   pen,
		touch: touch,
	}
}

// Close closes the input device files. Safe to call multiple times.
func (e *EventScanner) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true

	var firstErr error
	if err := e.pen.Close(); err != nil {
		firstErr = err
	}
	if err := e.touch.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// StartAndPublish ...
func (e *EventScanner) StartAndPublish(ctx context.Context, pubsub *pubsub.PubSub) {
	// Start a goroutine to read events and send them on the channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
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
		}
	}()
	// Start a goroutine to read events and send them on the channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
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
		}
	}()
}

func readEvent(inputDevice *os.File) (events.InputEvent, error) {
	var ev events.InputEvent
	_, err := inputDevice.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
	return ev, err

}
