//go:build arm && !nosimd

#include "textflag.h"

// func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
//
// Compares src and dst in 128-byte blocks using ARM NEON SIMD, copies src → dst
// only for changed portions, and writes 1 to mask[i] if block i differs, 0 otherwise.
//
// Each 128-byte block is processed as two 64-byte halves. Each half does its
// own conditional copy (unchanged halves skip the store). The mask combines
// both halves: if either half changed, mask[i] = 1.
//
// Uses NEON vector instructions encoded as WORD directives because Go's ARM32
// assembler does not support NEON mnemonics.
//
// NEON register allocation (per half):
//   Q0-Q3   (d0-d7):     src data (64 bytes)
//   Q8-Q11  (d16-d23):   dst data (64 bytes)
//   Q12-Q15 (d24-d31):   XOR differences and OR reduction
//
// ARM register allocation:
//   R0  = dst pointer (advances by 128 per block)
//   R1  = src pointer (advances by 128 per block)
//   R2  = mask pointer (advances by 1 per block)
//   R3  = nblocks counter (decrements to 0)
//   R4-R7 = temporaries for VMOV scalar extraction
//   R8  = temporary for mask byte value
//   R11 = saved first-half changed flag
//
// ABI0 stack layout (32-bit, all args on stack):
//   dst:        0(FP)   4 bytes
//   src:        4(FP)   4 bytes
//   mask base:  8(FP)   4 bytes
//   mask len:  12(FP)   4 bytes
//   mask cap:  16(FP)   4 bytes
//   nblocks:   20(FP)   4 bytes
//   Total args: 24 bytes
TEXT ·compareAndCopyBlocks(SB), NOSPLIT|NOFRAME, $0-24
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	mask+8(FP), R2
	MOVW	nblocks+20(FP), R3

	CMP	$0, R3
	BEQ	done

loop:
	// Prefetch data 512 bytes ahead into L1 cache.
	WORD	$0xF5D1F200	// PLD [R1, #512]  — prefetch src
	WORD	$0xF5D0F200	// PLD [R0, #512]  — prefetch dst

	// === First half (bytes 0-63) ===

	// Load 64 bytes from src with post-increment.
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3}, [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7}, [R1]!

	// Load 64 bytes from dst with post-increment.
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	// XOR to find differences
	WORD	$0xF3408170	// VEOR Q12, Q0, Q8
	WORD	$0xF342A172	// VEOR Q13, Q1, Q9
	WORD	$0xF344C174	// VEOR Q14, Q2, Q10
	WORD	$0xF346E176	// VEOR Q15, Q3, Q11

	// OR-reduce
	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Extract to scalar
	WORD	$0xEC554B38	// VMOV R4, R5, d24
	WORD	$0xEC576B39	// VMOV R6, R7, d25
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4		// R4 = first half changed flag

	// Conditional copy first half
	CMP	$0, R4
	BEQ	skip_first

	SUB	$64, R0
	WORD	$0xF400020D	// VST1.8 {d0,d1,d2,d3}, [R0]!
	WORD	$0xF400420D	// VST1.8 {d4,d5,d6,d7}, [R0]!

skip_first:
	// Save first half changed flag
	MOVW	R4, R11

	// === Second half (bytes 64-127) ===

	// Load 64 bytes from src with post-increment.
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3}, [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7}, [R1]!

	// Load 64 bytes from dst with post-increment.
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	// XOR to find differences
	WORD	$0xF3408170	// VEOR Q12, Q0, Q8
	WORD	$0xF342A172	// VEOR Q13, Q1, Q9
	WORD	$0xF344C174	// VEOR Q14, Q2, Q10
	WORD	$0xF346E176	// VEOR Q15, Q3, Q11

	// OR-reduce
	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Extract to scalar
	WORD	$0xEC554B38	// VMOV R4, R5, d24
	WORD	$0xEC576B39	// VMOV R6, R7, d25
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4		// R4 = second half changed flag

	// Conditional copy second half
	CMP	$0, R4
	BEQ	skip_second

	SUB	$64, R0
	WORD	$0xF400020D	// VST1.8 {d0,d1,d2,d3}, [R0]!
	WORD	$0xF400420D	// VST1.8 {d4,d5,d6,d7}, [R0]!

skip_second:
	// Set mask: combine both halves
	ORR	R11, R4		// R4 = any change in 128B block
	CMP	$0, R4
	MOVW.EQ	$0, R8
	MOVW.NE	$1, R8
	MOVB	R8, (R2)

	// Advance mask pointer
	ADD	$1, R2

	// Decrement block counter and loop
	SUB.S	$1, R3
	BNE	loop

done:
	RET
