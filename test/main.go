package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

type inputEvent struct {
	Type  uint16
	Code  uint16
	Value int32
}

func main() {
	event := inputEvent{Type: 1, Code: 2, Value: 3}

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, event)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	fmt.Printf("% x", buf.Bytes())

	// Now buf.Bytes() contains the binary representation of the structure
	// You can send this data to a JavaScript environment
}
