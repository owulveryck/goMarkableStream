package remarkable

import (
	"bytes"
	"testing"
)

func TestCopyVisibleRowsRemovesRightPadding(t *testing.T) {
	src := []byte{
		1, 2, 3, 9, 9,
		4, 5, 6, 8, 8,
	}
	dst := make([]byte, 6)

	if err := copyVisibleRows(dst, src, 3, 5, 2); err != nil {
		t.Fatalf("copyVisibleRows() error = %v", err)
	}
	if want := []byte{1, 2, 3, 4, 5, 6}; !bytes.Equal(dst, want) {
		t.Fatalf("copyVisibleRows() = %v, want %v", dst, want)
	}
}

func TestCopyVisibleRowsRejectsInvalidBuffers(t *testing.T) {
	tests := []struct {
		name            string
		dst             []byte
		src             []byte
		visibleRowBytes int
		storageRowBytes int
		height          int
	}{
		{name: "storage row too short", dst: make([]byte, 4), src: make([]byte, 4), visibleRowBytes: 2, storageRowBytes: 1, height: 2},
		{name: "destination size", dst: make([]byte, 3), src: make([]byte, 6), visibleRowBytes: 2, storageRowBytes: 3, height: 2},
		{name: "source size", dst: make([]byte, 4), src: make([]byte, 5), visibleRowBytes: 2, storageRowBytes: 3, height: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyVisibleRows(tt.dst, tt.src, tt.visibleRowBytes, tt.storageRowBytes, tt.height); err == nil {
				t.Fatal("copyVisibleRows() error = nil, want error")
			}
		})
	}
}
