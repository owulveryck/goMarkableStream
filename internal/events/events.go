package events

import "syscall"

const (
	// Input event types
	// see https://www.kernel.org/doc/Documentation/input/event-codes.txt
	// EvSyn is used as markers to separate events. Events may be separated in time or in
	// space, such as with the multitouch protocol.
	EvSyn = 0
	// EvKey is used to describe state changes of keyboards, buttons, or other key-like
	// devices.
	EvKey = 1
	// EvRel is used to describe relative axis value changes, e.g., moving the mouse
	// 5 units to the left.
	EvRel = 2
	// EvAbs is used to describe absolute axis value changes, e.g., describing the
	// coordinates of a touch on a touchscreen.
	EvAbs = 3
	// EvMsc is used to describe miscellaneous input data that do not fit into other types.
	EvMsc = 4
	// EvSw is used to describe binary state input switches.
	EvSw = 5
	// EvLed is used to turn LEDs on devices on and off.
	EvLed = 17
	// EvSnd is used to output sound to devices.
	EvSnd = 18
	// EvRep is used for autorepeating devices.
	EvRep = 20
	// EvFf is used to send force feedback commands to an input device.
	EvFf = 21
	// EvPwr is a special type for power button and switch input.
	EvPwr = 22
	// EvFfStatus is used to receive force feedback device status.
	EvFfStatus = 23
)

const (
	Pen   int = 1
	Touch int = 2
)

// InputEvent from the reMarkable
type InputEvent struct {
	Time syscall.Timeval `json:"-"`
	Type uint16
	// Code holds the position of the mouse/touch
	// In case of an EV_ABS event,
	// 1 -> X-axis (vertical movement) | 0 < Value < 15725 if mouse
	// 0 -> Y-axis (horizontal movement) | 0 < Value < 20966 if mouse
	Code  uint16
	Value int32
}

// InputEventFromSrouce add the source origin
type InputEventFromSource struct {
	Source int
	InputEvent
}
