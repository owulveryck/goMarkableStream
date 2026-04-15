package delta

import "unsafe"

// checksumChanged computes a 128-bit XOR-fold checksum of buf and compares
// it with the stored checksum at prevChecksum. Returns true if changed
// (updating prevChecksum), false if unchanged.
//
// Generic fallback implementation using uint64 operations.
func checksumChanged(buf unsafe.Pointer, nblocks int, prevChecksum unsafe.Pointer) bool {
	var accLo, accHi uint64

	for b := range nblocks {
		base := b * blockSize
		for i := 0; i < blockSize; i += 16 {
			off := base + i
			accLo ^= *(*uint64)(unsafe.Add(buf, off))
			accHi ^= *(*uint64)(unsafe.Add(buf, off+8))
		}
		// Rotate left by 8 bits (1 byte) for position dependence
		newLo := (accLo << 8) | (accHi >> 56)
		newHi := (accHi << 8) | (accLo >> 56)
		accLo = newLo
		accHi = newHi
	}

	prev := (*[2]uint64)(prevChecksum)
	if accLo != prev[0] || accHi != prev[1] {
		prev[0] = accLo
		prev[1] = accHi
		return true
	}
	return false
}
