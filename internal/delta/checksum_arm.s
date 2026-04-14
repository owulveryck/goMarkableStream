//go:build arm && !nosimd

#include "textflag.h"

// func checksumChanged(buf unsafe.Pointer, nblocks int, prevChecksum unsafe.Pointer) bool
//
// Single-buffer NEON XOR-fold checksum for idle frame detection on ARM32.
// Reads only the current buffer (10.5MB) instead of both src+dst (21MB),
// halving memory bandwidth compared to hasAnyChange.
//
// Algorithm: XOR-fold all 128-byte blocks into a 128-bit accumulator (Q0),
// with a 1-byte left rotation after each block to make the fold
// position-dependent (prevents false collisions from swapped blocks).
//
// The result is compared with the previous checksum stored at prevChecksum
// using ARM scalar operations. If different, the new checksum is written
// back and the function returns true. If identical, returns false.
//
// NEON register allocation:
//   Q0        (d0,d1):     128-bit XOR-fold accumulator
//   Q1-Q4     (d2-d9):     loaded data (64 bytes per pair of VLD1)
//
// ARM register allocation:
//   R0  = buf pointer (advances by 128 per block)
//   R1  = nblocks counter (decrements to 0)
//   R2  = prevChecksum pointer (preserved across loop)
//   R4-R7 = current checksum extracted from Q0
//   R0,R1,R3,R8 = previous checksum loaded from memory (reuse after loop)
//
// ABI0 stack layout (32-bit):
//   buf:          0(FP)  4 bytes
//   nblocks:      4(FP)  4 bytes
//   prevChecksum: 8(FP)  4 bytes
//   ret:         12(FP)  1 byte
//   Total args: 13 bytes
TEXT ·checksumChanged(SB), NOSPLIT|NOFRAME, $0-13
	MOVW	buf+0(FP), R0
	MOVW	nblocks+4(FP), R1
	MOVW	prevChecksum+8(FP), R2

	// Zero accumulator: Q0 = Q0 XOR Q0
	WORD	$0xF3000150	// VEOR Q0, Q0, Q0

	CMP	$0, R1
	BEQ	compare

loop:
	// Single-stream prefetch (one buffer only)
	WORD	$0xF5D0F200	// PLD [R0, #512]  — prefetch far (mem→L2)
	WORD	$0xF5D0F100	// PLD [R0, #256]  — prefetch near (L2→L1)

	// === First 64 bytes ===
	WORD	$0xF420220D	// VLD1.8 {d2,d3,d4,d5}, [R0]!
	WORD	$0xF420620D	// VLD1.8 {d6,d7,d8,d9}, [R0]!

	WORD	$0xF3000152	// VEOR Q0, Q0, Q1
	WORD	$0xF3000154	// VEOR Q0, Q0, Q2
	WORD	$0xF3000156	// VEOR Q0, Q0, Q3
	WORD	$0xF3000158	// VEOR Q0, Q0, Q4

	// === Second 64 bytes ===
	WORD	$0xF420220D	// VLD1.8 {d2,d3,d4,d5}, [R0]!
	WORD	$0xF420620D	// VLD1.8 {d6,d7,d8,d9}, [R0]!

	WORD	$0xF3000152	// VEOR Q0, Q0, Q1
	WORD	$0xF3000154	// VEOR Q0, Q0, Q2
	WORD	$0xF3000156	// VEOR Q0, Q0, Q3
	WORD	$0xF3000158	// VEOR Q0, Q0, Q4

	// Position-dependent rotation: left-rotate Q0 by 1 byte
	WORD	$0xF2B00140	// VEXT.8 Q0, Q0, Q0, #1

	SUB.S	$1, R1
	BNE	loop

compare:
	// Extract Q0 (current checksum) to ARM registers
	WORD	$0xEC554B10	// VMOV R4, R5, d0  (low 64 bits)
	WORD	$0xEC576B11	// VMOV R6, R7, d1  (high 64 bits)

	// Load previous checksum from memory (R0, R1, R3 free after loop)
	MOVW	0(R2), R0
	MOVW	4(R2), R1
	MOVW	8(R2), R3
	MOVW	12(R2), R8

	// Compare: XOR current with previous, OR all together
	EOR	R0, R4
	EOR	R1, R5
	EOR	R3, R6
	EOR	R8, R7
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4

	CMP	$0, R4
	BEQ	no_change

	// Changed: extract Q0 again and store to prevChecksum using ARM stores
	WORD	$0xEC554B10	// VMOV R4, R5, d0
	WORD	$0xEC576B11	// VMOV R6, R7, d1
	MOVW	R4, 0(R2)
	MOVW	R5, 4(R2)
	MOVW	R6, 8(R2)
	MOVW	R7, 12(R2)

	MOVW	$1, R4
	MOVB	R4, ret+12(FP)
	RET

no_change:
	MOVW	$0, R4
	MOVB	R4, ret+12(FP)
	RET
