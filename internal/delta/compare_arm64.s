#include "textflag.h"

// func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
//
// Compares src and dst in 128-byte blocks using NEON SIMD, copies src → dst
// only for changed blocks, and writes 1 to mask[i] if block i differs, 0 otherwise.
//
// Each 128-byte block is loaded as two 64-byte halves. All 128 bytes of src
// are held in registers (V0-V3, V24-V27) so the conditional copy needs no
// re-read. 8 XOR results are OR-reduced to a single scalar for the mask.
//
// Conditional copy optimization: unchanged blocks are NOT written back,
// saving memory bandwidth. This is safe because unchanged blocks already
// have identical data in dst.
//
// ABI0 stack layout:
//   dst:        0(FP)   8 bytes
//   src:        8(FP)   8 bytes
//   mask base: 16(FP)   8 bytes  (from slice header)
//   mask len:  24(FP)   8 bytes
//   mask cap:  32(FP)   8 bytes
//   nblocks:   40(FP)   8 bytes
//   Total args: 48 bytes
TEXT ·compareAndCopyBlocks(SB), NOSPLIT|NOFRAME, $0-48
	MOVD  dst+0(FP), R0        // dst = previous frame (read+write)
	MOVD  src+8(FP), R1        // src = current frame (read only)
	MOVD  mask+16(FP), R2      // mask base pointer
	MOVD  nblocks+40(FP), R5   // number of 128-byte blocks

	CBZ   R5, done

loop:
	// Load 128 bytes from src in two 64-byte halves
	VLD1.P  64(R1), [V0.B16, V1.B16, V2.B16, V3.B16]
	VLD1.P  64(R1), [V24.B16, V25.B16, V26.B16, V27.B16]

	// Load 128 bytes from dst in two 64-byte halves (post-increment)
	VLD1.P  64(R0), [V4.B16, V5.B16, V6.B16, V7.B16]
	VLD1.P  64(R0), [V28.B16, V29.B16, V30.B16, V31.B16]

	// XOR first half to find differences
	VEOR  V0.B16, V4.B16, V16.B16
	VEOR  V1.B16, V5.B16, V17.B16
	VEOR  V2.B16, V6.B16, V18.B16
	VEOR  V3.B16, V7.B16, V19.B16

	// XOR second half
	VEOR  V24.B16, V28.B16, V20.B16
	VEOR  V25.B16, V29.B16, V21.B16
	VEOR  V26.B16, V30.B16, V22.B16
	VEOR  V27.B16, V31.B16, V23.B16

	// OR-reduce all 8 XOR results into V16
	VORR  V16.B16, V17.B16, V16.B16
	VORR  V18.B16, V19.B16, V18.B16
	VORR  V20.B16, V21.B16, V20.B16
	VORR  V22.B16, V23.B16, V22.B16
	VORR  V16.B16, V18.B16, V16.B16
	VORR  V20.B16, V22.B16, V20.B16
	VORR  V16.B16, V20.B16, V16.B16

	// Extract 128-bit result to two 64-bit scalar registers
	VMOV  V16.D[0], R6
	VMOV  V16.D[1], R7
	ORR   R6, R7, R8

	// Set mask: 1 if any byte differs, 0 if identical
	CMP   $0, R8
	CSET  NE, R9
	MOVB  R9, (R2)
	ADD   $1, R2, R2

	// Conditional copy: only write back changed blocks.
	// R0 already points past the block from VLD1.P post-increment.
	CBZ   R8, skip_copy

	// Changed block: rewind dst by 128, store both halves, advance dst.
	SUB   $128, R0, R0
	VST1.P  [V0.B16, V1.B16, V2.B16, V3.B16], 64(R0)
	VST1.P  [V24.B16, V25.B16, V26.B16, V27.B16], 64(R0)

skip_copy:
	SUB   $1, R5, R5
	CBNZ  R5, loop

done:
	RET
