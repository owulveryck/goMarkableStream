package pubsub

import (
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/events"
)

// EventFilter allows subscribers to filter events by source and type
type EventFilter struct {
	Source *int    // nil = all sources
	Type   *uint16 // nil = all types
}

type subscriber struct {
	ch     chan events.InputEventFromSource
	filter EventFilter
}

// PubSub is a structure to hold publisher and subscribers to events
type PubSub struct {
	subscribers map[chan events.InputEventFromSource]subscriber
	mu          sync.Mutex
}

// NewPubSub creates a new pubsub
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan events.InputEventFromSource]subscriber),
	}
}

// Publish an event to all subscribers
func (ps *PubSub) Publish(event events.InputEventFromSource) {
	// Copy subscriber list under lock, applying filters
	ps.mu.Lock()
	subscribers := make([]chan events.InputEventFromSource, 0, len(ps.subscribers))
	for ch, sub := range ps.subscribers {
		// Apply filter
		if sub.filter.Source != nil && *sub.filter.Source != event.Source {
			continue // Skip this subscriber
		}
		if sub.filter.Type != nil && *sub.filter.Type != event.Type {
			continue // Skip this subscriber
		}
		subscribers = append(subscribers, ch)
	}
	ps.mu.Unlock()

	// Send to matching subscribers without holding lock
	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Successfully sent
		default:
			// Channel full - subscriber is slow, drop event
		}
	}
}

// Subscribe to the topics to get the event published by the publishers
func (ps *PubSub) Subscribe(name string) chan events.InputEventFromSource {
	return ps.SubscribeWithFilter(name, EventFilter{})
}

// SubscribeWithFilter subscribes to events with optional filtering by source and type
func (ps *PubSub) SubscribeWithFilter(name string, filter EventFilter) chan events.InputEventFromSource {
	eventChan := make(chan events.InputEventFromSource, 100)

	ps.mu.Lock()
	ps.subscribers[eventChan] = subscriber{
		ch:     eventChan,
		filter: filter,
	}
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
