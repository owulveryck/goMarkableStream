package stream

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
)

// MockReaderAt implements the io.ReaderAt interface.
type MockReaderAt struct {
	width  int
	height int
	data   []byte
}

// NewMockReaderAt creates a new MockReaderAt with the specified dimensions and initializes its data.
func NewMockReaderAt(width, height int) *MockReaderAt {
	size := width * height
	data := make([]byte, size)

	// Fill the slice with values where 70% are 0s and the rest are between 1 and 17.
	for i := 0; i < size; i++ {
		if rand.Float64() > 0.7 {
			data[i] = byte(rand.Intn(17) + 1)
		}
	}

	return &MockReaderAt{
		width:  width,
		height: height,
		data:   data,
	}
}

// ReadAt reads len(p) bytes into p starting at offset off in the mock data.
// It implements the io.ReaderAt interface.
func (m *MockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}

	n = copy(p, m.data[off:])
	if n < len(p) {
		err = io.EOF
	}

	return n, err
}

// mockResponseWriter simulates an http.ResponseWriter for benchmarking purposes.
type mockResponseWriter struct {
	headerMap http.Header
	body      *bytes.Buffer
}

func NewMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headerMap: make(http.Header),
		body:      new(bytes.Buffer),
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headerMap
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	return m.body.Write(data)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	// For benchmarking, we might not need to simulate the status code.
}

func (m *mockResponseWriter) Flush() {
	// Simulate flushing the buffer if implementing http.Flusher.
}
