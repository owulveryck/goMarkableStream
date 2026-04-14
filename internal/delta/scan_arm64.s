#include "textflag.h"

// func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool
//
// Read-only NEON scan for arm64: compares dst and src in 128-byte blocks.
// Returns true (1) at the first differing block, false (0) if identical.
// No writes to memory — pure read-only scan with early exit.
//
// ABI0 stack layout:
//   dst:     0(FP)   8 bytes
//   src:     8(FP)   8 bytes
//   nblocks: 16(FP)  8 bytes
//   ret:     24(FP)  1 byte
//   Total args: 25 bytes
TEXT ·hasAnyChange(SB), NOSPLIT|NOFRAME, $0-25
	MOVD	dst+0(FP), R0
	MOVD	src+8(FP), R1
	MOVD	nblocks+16(FP), R5

	CBZ	R5, no_change

loop:
	// Load 128 bytes from src (two 64-byte halves)
	VLD1.P	64(R1), [V0.B16, V1.B16, V2.B16, V3.B16]
	VLD1.P	64(R1), [V24.B16, V25.B16, V26.B16, V27.B16]

	// Load 128 bytes from dst (two 64-byte halves)
	VLD1.P	64(R0), [V4.B16, V5.B16, V6.B16, V7.B16]
	VLD1.P	64(R0), [V28.B16, V29.B16, V30.B16, V31.B16]

	// XOR first half
	VEOR	V0.B16, V4.B16, V16.B16
	VEOR	V1.B16, V5.B16, V17.B16
	VEOR	V2.B16, V6.B16, V18.B16
	VEOR	V3.B16, V7.B16, V19.B16

	// XOR second half
	VEOR	V24.B16, V28.B16, V20.B16
	VEOR	V25.B16, V29.B16, V21.B16
	VEOR	V26.B16, V30.B16, V22.B16
	VEOR	V27.B16, V31.B16, V23.B16

	// OR-reduce all 8 XOR results
	VORR	V16.B16, V17.B16, V16.B16
	VORR	V18.B16, V19.B16, V18.B16
	VORR	V20.B16, V21.B16, V20.B16
	VORR	V22.B16, V23.B16, V22.B16
	VORR	V16.B16, V18.B16, V16.B16
	VORR	V20.B16, V22.B16, V20.B16
	VORR	V16.B16, V20.B16, V16.B16

	// Extract to scalar
	VMOV	V16.D[0], R6
	VMOV	V16.D[1], R7
	ORR	R6, R7, R8

	// Early exit if any difference found
	CBNZ	R8, found_change

	SUB	$1, R5, R5
	CBNZ	R5, loop

no_change:
	MOVD	$0, R9
	MOVB	R9, ret+24(FP)
	RET

found_change:
	MOVD	$1, R9
	MOVB	R9, ret+24(FP)
	RET
