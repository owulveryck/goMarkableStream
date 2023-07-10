package remarkable

import (
	"log"
	"os"
	"syscall"
	"unsafe"

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
	pen, err := os.OpenFile("/dev/input/event1", os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	touch, err := os.OpenFile("/dev/input/event2", os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	return &EventScanner{
		pen:    pen,
		touch:  touch,
		EventC: make(chan InputEvent),
	}
}

func (e *EventScanner) Close() {
	close(e.EventC)
	e.pen.Close()
	e.touch.Close()
}

func (e *EventScanner) Start(ctx context.Context) {
	// Start a goroutine to read events and send them on the channel
	go func() {
		for {
			var ev InputEvent
			_, err := e.pen.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
			if err != nil {
				log.Println(err)
				return
			}
			select {
			case <-ctx.Done():
				return
			case e.EventC <- ev:
			}
		}
	}()
	go func() {
		for {
			var ev InputEvent
			_, err := e.touch.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
			if err != nil {
				log.Println(err)
				return
			}
			select {
			case <-ctx.Done():
				return
			case e.EventC <- ev:
			}
		}
	}()
}
