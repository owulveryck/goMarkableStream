//go:build arm && !nosimd

#include "textflag.h"

// func hasAnyChange(dst, src unsafe.Pointer, nblocks int) bool
//
// Read-only NEON scan: compares dst and src in 128-byte blocks.
// Returns true (1) at the first block that differs, false (0) if all
// blocks are identical. No writes to dst, src, or any mask — pure reads.
//
// Optimized for both Cortex-A9 (reMarkable 1) and Cortex-A7 (reMarkable 2):
//   - Folds the entire 128-byte block down to 64 bits inside NEON (VORR Q8,Q8,Q12
//     + VORR d16,d16,d17) so only ONE NEON→ARM VMOV + branch is executed per
//     128B block instead of four VMOVs and two branches (4x fewer cross-domain
//     stalls on Cortex-A9).
//   - Dual-stream PLD at near (+2 blocks) and far (+4 blocks) distances covering
//     all 4 cache lines (32B each) per 128B block.
//   - Preserves callee-saved VFP registers d8-d15 (Q4-Q7) untouched.
//
// NEON register allocation:
//   Q0-Q3   (d0-d7):   src data (64 bytes per half)
//   Q8-Q11  (d16-d23): first-half dst data & XOR/OR reduction
//   Q12-Q15 (d24-d31): second-half dst data & XOR/OR reduction
//
// ARM register allocation:
//   R0 = dst pointer (advances by 128 per block)
//   R1 = src pointer (advances by 128 per block)
//   R2 = nblocks counter (decrements to 0)
//   R4,R5 = 64-bit diff extracted from d16
TEXT ·hasAnyChange(SB), NOSPLIT|NOFRAME, $0-13
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	nblocks+8(FP), R2

	CMP	$0, R2
	BEQ	no_change

loop:
	// === First half (bytes 0-63) ===
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3},     [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7},     [R1]!
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	WORD	$0xF3400170	// VEOR Q8,  Q0, Q8
	WORD	$0xF3422172	// VEOR Q9,  Q1, Q9
	WORD	$0xF3444174	// VEOR Q10, Q2, Q10
	WORD	$0xF3466176	// VEOR Q11, Q3, Q11

	WORD	$0xF26001F2	// VORR Q8,  Q8,  Q9
	WORD	$0xF26441F6	// VORR Q10, Q10, Q11
	WORD	$0xF26001F4	// VORR Q8,  Q8,  Q10

	// Prefetch all 4 cache lines of upcoming blocks (R0, R1 at block + 64)
	WORD	$0xF5D1F1C0	// PLD [R1, #448]  — src block+4 byte 0
	WORD	$0xF5D0F1C0	// PLD [R0, #448]  — dst block+4 byte 0
	WORD	$0xF5D1F200	// PLD [R1, #512]  — src block+4 byte 64
	WORD	$0xF5D0F200	// PLD [R0, #512]  — dst block+4 byte 64
	WORD	$0xF5D1F0C0	// PLD [R1, #192]  — src block+2 byte 0
	WORD	$0xF5D0F0C0	// PLD [R0, #192]  — dst block+2 byte 0
	WORD	$0xF5D1F100	// PLD [R1, #256]  — src block+2 byte 64
	WORD	$0xF5D0F100	// PLD [R0, #256]  — dst block+2 byte 64

	// === Second half (bytes 64-127) ===
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3},     [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7},     [R1]!
	WORD	$0xF460820D	// VLD1.8 {d24,d25,d26,d27}, [R0]!
	WORD	$0xF460C20D	// VLD1.8 {d28,d29,d30,d31}, [R0]!

	WORD	$0xF3408178	// VEOR Q12, Q0, Q12
	WORD	$0xF342A17A	// VEOR Q13, Q1, Q13
	WORD	$0xF344C17C	// VEOR Q14, Q2, Q14
	WORD	$0xF346E17E	// VEOR Q15, Q3, Q15

	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Combine both halves inside NEON → single 64-bit extraction per 128B block
	WORD	$0xF26001F8	// VORR Q8,  Q8,  Q12
	WORD	$0xF26001B1	// VORR d16, d16, d17
	WORD	$0xEC554B30	// VMOV R4, R5, d16

	ORR.S	R5, R4, R4
	BNE	found_change

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
