package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type zstdResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w zstdResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// zstdMiddleware applies zstd compression to HTTP responses.
func zstdMiddleware(next http.Handler, compressionLevel int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "zstd") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "zstd")

		encoder, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
		if err != nil {
			http.Error(w, "Failed to create zstd encoder", http.StatusInternalServerError)
			return
		}
		defer encoder.Close()

		compressedWriter := zstdResponseWriter{Writer: encoder, ResponseWriter: w}

		next.ServeHTTP(compressedWriter, r)
	})
}
