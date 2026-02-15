package pubsub

import (
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/debug"
	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/trace"
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
	mu          sync.RWMutex // Use RWMutex for better read concurrency
	slicePool   sync.Pool    // Pool for subscriber slice allocations
}

// NewPubSub creates a new pubsub
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan events.InputEventFromSource]subscriber),
		slicePool: sync.Pool{
			New: func() any {
				// Pre-allocate slice with small capacity
				// Will grow as needed
				s := make([]chan events.InputEventFromSource, 0, 8)
				return &s
			},
		},
	}
}

// Publish an event to all subscribers
func (ps *PubSub) Publish(event events.InputEventFromSource) {
	span := trace.BeginSpan("pubsub_publish")

	// Get slice from pool
	subscribersPtr := ps.slicePool.Get().(*[]chan events.InputEventFromSource)
	subscribers := (*subscribersPtr)[:0] // Reset to zero length, keep capacity

	// Copy subscriber list under read lock, applying filters
	ps.mu.RLock()
	for ch, sub := range ps.subscribers {
		// Apply filter - cache filter values to avoid pointer dereferences
		if sub.filter.Source != nil && *sub.filter.Source != event.Source {
			continue // Skip this subscriber
		}
		if sub.filter.Type != nil && *sub.filter.Type != event.Type {
			continue // Skip this subscriber
		}
		subscribers = append(subscribers, ch)
	}
	ps.mu.RUnlock()

	// Send to matching subscribers without holding lock
	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Successfully sent
		default:
			// Channel full - subscriber is slow, drop event
		}
	}

	// Return slice to pool
	*subscribersPtr = subscribers
	ps.slicePool.Put(subscribersPtr)

	trace.EndSpan(span, map[string]any{
		"event_type":   event.Type,
		"event_source": event.Source,
		"subscribers":  len(subscribers),
	})
}

// Subscribe to the topics to get the event published by the publishers
func (ps *PubSub) Subscribe(name string) chan events.InputEventFromSource {
	return ps.SubscribeWithFilter(name, EventFilter{})
}

// SubscribeWithFilter subscribes to events with optional filtering by source and type
func (ps *PubSub) SubscribeWithFilter(name string, filter EventFilter) chan events.InputEventFromSource {
	eventChan := make(chan events.InputEventFromSource, 100)

	ps.mu.Lock() // Full write lock for subscription
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
