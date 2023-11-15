package events

import "syscall"

const (
	// Input event types
	// see https://www.kernel.org/doc/Documentation/input/event-codes.txt
	EvKey = 1
	EvRel = 2
	// We got EV_ABS from the reMarkable whatever the imput device is
	EvAbs  = 3
	EvMsc  = 4
	EvSw   = 5
	EvLed  = 17
	EvSnd  = 18
	EvRep  = 20
	EvFf   = 21
	EvPwr  = 22
	EvFfSt = 23
)

// InputEvent from the reMarkable
type InputEvent struct {
	Time syscall.Timeval
	Type uint16
	// Code holds the position of the mouse/touch
	// In case of an EV_ABS event,
	// 1 -> X-axis (vertical movement) | 0 < Value < 15725 if mouse
	// 0 -> Y-axis (horizontal movement) | 0 < Value < 20966 if mouse
	Code  uint16
	Value int32
}

type InputEventFromSource struct {
	Source string
	InputEvent
}
