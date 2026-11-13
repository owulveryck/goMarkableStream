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
	mu          sync.RWMutex
}

// NewPubSub creates a new pubsub
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan events.InputEventFromSource]subscriber),
	}
}

// Publish an event to all subscribers.
//
// Sends happen while holding the read lock: Unsubscribe closes the channel
// under the write lock, so a send can never race with that close (sending on
// a closed channel panics, and Publish runs in the input scanner goroutine
// where nothing would recover it). Holding the lock is cheap because every
// send is non-blocking: a slow subscriber just drops the event.
func (ps *PubSub) Publish(event events.InputEventFromSource) {
	span := trace.BeginSpan("pubsub_publish")

	delivered := 0
	ps.mu.RLock()
	for ch, sub := range ps.subscribers {
		if sub.filter.Source != nil && *sub.filter.Source != event.Source {
			continue
		}
		if sub.filter.Type != nil && *sub.filter.Type != event.Type {
			continue
		}
		select {
		case ch <- event:
			delivered++
		default:
			// Channel full - subscriber is slow, drop event
		}
	}
	ps.mu.RUnlock()

	trace.EndSpan(span, map[string]any{
		"event_type":   event.Type,
		"event_source": event.Source,
		"subscribers":  delivered,
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
