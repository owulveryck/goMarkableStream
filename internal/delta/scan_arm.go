//go:build arm && !nosimd

package delta

import "unsafe"

// hasAnyChange scans dst and src for any difference without copying or
// writing a mask. Returns true at the first differing block (early exit).
// This is much cheaper than compareAndCopyBlocks for unchanged frames
// because it only reads memory (no stores) and can bail early.
// Implemented in scan_arm.s using ARM NEON SIMD instructions.
//
//go:noescape
func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool
