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
	data := make([]byte, ScreenWidth*ScreenHeight)
	for i := 0; i < len(data); i++ {
		// Generate a random number between 1 and 10
		num := rand.Intn(10) + 1

		// Set the byte to zero with a probability of 80%
		if num <= 8 {
			data[i] = 0
		} else {
			// Otherwise, generate a random byte between 1 and 15
			data[i] = uint8(rand.Intn(15))
		}
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
		for i := 0; i < len(sample); i++ {
			if sample[i] != decoded[i] {
				t.Fatalf("at index %v, sample: %v, decoded: %v", i, sample[i-20:i+1], decoded[i-20:i+1])
			}
		}
		t.Errorf("Decoded data does not match the original data")
	}
}

func BenchmarkRleWriter(b *testing.B) {
	data := generateSampleData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		rw := rleWriter{sub: &buf}
		_, _ = rw.Write(data)
	}
}
