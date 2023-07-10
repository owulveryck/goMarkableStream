package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
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

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

func main() {
	// Open the input device file
	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create a channel to send events
	eventCh := make(chan inputEvent)

	// Start a goroutine to read events and send them on the channel
	go func() {
		for {
			var ev inputEvent
			_, err := file.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
			if err != nil {
				log.Fatal(err)
			}
			eventCh <- ev
		}
	}()

	// Listen for events on the channel
	for ev := range eventCh {
		fmt.Printf("Event: %+v\n", ev)
	}
}
