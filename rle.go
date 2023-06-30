package main

import "log"

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

func encodeRLE(data []uint8) []uint8 {
	var encoded []uint8

	length := len(data)
	if length == 0 {
		return encoded
	}

	current := data[0]
	var count uint8 = 1

	for i := 1; i < length; i++ {
		if data[i] == current && count < 16 {
			count++
		} else {
			encoded = append(encoded, pack(count, current))
			current = data[i]
			count = 1
		}
	}

	return append(encoded, pack(count, current))
}
