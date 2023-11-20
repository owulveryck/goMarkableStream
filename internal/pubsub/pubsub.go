package pubsub

import (
	"sync"
	"time"

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
	// Create a ticker for the timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for ch := range ps.subscribers {
		select {
		case ch <- event:
		case <-ticker.C:
		}
	}
}

// Subscribe to the topics to get the event published by the publishers
func (ps *PubSub) Subscribe(name string) chan events.InputEventFromSource {
	eventChan := make(chan events.InputEventFromSource)

	ps.mu.Lock()

	ps.subscribers[eventChan] = true
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

	}
}
