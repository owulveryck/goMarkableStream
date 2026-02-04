package stream

import (
	"testing"

	"github.com/owulveryck/goMarkableStream/internal/delta"
)

func BenchmarkFetchAndSendDelta(b *testing.B) {
	// Setup: Create a large enough mockReaderAt to test performance
	width, height := 2872, 2404                  // Example size; adjust based on your needs
	mockReader := NewMockReaderAt(width, height) // Using the mock from the previous example

	handler := StreamHandler{
		file:         mockReader,
		pointerAddr:  0,
		deltaEncoder: delta.NewEncoder(0.30),
	}

	mockWriter := NewMockResponseWriter()

	data := make([]byte, width*height*4) // BGRA = 4 bytes per pixel

	b.ResetTimer() // Start timing here
	for i := 0; i < b.N; i++ {
		handler.fetchAndSendDelta(mockWriter, data)
	}
}
