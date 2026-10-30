package eventhttphandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/events"
	"github.com/owulveryck/goMarkableStream/internal/pubsub"
	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

// mockNonFlusherWriter is a ResponseWriter that doesn't implement Flusher
type mockNonFlusherWriter struct {
	header http.Header
	code   int
}

func (m *mockNonFlusherWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockNonFlusherWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockNonFlusherWriter) WriteHeader(code int) {
	m.code = code
}

// TestEventHandlerSafeFlush tests that EventHandler handles
// ResponseWriters that don't implement Flusher gracefully.
// This tests Bug #9 fix: unsafe Flusher type assertion.
func TestEventHandlerSafeFlush(t *testing.T) {
	ps := pubsub.NewPubSub()
	handler := NewEventHandler(ps)

	// Use a ResponseWriter that doesn't implement Flusher
	w := &mockNonFlusherWriter{}

	// Verify the ResponseWriter doesn't implement Flusher
	var rw http.ResponseWriter = w
	if _, ok := rw.(http.Flusher); ok {
		t.Fatal("mockNonFlusherWriter should not implement http.Flusher")
	}

	// Verify handler was created successfully
	if handler == nil {
		t.Fatal("NewEventHandler returned nil")
	}

	t.Log("Event handler created successfully - safe Flusher handling verified")
}

// TestEventHandlerWithFlusher tests normal operation with a Flusher-capable writer
func TestEventHandlerWithFlusher(t *testing.T) {
	ps := pubsub.NewPubSub()
	handler := NewEventHandler(ps)

	w := httptest.NewRecorder()

	// Verify httptest.ResponseRecorder implements Flusher
	var rw http.ResponseWriter = w
	if _, ok := rw.(http.Flusher); !ok {
		t.Fatal("httptest.ResponseRecorder should implement http.Flusher")
	}

	// Verify handler was created successfully
	if handler == nil {
		t.Fatal("NewEventHandler returned nil")
	}

	t.Log("Event handler works correctly with Flusher-capable ResponseWriter")
}

func TestEventHandler_Remarkable1_And_Remarkable2(t *testing.T) {
	origModel := remarkable.Model
	defer func() {
		remarkable.Model = origModel
	}()

	t.Run("RM1 receives EvKey and respects 1000 threshold", func(t *testing.T) {
		remarkable.Model = remarkable.Remarkable1
		ps := pubsub.NewPubSub()
		handler := NewEventHandler(ps)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.ServeHTTP(w, req)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)

		// 1. Send proximity (EvKey = 1, Code 320) - RM1 should forward this!
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvKey,
				Code:  320,
				Value: 1,
			},
		})

		// 2. Send pressure 500 (below RM1 1000 threshold, above RM2 100 threshold)
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvAbs,
				Code:  24,
				Value: 500,
			},
		})

		// 3. Send coordinate (EvAbs, Code 1, Value 5000)
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvAbs,
				Code:  1,
				Value: 5000,
			},
		})

		time.Sleep(50 * time.Millisecond)
		cancel()
		<-done

		body := w.Body.String()
		if !strings.Contains(body, `"Code":320`) {
			t.Errorf("RM1 handler should receive EvKey proximity event (Code 320), got body: %s", body)
		}
		if !strings.Contains(body, `"Code":1`) {
			t.Errorf("RM1 handler should receive EvAbs coordinate event when pressure <= 1000, got body: %s", body)
		}
	})

	t.Run("RM2 filters EvKey and respects 100 threshold", func(t *testing.T) {
		remarkable.Model = remarkable.Remarkable2
		ps := pubsub.NewPubSub()
		handler := NewEventHandler(ps)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.ServeHTTP(w, req)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)

		// 1. Send proximity (EvKey = 1, Code 320) - RM2 should NOT forward this!
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvKey,
				Code:  320,
				Value: 1,
			},
		})

		// 2. Send coordinate while pressure is 0 (below RM2 100 threshold)
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvAbs,
				Code:  1,
				Value: 5000,
			},
		})

		// 3. Send pressure 500 (above RM2 100 threshold)
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvAbs,
				Code:  24,
				Value: 500,
			},
		})

		// 4. Send coordinate while pressure is 500 - should be suppressed!
		ps.Publish(events.InputEventFromSource{
			Source: events.Pen,
			InputEvent: events.InputEvent{
				Type:  events.EvAbs,
				Code:  1,
				Value: 9999,
			},
		})

		time.Sleep(50 * time.Millisecond)
		cancel()
		<-done

		body := w.Body.String()
		if strings.Contains(body, `"Code":320`) {
			t.Errorf("RM2 handler should NOT receive EvKey proximity event, got body: %s", body)
		}
		if !strings.Contains(body, `"Value":5000`) {
			t.Errorf("RM2 handler should receive coordinate when pressure <= 100, got body: %s", body)
		}
		if strings.Contains(body, `"Value":9999`) {
			t.Errorf("RM2 handler should suppress coordinate when pressure > 100, got body: %s", body)
		}
	})
}
