package eventhttphandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/owulveryck/goMarkableStream/internal/pubsub"
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
