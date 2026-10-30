//go:build arm && !nosimd

package delta

import "unsafe"

// copyBlocksNEON copies nblocks*128 bytes from src to dst using 32-byte burst
// VLD1.8 / VST1.8 NEON instructions.
//
//go:noescape
func copyBlocksNEON(dst, src unsafe.Pointer, nblocks int)

// CopyFrame copies min(len(dst), len(src)) bytes from src to dst.
// On ARM32 with NEON, uses 32-byte burst VLD1.8 loads rather than Go's scalar
// runtime.memmove, which is ~4-8x faster when reading Write-Combine mmap'd
// /dev/fb0 memory on the reMarkable 1 (Cortex-A9).
func CopyFrame(dst, src []byte) int {
	n := min(len(dst), len(src))
	if n == 0 {
		return 0
	}
	nblocks := n / blockSize
	if nblocks > 0 {
		copyBlocksNEON(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), nblocks)
	}
	rem := nblocks * blockSize
	if rem < n {
		copy(dst[rem:n], src[rem:n])
	}
	return n
}
