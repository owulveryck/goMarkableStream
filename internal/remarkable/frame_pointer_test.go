package remarkable

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestCalculateFramePointerFrom(t *testing.T) {
	memory := make([]byte, 64)
	binary.LittleEndian.PutUint32(memory[8:12], 10)
	binary.LittleEndian.PutUint32(memory[16:20], 20)

	got, err := calculateFramePointerFrom(strings.NewReader(string(memory)), memoryRange{start: 0, end: 64}, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got != 8 {
		t.Fatalf("pointer = %d, want 8", got)
	}
}

func TestCalculateFramePointerFromRejectsInvalidHeader(t *testing.T) {
	memory := make([]byte, 64)
	if _, err := calculateFramePointerFrom(strings.NewReader(string(memory)), memoryRange{start: 0, end: 64}, 20, 20); err == nil {
		t.Fatal("expected invalid-header error")
	}
}

func TestCalculateFramePointerFromIsBounded(t *testing.T) {
	memory := make([]byte, 64)
	binary.LittleEndian.PutUint32(memory[8:12], minHeaderLength)
	_, err := calculateFramePointerFrom(strings.NewReader(string(memory)), memoryRange{start: 0, end: 64}, 20, 20)
	if err == nil || !strings.Contains(err.Error(), "exceeded 100") {
		t.Fatalf("error = %v, want bounded-iteration error", err)
	}
}
