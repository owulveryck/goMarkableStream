//go:build arm64

package delta

import "unsafe"

// hasAnyChange scans dst and src for any difference without copying or
// writing a mask. Returns true at the first differing block (early exit).
// On arm64, xxhash is fast enough that we use the hash-based early exit
// instead, but this function is available as a fallback.
// Implemented in scan_arm64.s using NEON SIMD instructions.
//
//go:noescape
func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool
