//go:build !linux

package remarkable

import (
	"os"
	"time"

	"context"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
)

// EventScanner listens to events on input2 and 3 and sends them to the EventC
type EventScanner struct {
	pen, touch *os.File
}

// NewEventScanner ...
func NewEventScanner() *EventScanner {
	return &EventScanner{}
}

// Start the event scanner and feed the EventC on movement. use the context to end the routine
func (e *EventScanner) StartAndPublish(ctx context.Context, pubsub *pubsub.PubSub) {
	go func(ctx context.Context) {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				pubsub.Publish(events.InputEventFromSource{
					Source:     1,
					InputEvent: events.InputEvent{},
				})
			}
		}
	}(ctx)
}
