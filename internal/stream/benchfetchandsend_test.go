package stream

import "testing"

func BenchmarkFetchAndSend(b *testing.B) {
	// Setup: Create a large enough mockReaderAt to test performance
	width, height := 1872	, 1404                    // Example size; adjust based on your needs
	mockReader := NewMockReaderAt(width, height) // Using the mock from the previous example

	handler := StreamHandler{
		file:        mockReader,
		pointerAddr: 0,
	}

	mockWriter := NewMockResponseWriter()

	data := make([]byte, width*height) // Adjust based on your payload size

	b.ResetTimer() // Start timing here
	for i := 0; i < b.N; i++ {
		handler.fetchAndSend(mockWriter, data)
	}
}
