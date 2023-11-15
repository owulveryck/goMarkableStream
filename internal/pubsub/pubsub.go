package pubsub

import (
	"sync"

	"github.com/owulveryck/goMarkableStream/internal/events"
)

type PubSub struct {
	subscribers map[chan events.InputEventFromSource]bool
	mu          sync.Mutex
}

func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[chan events.InputEventFromSource]bool),
	}
}

func (ps *PubSub) Publish(event events.InputEventFromSource) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for ch := range ps.subscribers {
		ch <- event
	}
}

func (ps *PubSub) Subscribe(name string) chan events.InputEventFromSource {
	eventChan := make(chan events.InputEventFromSource)

	ps.mu.Lock()
	ps.subscribers[eventChan] = true
	ps.mu.Unlock()

	return eventChan
}
func (ps *PubSub) Unsubscribe(ch chan events.InputEventFromSource) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.subscribers[ch]; ok {
		delete(ps.subscribers, ch)
		close(ch) // Close the channel to signal subscriber to exit.
	}
}
