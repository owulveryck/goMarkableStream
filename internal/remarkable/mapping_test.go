package remarkable

import (
	"strings"
	"testing"
)

func TestParseFramebufferMemoryRange(t *testing.T) {
	maps := strings.Join([]string{
		"1000-2000 rw-s 00000000 00:06 1 /dev/dri/card0",
		"2000-3000 rw-p 00000000 00:00 0",
		"4000-5000 rw-s 00000000 00:06 2 /dev/dri/card0",
		"5000-9000 rw-p 00000000 00:00 0",
	}, "\n")

	got, err := parseFramebufferMemoryRange(strings.NewReader(maps))
	if err != nil {
		t.Fatal(err)
	}
	if got.start != 0x5000 || got.end != 0x9000 {
		t.Fatalf("range = %#x-%#x, want 0x5000-0x9000", got.start, got.end)
	}
}

func TestParseFramebufferMemoryRangeRejectsMissingAdjacentMapping(t *testing.T) {
	maps := "1000-2000 rw-s 00000000 00:06 1 /dev/dri/card0\n3000-4000 rw-p 00000000 00:00 0\n"
	if _, err := parseFramebufferMemoryRange(strings.NewReader(maps)); err == nil {
		t.Fatal("expected missing adjacent mapping error")
	}
}
