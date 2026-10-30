//go:build arm && !nosimd

#include "textflag.h"

// func checksumChanged(buf unsafe.Pointer, nblocks int, prevChecksum unsafe.Pointer) bool
//
// Single-buffer ARMv7 NEON 128-bit XOR-fold checksum for idle frame detection.
// Produces the exact same 16-byte position-dependent checksum as checksum_generic.go,
// while issuing 32-byte burst VLD1.8 loads (critical for Write-Combine mmap'd
// /dev/fb0 on reMarkable 1 Cortex-A9) and prefetching all 4 cache lines per
// 128-byte block (#256, #320, #512, #576) for cached RAM on reMarkable 2.
//
// Leaves callee-saved VFP registers d8-d15 (Q4-Q7) completely untouched.
//
// NEON register allocation:
//   Q0    (d0,d1):     128-bit XOR-fold accumulator
//   Q1,Q2 (d2-d5):     32-byte burst load 1
//   Q8,Q9 (d16-d19):   32-byte burst load 2
//
// ARM register allocation:
//   R0  = buf pointer (advances by 128 per block)
//   R1  = nblocks counter (decrements to 0)
//   R2  = prevChecksum pointer
//   R4-R7 = current checksum extracted from Q0
//   R0,R1,R3,R8 = previous checksum loaded from prevChecksum after loop
TEXT ·checksumChanged(SB), NOSPLIT|NOFRAME, $0-13
	MOVW	buf+0(FP), R0
	MOVW	nblocks+4(FP), R1
	MOVW	prevChecksum+8(FP), R2

	// Zero accumulator: Q0 = 0
	WORD	$0xF3000150	// VEOR Q0, Q0, Q0

	CMP	$0, R1
	BEQ	compare

loop:
	// Prefetch all 4 cache lines (32B each) of upcoming 128B blocks
	WORD	$0xF5D0F200	// PLD [R0, #512]  — far  (block+4, byte 0)
	WORD	$0xF5D0F240	// PLD [R0, #576]  — far  (block+4, byte 64)
	WORD	$0xF5D0F100	// PLD [R0, #256]  — near (block+2, byte 0)
	WORD	$0xF5D0F140	// PLD [R0, #320]  — near (block+2, byte 64)

	// === First 64 bytes (tree reduction avoids Q0 accumulator dependency chain) ===
	WORD	$0xF420220D	// VLD1.8 {d2,d3,d4,d5},     [R0]!
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF3022154	// VEOR   Q1, Q1, Q2
	WORD	$0xF34001F2	// VEOR   Q8, Q8, Q9
	WORD	$0xF3022170	// VEOR   Q1, Q1, Q8
	WORD	$0xF3000152	// VEOR   Q0, Q0, Q1

	// === Second 64 bytes ===
	WORD	$0xF420220D	// VLD1.8 {d2,d3,d4,d5},     [R0]!
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF3022154	// VEOR   Q1, Q1, Q2
	WORD	$0xF34001F2	// VEOR   Q8, Q8, Q9
	WORD	$0xF3022170	// VEOR   Q1, Q1, Q8
	WORD	$0xF3000152	// VEOR   Q0, Q0, Q1

	// Rotate left by 1 byte (8 bits in LE, matching checksum_generic.go)
	WORD	$0xF2B00F40	// VEXT.8 Q0, Q0, Q0, #15

	SUB.S	$1, R1
	BNE	loop

compare:
	// Extract Q0 (current checksum) to ARM registers once
	WORD	$0xEC554B10	// VMOV R4, R5, d0
	WORD	$0xEC576B11	// VMOV R6, R7, d1

	// Load previous checksum from memory (R0, R1, R3, R8 free after loop)
	MOVW	0(R2), R0
	MOVW	4(R2), R1
	MOVW	8(R2), R3
	MOVW	12(R2), R8

	// Compare without clobbering R4-R7 so we don't need a second VMOV on change
	EOR	R4, R0, R0
	EOR	R5, R1, R1
	EOR	R6, R3, R3
	EOR	R7, R8, R8
	ORR	R1, R0, R0
	ORR	R8, R3, R3
	ORR.S	R3, R0, R0
	BEQ	no_change

	// Changed: store R4-R7 directly to prevChecksum
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
