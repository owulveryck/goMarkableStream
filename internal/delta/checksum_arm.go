//go:build arm && !nosimd

package delta

import "unsafe"

// checksumChanged computes a 128-bit XOR-fold checksum of buf (single-stream,
// 10.5MB read) and compares it with the stored checksum at prevChecksum.
// Returns true if the checksum differs (frame changed), updating prevChecksum
// in-place. Returns false if identical (frame unchanged).
//
// This is ~2x faster than hasAnyChange for unchanged frames because it reads
// only one buffer instead of two (halving memory bandwidth on Cortex-A7 where
// bandwidth is the bottleneck).
//
//go:noescape
func checksumChanged(buf unsafe.Pointer, nblocks int, prevChecksum unsafe.Pointer) bool
