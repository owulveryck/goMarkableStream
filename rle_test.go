package main

import (
	"bytes"
	"math/rand"
	"testing"
)

func decode(data []byte) []byte {
	decoded := make([]byte, 0, len(data)*15)
	for _, value := range data {
		count := value >> 4
		item := value & 0x0F
		for i := uint8(0); i < count+1; i++ {
			decoded = append(decoded, uint8(item))
		}
	}
	return decoded
}

func generateSampleData() []byte {
	//data := make([]byte, ScreenWidth*ScreenHeight)
	data := make([]byte, ScreenHeight*ScreenWidth)
	for i := 0; i < len(data); i++ {
		data[i] = uint8(rand.Intn(15)) // random value between 0 and 15
	}
	return data
}

func TestRleWriter(t *testing.T) {
	sample := generateSampleData()

	var buf bytes.Buffer
	rw := rleWriter{sub: &buf}

	_, err := rw.Write(sample)
	if err != nil {
		t.Fatal(err)
	}

	encoded := buf.Bytes()

	decoded := decode(encoded)

	if !bytes.Equal(decoded, sample) {
		t.Errorf("Decoded data does not match the original data")
	}
}
