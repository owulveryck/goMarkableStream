package pubsub

import (
	"sync"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/events"
)

// PubSub is a structure to hold publisher and subscribers to events
type PubSub struct {
	subscribers map[chan events.InputEventFromSource]bool
	mu          sync.Mutex
}

// NewPubSub creates a new pubsub
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan events.InputEventFromSource]bool),
	}
}

// Publish an event to all subscribers
func (ps *PubSub) Publish(event events.InputEventFromSource) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	debug.Log("PubSub: publishing event code=%d", event.Code)
	for ch := range ps.subscribers {
		select {
		case ch <- event:
		case <-time.After(100 * time.Millisecond):
			// Timeout - subscriber is too slow, drop event
			debug.Log("PubSub: dropped event for slow subscriber (code=%d)", event.Code)
		}
	}
}

// Subscribe to the topics to get the event published by the publishers
func (ps *PubSub) Subscribe(name string) chan events.InputEventFromSource {
	eventChan := make(chan events.InputEventFromSource)

	ps.mu.Lock()
	ps.subscribers[eventChan] = true
	debug.Log("PubSub: new subscriber '%s', total=%d", name, len(ps.subscribers))
	ps.mu.Unlock()

	return eventChan
}

// Unsubscribe from the events
func (ps *PubSub) Unsubscribe(ch chan events.InputEventFromSource) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.subscribers[ch]; ok {
		delete(ps.subscribers, ch)
		close(ch) // Close the channel to signal subscriber to exit.
		debug.Log("PubSub: unsubscribed, remaining=%d", len(ps.subscribers))
	}
}
