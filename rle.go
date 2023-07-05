package main

import (
	"bufio"
	"log"
)

func pack(value1, value2 uint8) uint8 {
	// Ensure that the values are within the valid range (0-15)
	if value1 > 16 || value2 > 255 {
		log.Fatalf("invalid value; count=%v, value=%v", value1, value2)
	}

	value1 = value1 - 1
	// Shift the first value by 4 bits and OR it with the second value
	encodedValue := (value1 << 4) | value2
	return encodedValue
}

type rleWriter struct {
	sub *bufio.Writer
}

func (rlewriter *rleWriter) Write(data []byte) (n int, err error) {
	length := len(data)
	if length == 0 {
		return 0, nil
	}

	current := data[0]
	var count uint8 = 1
	var global int

	for i := 1; i < length; i++ {
		if data[i] == current && count < 16 {
			count++
		} else {
			err = rlewriter.sub.WriteByte(pack(count, current))
			if err != nil {
				return global, err
			}
			global++
			current = data[i]
			count = 1
		}
	}

	err = rlewriter.sub.WriteByte(pack(count, current))
	if err != nil {
		return global, err
	}
	global++
	return global, nil
}
