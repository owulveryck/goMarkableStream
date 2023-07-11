//go:build !linux || !arm

package remarkable

import (
	"os"
	"syscall"
	"time"

	"context"
)

const (
	// Input event types
	evKey  = 1
	evRel  = 2
	evAbs  = 3
	evMsc  = 4
	evSw   = 5
	evLed  = 17
	evSnd  = 18
	evRep  = 20
	evFf   = 21
	evPwr  = 22
	evFfSt = 23
)

type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type EventScanner struct {
	pen, touch *os.File
	EventC     chan InputEvent
}

func NewEventScanner() *EventScanner {
	return &EventScanner{
		EventC: make(chan InputEvent),
	}
}

func (e *EventScanner) Start(ctx context.Context) {
	go func(ctx context.Context) {
		tick := time.NewTicker(4000 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				e.EventC <- InputEvent{}
			}
		}
	}(ctx)
}
