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
				{Time: syscall.Timeval{Sec: 1, Usec: 2000}, Type: 3, Code: 54, Value: 110},
				{Time: syscall.Timeval{Sec: 1, Usec: 3000}, Type: 3, Code: 54, Value: 120},
				{Time: syscall.Timeval{Sec: 1, Usec: 4000}, Type: 3, Code: 54, Value: 140},
				{Time: syscall.Timeval{Sec: 1, Usec: 5000}, Type: 3, Code: 54, Value: 160},
				{Time: syscall.Timeval{Sec: 1, Usec: 6000}, Type: 3, Code: 54, Value: 180},
				{Time: syscall.Timeval{Sec: 1, Usec: 7000}, Type: 3, Code: 54, Value: 200},
				{Time: syscall.Timeval{Sec: 1, Usec: 8000}, Type: 3, Code: 54, Value: 220},
				{Time: syscall.Timeval{Sec: 1, Usec: 9000}, Type: 3, Code: 54, Value: 240},
				{Time: syscall.Timeval{Sec: 1, Usec: 10000}, Type: 3, Code: 54, Value: 260},
				{Time: syscall.Timeval{Sec: 1, Usec: 11000}, Type: 3, Code: 54, Value: 280},
				{Time: syscall.Timeval{Sec: 1, Usec: 12000}, Type: 3, Code: 54, Value: 300},
				{Time: syscall.Timeval{Sec: 1, Usec: 13000}, Type: 3, Code: 54, Value: 320},
				{Time: syscall.Timeval{Sec: 1, Usec: 14000}, Type: 3, Code: 54, Value: 340},
				{Time: syscall.Timeval{Sec: 1, Usec: 15000}, Type: 3, Code: 54, Value: 360},
				{Time: syscall.Timeval{Sec: 1, Usec: 16000}, Type: 3, Code: 54, Value: 380},
				{Time: syscall.Timeval{Sec: 1, Usec: 17000}, Type: 3, Code: 54, Value: 400},
			},
			expected: SwipeRight,
		},
		{
			name: "Swipe Left",
			events: []InputEvent{
				{Time: syscall.Timeval{Sec: 1, Usec: 0}, Type: 3, Code: 54, Value: 400},
				{Time: syscall.Timeval{Sec: 1, Usec: 2000}, Type: 3, Code: 54, Value: 380},
				{Time: syscall.Timeval{Sec: 1, Usec: 3000}, Type: 3, Code: 54, Value: 360},
				{Time: syscall.Timeval{Sec: 1, Usec: 4000}, Type: 3, Code: 54, Value: 340},
				{Time: syscall.Timeval{Sec: 1, Usec: 5000}, Type: 3, Code: 54, Value: 320},
				{Time: syscall.Timeval{Sec: 1, Usec: 6000}, Type: 3, Code: 54, Value: 300},
				{Time: syscall.Timeval{Sec: 1, Usec: 7000}, Type: 3, Code: 54, Value: 280},
				{Time: syscall.Timeval{Sec: 1, Usec: 8000}, Type: 3, Code: 54, Value: 260},
				{Time: syscall.Timeval{Sec: 1, Usec: 9000}, Type: 3, Code: 54, Value: 240},
				{Time: syscall.Timeval{Sec: 1, Usec: 10000}, Type: 3, Code: 54, Value: 220},
				{Time: syscall.Timeval{Sec: 1, Usec: 11000}, Type: 3, Code: 54, Value: 200},
				{Time: syscall.Timeval{Sec: 1, Usec: 12000}, Type: 3, Code: 54, Value: 180},
				{Time: syscall.Timeval{Sec: 1, Usec: 13000}, Type: 3, Code: 54, Value: 160},
				{Time: syscall.Timeval{Sec: 1, Usec: 14000}, Type: 3, Code: 54, Value: 140},
				{Time: syscall.Timeval{Sec: 1, Usec: 15000}, Type: 3, Code: 54, Value: 120},
				{Time: syscall.Timeval{Sec: 1, Usec: 16000}, Type: 3, Code: 54, Value: 110},
				{Time: syscall.Timeval{Sec: 1, Usec: 17000}, Type: 3, Code: 54, Value: 100},
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
