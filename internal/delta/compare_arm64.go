//go:build arm64

package delta

import "unsafe"

// compareAndCopyBlocks compares src to dst in 64-byte blocks, copies src→dst,
// and sets mask[i] to non-zero for each block where any byte differs.
// Implemented in compare_arm64.s using NEON SIMD instructions.
//
//go:noescape
func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
