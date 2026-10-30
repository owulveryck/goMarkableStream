//go:build arm && !nosimd

#include "textflag.h"

// func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int)
//
// Compares src and dst in 128-byte blocks using ARM NEON SIMD, copies src → dst
// only for changed 64-byte halves, and writes 1 to mask[i] if block i differs,
// 0 otherwise.
//
// Microarchitectural optimizations for BOTH reMarkable 1 (Cortex-A9) and
// reMarkable 2 (Cortex-A7):
//   1. Cortex-A9 VMOV cross-domain latency hiding:
//      On Cortex-A9, moving NEON registers to ARM registers (VMOV Rd, Rn, Dm)
//      and immediately branching on Rd incurs a ~20-cycle pipeline stall.
//      We fold 128b → 64b inside NEON (VORR d16, d16, d17) to halve the number
//      of VMOVs (2 per 128B block instead of 4), and issue the first half's
//      VMOV BEFORE the 20 independent PLD + second-half VLD1/VEOR/VORR
//      instructions so R4/R5 arrive stall-free on Cortex-A9's OoO core.
//   2. Cortex-A7 + Cortex-A9 dual-stream software prefetching (PLD):
//      Prefetches all 4 cache lines (32B each) of upcoming 128B blocks at near
//      (+2 blocks) and far (+4 blocks) distances. Issued after first-half
//      post-increments (+64), so offsets #448/#512/#192/#256 correspond to
//      #512/#576/#256/#320 from block start.
//   3. Write-Combine burst friendly:
//      Reads src strictly once via 32-byte VLD1.8 bursts (holding first half in
//      Q0-Q3 and second half in Q12-Q15), ensuring tear-free snapshots and full
//      AXI bursts when src is mmap'd /dev/fb0 on reMarkable 1.
//   4. Callee-saved VFP registers d8-d15 (Q4-Q7) are left completely untouched.
//
// NEON register allocation:
//   Q0-Q3   (d0-d7):   first-half src data (64 bytes, preserved for store)
//   Q12-Q15 (d24-d31): second-half src data (64 bytes, preserved for store)
//   Q8-Q11  (d16-d23): dst data (64 bytes per half) & XOR/OR reduction
//
// ARM register allocation:
//   R0  = dst pointer (advances by 128 per block)
//   R1  = src pointer (advances by 128 per block)
//   R2  = mask pointer (advances by 1 per block)
//   R3  = nblocks counter (decrements to 0)
//   R4,R5 = first-half 64-bit diff extracted from d16
//   R6,R7 = second-half 64-bit diff extracted from d16
//   R8  = store address / mask byte temporary
TEXT ·compareAndCopyBlocks(SB), NOSPLIT|NOFRAME, $0-24
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	mask+8(FP), R2
	MOVW	nblocks+20(FP), R3

	CMP	$0, R3
	BEQ	done

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
	WORD	$0xF26001B1	// VORR d16, d16, d17

	// Issue first-half NEON→ARM transfer early to hide ~20-cycle Cortex-A9 latency
	WORD	$0xEC554B30	// VMOV R4, R5, d16

	// Dual-stream software prefetches (R0 and R1 are currently at block + 64)
	WORD	$0xF5D1F1C0	// PLD [R1, #448]  — src block+4 byte 0
	WORD	$0xF5D0F1C0	// PLD [R0, #448]  — dst block+4 byte 0
	WORD	$0xF5D1F200	// PLD [R1, #512]  — src block+4 byte 64
	WORD	$0xF5D0F200	// PLD [R0, #512]  — dst block+4 byte 64
	WORD	$0xF5D1F0C0	// PLD [R1, #192]  — src block+2 byte 0
	WORD	$0xF5D0F0C0	// PLD [R0, #192]  — dst block+2 byte 0
	WORD	$0xF5D1F100	// PLD [R1, #256]  — src block+2 byte 64
	WORD	$0xF5D0F100	// PLD [R0, #256]  — dst block+2 byte 64

	// === Second half (bytes 64-127) ===
	WORD	$0xF461820D	// VLD1.8 {d24,d25,d26,d27}, [R1]!
	WORD	$0xF461C20D	// VLD1.8 {d28,d29,d30,d31}, [R1]!
	WORD	$0xF460020D	// VLD1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF460420D	// VLD1.8 {d20,d21,d22,d23}, [R0]!

	WORD	$0xF34801F0	// VEOR Q8,  Q12, Q8
	WORD	$0xF34A21F2	// VEOR Q9,  Q13, Q9
	WORD	$0xF34C41F4	// VEOR Q10, Q14, Q10
	WORD	$0xF34E61F6	// VEOR Q11, Q15, Q11

	WORD	$0xF26001F2	// VORR Q8,  Q8,  Q9
	WORD	$0xF26441F6	// VORR Q10, Q10, Q11
	WORD	$0xF26001F4	// VORR Q8,  Q8,  Q10
	WORD	$0xF26001B1	// VORR d16, d16, d17

	// Issue second-half NEON→ARM transfer before checking first-half result
	WORD	$0xEC576B30	// VMOV R6, R7, d16

	// Conditional copy first half (R4, R5 have been ready for >20 instructions)
	ORR.S	R5, R4, R4
	BEQ	skip_first

	SUB	$128, R0, R8
	WORD	$0xF408020D	// VST1.8 {d0,d1,d2,d3}, [R8]!
	WORD	$0xF408420D	// VST1.8 {d4,d5,d6,d7}, [R8]!

skip_first:
	// Conditional copy second half
	ORR.S	R7, R6, R6
	BEQ	skip_second

	SUB	$64, R0, R8
	WORD	$0xF448820D	// VST1.8 {d24,d25,d26,d27}, [R8]!
	WORD	$0xF448C20D	// VST1.8 {d28,d29,d30,d31}, [R8]!

skip_second:
	// Set mask byte: 1 if either half changed, 0 otherwise
	ORR.S	R6, R4, R4
	MOVW.EQ	$0, R8
	MOVW.NE	$1, R8
	MOVB	R8, (R2)
	ADD	$1, R2

	SUB.S	$1, R3
	BNE	loop

done:
	RET
