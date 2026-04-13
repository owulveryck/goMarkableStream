//go:build arm && !nosimd

#include "textflag.h"

// func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
//
// Compares src and dst in 64-byte blocks using ARM NEON SIMD, copies src → dst
// only for changed blocks, and writes 1 to mask[i] if block i differs, 0 otherwise.
//
// Conditional copy optimization: unchanged blocks (98% during typical handwriting)
// are NOT written back, saving ~33% memory bandwidth on the write side.
// This is safe because unchanged blocks already have identical data in dst.
//
// Uses NEON vector instructions encoded as WORD directives because Go's ARM32
// assembler does not support NEON mnemonics. Each WORD is annotated with its
// ARM NEON mnemonic equivalent.
//
// NEON register allocation:
//   Q0-Q3   (d0-d7):     src data (64 bytes per iteration)
//   Q8-Q11  (d16-d23):   dst data (64 bytes per iteration)
//   Q12-Q15 (d24-d31):   XOR differences and OR reduction
//
// ARM register allocation:
//   R0 = dst pointer (previous frame, read+write, advances by 64 per block)
//   R1 = src pointer (current frame, read only, advances by 64 per block)
//   R2 = mask pointer (advances by 1 per block)
//   R3 = nblocks counter (decrements to 0)
//   R4-R7 = temporaries for VMOV scalar extraction
//   R8 = temporary for mask byte value
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
	// Prefetch data 256 bytes ahead into L1 cache.
	// On Cortex-A9, PLD initiates a linefill without stalling the pipeline.
	WORD	$0xF5D1F100	// PLD [R1, #256]  — prefetch src
	WORD	$0xF5D0F100	// PLD [R0, #256]  — prefetch dst

	// Load 64 bytes from src (current frame) with post-increment.
	// Two VLD1 instructions, each loading 32 bytes (4 × d-registers).
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3}, [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7}, [R1]!

	// Load 64 bytes from dst (previous frame) with post-increment.
	// After this, R0 points to the NEXT block. If unchanged, no rewind needed.
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	// XOR each 16-byte chunk to detect byte-level differences.
	// Q0 ^Q8, Q1 ^Q9, Q2 ^Q10, Q3 ^Q11
	WORD	$0xF3408170	// VEOR Q12, Q0, Q8
	WORD	$0xF342A172	// VEOR Q13, Q1, Q9
	WORD	$0xF344C174	// VEOR Q14, Q2, Q10
	WORD	$0xF346E176	// VEOR Q15, Q3, Q11

	// OR-reduce all difference vectors into Q12.
	// Any non-zero byte in Q12 means the block has changed.
	WORD	$0xF26881FA	// VORR Q12, Q12, Q13
	WORD	$0xF26CC1FE	// VORR Q14, Q14, Q15
	WORD	$0xF26881FC	// VORR Q12, Q12, Q14

	// Extract the 128-bit result to four 32-bit ARM registers.
	WORD	$0xEC554B38	// VMOV R4, R5, d24  — lower 64 bits of Q12
	WORD	$0xEC576B39	// VMOV R6, R7, d25  — upper 64 bits of Q12

	// Combine all 32-bit values: if any is non-zero, block has changed.
	ORR	R5, R4
	ORR	R7, R6
	ORR	R6, R4

	// Set mask byte: 1 if any byte differs, 0 if block is identical.
	CMP	$0, R4
	MOVW.EQ	$0, R8
	MOVW.NE	$1, R8
	MOVB	R8, (R2)

	// Advance mask pointer to next byte.
	ADD	$1, R2

	// Conditional copy: only write back changed blocks.
	// CMP above set Z=1 if R4==0 (unchanged). MOVW.EQ/NE don't modify flags.
	// For unchanged blocks (~98% during handwriting), skip the store entirely.
	// R0 already points to the next block from VLD1 post-increment.
	BEQ	skip_copy

	// Changed block: rewind dst, store src data, advance dst.
	SUB	$64, R0
	WORD	$0xF400020D	// VST1.8 {d0,d1,d2,d3}, [R0]!
	WORD	$0xF400420D	// VST1.8 {d4,d5,d6,d7}, [R0]!

skip_copy:
	// R0 is at the next block regardless of whether we copied.

	// Decrement block counter and loop if blocks remain.
	SUB.S	$1, R3
	BNE	loop

done:
	RET
