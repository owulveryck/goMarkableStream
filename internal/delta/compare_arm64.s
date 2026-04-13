#include "textflag.h"

// func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
//
// Compares src and dst in 64-byte blocks using NEON SIMD, copies src → dst,
// and writes 1 to mask[i] if block i differs, 0 otherwise.
//
// Uses 4 × 128-bit NEON registers per iteration = 64 bytes.
// XOR detects differences, ORR combines, VMOV extracts to scalar.
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
	MOVD  nblocks+40(FP), R5   // number of 64-byte blocks

	CBZ   R5, done

loop:
	// Load 64 bytes from src (current frame) with post-increment
	VLD1.P  64(R1), [V0.B16, V1.B16, V2.B16, V3.B16]

	// Load 64 bytes from dst (previous frame) — no post-increment yet
	VLD1    (R0), [V4.B16, V5.B16, V6.B16, V7.B16]

	// Copy: store src data to dst with post-increment
	VST1.P  [V0.B16, V1.B16, V2.B16, V3.B16], 64(R0)

	// XOR to find byte-level differences
	VEOR  V0.B16, V4.B16, V16.B16
	VEOR  V1.B16, V5.B16, V17.B16
	VEOR  V2.B16, V6.B16, V18.B16
	VEOR  V3.B16, V7.B16, V19.B16

	// OR all difference vectors together
	VORR  V16.B16, V17.B16, V20.B16
	VORR  V18.B16, V19.B16, V21.B16
	VORR  V20.B16, V21.B16, V22.B16

	// Extract 128-bit result to two 64-bit scalar registers
	VMOV  V22.D[0], R6
	VMOV  V22.D[1], R7
	ORR   R6, R7, R8

	// Set mask: 1 if any byte differs, 0 if identical
	CMP   $0, R8
	CSET  NE, R9
	MOVB  R9, (R2)
	ADD   $1, R2, R2

	SUB   $1, R5, R5
	CBNZ  R5, loop

done:
	RET
