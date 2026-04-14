//go:build !arm64 && (!arm || nosimd)

package delta

import "unsafe"

// hasAnyChange scans dst and src for any difference without copying.
// Returns true at the first differing qword (early exit).
// Generic fallback for platforms without NEON SIMD.
func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool {
	totalBytes := nblocks * blockSize
	numQwords := totalBytes / 8
	for i := range numQwords {
		offset := i * 8
		if *(*uint64)(unsafe.Add(src, offset)) != *(*uint64)(unsafe.Add(dst, offset)) {
			return true
		}
	}
	return false
}
