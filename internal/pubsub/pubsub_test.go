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
// With non-blocking send, events are dropped immediately if channel is full.
func TestPublishToSlowSubscriber(t *testing.T) {
	ps := NewPubSub()
	ch := ps.Subscribe("slow")

	// Don't read from channel to simulate slow subscriber
	// Publish should complete immediately (non-blocking)

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

	// Should complete almost immediately (< 10ms) with non-blocking send
	if elapsed > 10*time.Millisecond {
		t.Errorf("Publish took %v, expected < 10ms with non-blocking send", elapsed)
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

// TestPublishNonBlocking tests that Publish doesn't block with buffered channels
func TestPublishNonBlocking(t *testing.T) {
	ps := NewPubSub()
	ch := ps.Subscribe("test")

	// Fill buffer (100 events)
	for i := 0; i < 100; i++ {
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Code:  uint16(i),
				Value: int32(i),
			},
		})
	}

	// Publishing one more event should not block (will be dropped)
	start := time.Now()
	ps.Publish(events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Code:  101,
			Value: 101,
		},
	})
	elapsed := time.Since(start)

	if elapsed > 10*time.Millisecond {
		t.Errorf("Publish blocked for %v, expected < 10ms", elapsed)
	}

	ps.Unsubscribe(ch)
}

// TestEventFiltering tests that event filters work correctly
func TestEventFiltering(t *testing.T) {
	ps := NewPubSub()

	// Subscribe with Pen source filter
	penSource := events.Pen
	chPen := ps.SubscribeWithFilter("pen-only", EventFilter{Source: &penSource})

	// Subscribe with Touch source filter
	touchSource := events.Touch
	chTouch := ps.SubscribeWithFilter("touch-only", EventFilter{Source: &touchSource})

	// Subscribe with EvAbs type filter
	absType := uint16(events.EvAbs)
	chAbs := ps.SubscribeWithFilter("abs-only", EventFilter{Type: &absType})

	// Subscribe with Pen + EvAbs filter
	chPenAbs := ps.SubscribeWithFilter("pen-abs", EventFilter{
		Source: &penSource,
		Type:   &absType,
	})

	// Publish touch event
	ps.Publish(events.InputEventFromSource{
		Source: events.Touch,
		InputEvent: events.InputEvent{
			Type:  events.EvAbs,
			Code:  1,
			Value: 100,
		},
	})

	// Publish pen event (EvKey type)
	ps.Publish(events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Type:  events.EvKey,
			Code:  2,
			Value: 200,
		},
	})

	// Publish pen event (EvAbs type)
	ps.Publish(events.InputEventFromSource{
		Source: events.Pen,
		InputEvent: events.InputEvent{
			Type:  events.EvAbs,
			Code:  3,
			Value: 300,
		},
	})

	// Check Pen filter - should receive 2 pen events
	receivedPen := 0
	for receivedPen < 2 {
		select {
		case ev := <-chPen:
			if ev.Source != events.Pen {
				t.Errorf("Pen filter received non-pen event: source=%d", ev.Source)
			}
			receivedPen++
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Pen filter timeout, received %d/2 events", receivedPen)
		}
	}

	// Check Touch filter - should receive 1 touch event
	select {
	case ev := <-chTouch:
		if ev.Source != events.Touch {
			t.Errorf("Touch filter received non-touch event: source=%d", ev.Source)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Touch filter timeout")
	}

	// Verify no extra events
	select {
	case ev := <-chTouch:
		t.Errorf("Touch filter received unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// Good - no extra events
	}

	// Check EvAbs filter - should receive 2 EvAbs events (touch + pen)
	receivedAbs := 0
	for receivedAbs < 2 {
		select {
		case ev := <-chAbs:
			if ev.Type != events.EvAbs {
				t.Errorf("EvAbs filter received wrong type: type=%d", ev.Type)
			}
			receivedAbs++
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("EvAbs filter timeout, received %d/2 events", receivedAbs)
		}
	}

	// Check Pen+EvAbs filter - should receive only 1 event (pen with EvAbs)
	select {
	case ev := <-chPenAbs:
		if ev.Source != events.Pen || ev.Type != events.EvAbs {
			t.Errorf("Pen+EvAbs filter received wrong event: source=%d, type=%d", ev.Source, ev.Type)
		}
		if ev.Code != 3 {
			t.Errorf("Pen+EvAbs filter received wrong event: code=%d, expected 3", ev.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Pen+EvAbs filter timeout")
	}

	// Verify no extra events
	select {
	case ev := <-chPenAbs:
		t.Errorf("Pen+EvAbs filter received unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// Good - no extra events
	}

	ps.Unsubscribe(chPen)
	ps.Unsubscribe(chTouch)
	ps.Unsubscribe(chAbs)
	ps.Unsubscribe(chPenAbs)
}
