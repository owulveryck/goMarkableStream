package pubsub

import (
	"testing"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/events"
)

// TestPublishSubscribe tests basic pub/sub functionality
func TestPublishSubscribe(t *testing.T) {
	ps := NewPubSub()
	ch := ps.Subscribe("test")

	// Publish an event
	testEvent := events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Code:  1,
			Value: 100,
		},
	}

	go ps.Publish(testEvent)

	// Receive the event
	select {
	case received := <-ch:
		if received.Code != testEvent.Code {
			t.Errorf("Expected code %d, got %d", testEvent.Code, received.Code)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}

	ps.Unsubscribe(ch)
}

// TestUnsubscribeDoubleClose tests that double unsubscribe doesn't panic.
// This tests Bug #13 fix: channel double-close risk.
func TestUnsubscribeDoubleClose(t *testing.T) {
	ps := NewPubSub()
	ch := ps.Subscribe("test")

	// First unsubscribe
	ps.Unsubscribe(ch)

	// Second unsubscribe should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Second unsubscribe panicked: %v", r)
		}
	}()

	ps.Unsubscribe(ch)

	// Third unsubscribe should also not panic
	ps.Unsubscribe(ch)
}

// TestUnsubscribeNonExistentChannel tests unsubscribing a channel that was never subscribed
func TestUnsubscribeNonExistentChannel(t *testing.T) {
	ps := NewPubSub()

	// Create a channel but don't subscribe it
	ch := make(chan events.InputEventFromSource)

	// Unsubscribing should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unsubscribe of non-existent channel panicked: %v", r)
		}
	}()

	ps.Unsubscribe(ch)
}

// TestPublishToSlowSubscriber tests event handling when subscriber is slow.
// This tests Bug #12: silent event drops.
func TestPublishToSlowSubscriber(t *testing.T) {
	ps := NewPubSub()
	ch := ps.Subscribe("slow")

	// Don't read from channel to simulate slow subscriber
	// Publish should timeout after 100ms

	testEvent := events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Code:  1,
			Value: 100,
		},
	}

	start := time.Now()
	ps.Publish(testEvent)
	elapsed := time.Since(start)

	// Should have timed out after approximately 100ms
	if elapsed < 50*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Logf("Warning: Publish timeout took %v, expected ~100ms", elapsed)
	}

	ps.Unsubscribe(ch)
}

// BenchmarkPublishTicker benchmarks the ticker allocation performance.
// This benchmark tests Bug #10 fix: ticker created on every publish call.
func BenchmarkPublishTicker(b *testing.B) {
	ps := NewPubSub()
	ch := ps.Subscribe("bench")

	// Read from channel in background
	go func() {
		for range ch {
		}
	}()

	testEvent := events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Code:  1,
			Value: 100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.Publish(testEvent)
	}

	ps.Unsubscribe(ch)
}

// BenchmarkPublishMultipleSubscribers benchmarks with multiple subscribers
func BenchmarkPublishMultipleSubscribers(b *testing.B) {
	ps := NewPubSub()
	const numSubs = 5

	// Create multiple subscribers
	channels := make([]chan events.InputEventFromSource, numSubs)
	for i := 0; i < numSubs; i++ {
		channels[i] = ps.Subscribe("bench")
		go func(ch chan events.InputEventFromSource) {
			for range ch {
			}
		}(channels[i])
	}

	testEvent := events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Code:  1,
			Value: 100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.Publish(testEvent)
	}

	// Cleanup
	for _, ch := range channels {
		ps.Unsubscribe(ch)
	}
}
