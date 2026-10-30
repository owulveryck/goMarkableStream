//go:build arm && !nosimd

package delta

import "unsafe"

// checksumChanged computes a 128-bit XOR-fold checksum of buf (nblocks * 128B)
// using ARMv7 NEON 32-byte burst loads and compares it with prevChecksum.
// Returns true if changed (updating prevChecksum), false if unchanged.
//
//go:noescape
func checksumChanged(buf unsafe.Pointer, nblocks int, prevChecksum unsafe.Pointer) bool
