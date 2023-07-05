package main

import (
	"io"
	"sync"
)

var encodedPool = sync.Pool{
	New: func() interface{} {
		return make([]uint8, 0, 1872*1404/2) // Adjust the initial capacity as needed
	},
}

func pack(value1, value2 uint8) uint8 {
	// Shift the first value by 4 bits and OR it with the second value
	encodedValue := (value1 << 4) | value2
	return encodedValue
}

type rleWriter struct {
	sub io.Writer
}

func (rlewriter *rleWriter) Write(data []byte) (n int, err error) {
	length := len(data)
	if length == 0 {
		return 0, nil
	}
	encoded := encodedPool.Get().([]uint8)[:0] // Borrow a slice from the pool
	defer encodedPool.Put(encoded)

	current := data[0]
	var count uint8 = 0

	for _, datum := range data {
		if count == 15 {
			encoded = append(encoded, pack(count, current))
			count = 0
			continue
		}
		if datum == current {
			count++
		} else {
			encoded = append(encoded, pack(count, current))
			current = datum
			count = 0
		}
	}

	encoded = append(encoded, pack(count, current))
	return rlewriter.sub.Write(encoded)
}
