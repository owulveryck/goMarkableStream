//go:build arm

package delta

// useHashEarlyExit controls whether xxhash.Sum64 is used for frame
// change detection. On ARM32, xxhash has no assembly implementation —
// 64-bit multiplications compile to UMULL+MLA sequences, making the
// hash ~6x slower than the NEON SIMD compare+copy pass it tries to
// skip. We disable it and rely on the mask result from
// compareAndCopyBlocks instead.
const useHashEarlyExit = false
