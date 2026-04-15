//go:build arm && !nosimd

#include "textflag.h"

// func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool
//
// Read-only NEON scan: compares dst and src in 128-byte blocks.
// Returns true (1) at the first block that differs, false (0) if all
// blocks are identical. No writes to dst, src, or any mask — pure reads.
//
// Optimized for Cortex-A7:
//   - PLD at multiple distances covering all 4 cache lines per 128B block
//     (critical: removing PLD causes ~40% slowdown on dual-stream reads
//     because hardware prefetcher can't keep up; mid-block PLDs at +64
//     eliminate stalls on the second half of each block)
//   - Early exit on first difference (checks each 64-byte half independently)
//   - No conditional stores, no mask writes
//
// NEON register allocation (per 128B block, two 64B halves):
//   Q0-Q3   (d0-d7):     src data (64 bytes)
//   Q8-Q11  (d16-d23):   dst data (64 bytes)
//   Q12-Q15 (d24-d31):   XOR differences and OR reduction
//
// ARM register allocation:
//   R0  = dst pointer (advances by 128 per block)
//   R1  = src pointer (advances by 128 per block)
//   R2  = nblocks counter (decrements to 0)
//   R4-R7 = temporaries for VMOV scalar extraction
TEXT ·hasAnyChange(SB), NOSPLIT|NOFRAME, $0-13
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	nblocks+8(FP), R2

	CMP	$0, R2
	BEQ	no_change

loop:
	// Prefetch data ahead into cache hierarchy.
	// Critical for Cortex-A7: dual-stream reads need software prefetch.
	// Cover all 4 cache lines (32B each) of each 128B block at two distances.
	WORD	$0xF5D1F200	// PLD [R1, #512]  — src block+4 byte 0
	WORD	$0xF5D0F200	// PLD [R0, #512]  — dst block+4 byte 0
	WORD	$0xF5D1F240	// PLD [R1, #576]  — src block+4 byte 64
	WORD	$0xF5D0F240	// PLD [R0, #576]  — dst block+4 byte 64
	WORD	$0xF5D1F100	// PLD [R1, #256]  — src block+2 byte 0
	WORD	$0xF5D0F100	// PLD [R0, #256]  — dst block+2 byte 0
	WORD	$0xF5D1F140	// PLD [R1, #320]  — src block+2 byte 64
	WORD	$0xF5D0F140	// PLD [R0, #320]  — dst block+2 byte 64

	// === First half (bytes 0-63) ===

	// Load 64 bytes from src with post-increment
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3}, [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7}, [R1]!

	// Load 64 bytes from dst with post-increment
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	// XOR to find differences
	WORD	$0xF3408170	// VEOR Q12, Q0, Q8
	WORD	$0xF342A172	// VEOR Q13, Q1, Q9
	WORD	$0xF344C174	// VEOR Q14, Q2, Q10
	WORD	$0xF346E176	// VEOR Q15, Q3, Q11

	// OR-reduce first half
	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Check first half: extract to scalar and early-exit if any diff
	WORD	$0xEC554B38	// VMOV R4, R5, d24
	WORD	$0xEC576B39	// VMOV R6, R7, d25
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4
	CMP	$0, R4
	BNE	found_change

	// === Second half (bytes 64-127) ===

	// Load 64 bytes from src with post-increment
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3}, [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7}, [R1]!

	// Load 64 bytes from dst with post-increment
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	// XOR to find differences
	WORD	$0xF3408170	// VEOR Q12, Q0, Q8
	WORD	$0xF342A172	// VEOR Q13, Q1, Q9
	WORD	$0xF344C174	// VEOR Q14, Q2, Q10
	WORD	$0xF346E176	// VEOR Q15, Q3, Q11

	// OR-reduce second half
	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Check second half
	WORD	$0xEC554B38	// VMOV R4, R5, d24
	WORD	$0xEC576B39	// VMOV R6, R7, d25
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4

	CMP	$0, R4
	BNE	found_change

	// Next block
	SUB.S	$1, R2
	BNE	loop

no_change:
	MOVW	$0, R4
	MOVB	R4, ret+12(FP)
	RET

found_change:
	MOVW	$1, R4
	MOVB	R4, ret+12(FP)
	RET
