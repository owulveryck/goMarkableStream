package main

import (
	"syscall"
	"testing"
	"time"
)

func TestDetectSwipe(t *testing.T) {
	tests := []struct {
		name     string
		events   []InputEvent
		expected SwipeDirection
	}{
		{
			name: "Swipe Right",
			events: []InputEvent{
				{Time: syscall.Timeval{Sec: 1, Usec: 0}, Type: 3, Code: 54, Value: 100},
				{Time: syscall.Timeval{Sec: 1, Usec: 200000}, Type: 3, Code: 54, Value: 110},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 120},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 140},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 160},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 180},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 200},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 220},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 240},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 260},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 280},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 300},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 320},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 340},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 360},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 380},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 400},
			},
			expected: SwipeRight,
		},
		{
			name: "Swipe Left",
			events: []InputEvent{
				{Time: syscall.Timeval{Sec: 1, Usec: 0}, Type: 3, Code: 54, Value: 700},
				{Time: syscall.Timeval{Sec: 1, Usec: 200000}, Type: 3, Code: 54, Value: 680},
				{Time: syscall.Timeval{Sec: 1, Usec: 600000}, Type: 3, Code: 54, Value: 660},
			},
			expected: SwipeLeft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chanEvent := make(chan InputEvent)
			swipeC := make(chan SwipeDirection, 1) // Buffered to prevent blocking

			go detectSwipe(chanEvent, swipeC)

			for _, event := range tt.events {
				chanEvent <- event
			}
			close(chanEvent)

			select {
			case got := <-swipeC:
				if got != tt.expected {
					t.Errorf("detectSwipe() = %v, want %v", got, tt.expected)
				}
			case <-time.After(1 * time.Second):
				t.Error("Timeout: No swipe detected")
			}
		})
	}
}
