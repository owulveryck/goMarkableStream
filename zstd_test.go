package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test handler to wrap in the middleware
func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Hello, World!"))
	if err != nil {
		panic(err)
	}
}

func TestZstdMiddleware_NoCompression(t *testing.T) {
	handler := zstdMiddleware(http.HandlerFunc(testHandler), 3)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.Header.Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding, got %s", res.Header.Get("Content-Encoding"))
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "Hello, World!" {
		t.Errorf("unexpected body: got %s, want %s", string(body), "Hello, World!")
	}
}

func TestZstdMiddleware_WithCompression(t *testing.T) {
	handler := zstdMiddleware(http.HandlerFunc(testHandler), 3)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "zstd")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.Header.Get("Content-Encoding") != "zstd" {
		t.Errorf("expected Content-Encoding to be zstd, got %s", res.Header.Get("Content-Encoding"))
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) == "Hello, World!" {
		t.Errorf("response should be compressed, found plaintext in body")
	}
}
