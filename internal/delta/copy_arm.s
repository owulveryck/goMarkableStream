//go:build arm && !nosimd

#include "textflag.h"

// func copyBlocksNEON(dst, src unsafe.Pointer, nblocks int)
//
// Copies nblocks * 128 bytes from src to dst using 32-byte NEON VLD1.8 / VST1.8
// bursts. Crucial on reMarkable 1 (NXP i.MX6SL Cortex-A9) when src is mmap'd
// /dev/fb0 Write-Combine memory, where scalar LDR issues non-burst 4-byte AXI
// reads while VLD1.8 {d0-d3} issues full 32-byte cache-line AXI bursts.
//
// Leaves callee-saved VFP registers d8-d15 (Q4-Q7) completely untouched.
//
// NEON register allocation:
//   Q0-Q3   (d0-d7):   bytes 0..63
//   Q8-Q11  (d16-d23): bytes 64..127
//
// ARM register allocation:
//   R0 = dst pointer (advances by 128 per block)
//   R1 = src pointer (advances by 128 per block)
//   R2 = nblocks counter (decrements to 0)
TEXT ·copyBlocksNEON(SB), NOSPLIT|NOFRAME, $0-12
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	nblocks+8(FP), R2

	CMP	$0, R2
	BEQ	done

copy_loop:
	// Prefetch source cache lines (NOP on Write-Combine, active on cached RAM)
	WORD	$0xF5D1F200	// PLD [R1, #512]
	WORD	$0xF5D1F240	// PLD [R1, #576]
	WORD	$0xF5D1F100	// PLD [R1, #256]
	WORD	$0xF5D1F140	// PLD [R1, #320]

	// Load 128 bytes in four 32-byte bursts
	WORD	$0xF421020D	// VLD1.8 {d0,d1,d2,d3},     [R1]!
	WORD	$0xF421420D	// VLD1.8 {d4,d5,d6,d7},     [R1]!
	WORD	$0xF461020D	// VLD1.8 {d16,d17,d18,d19}, [R1]!
	WORD	$0xF461420D	// VLD1.8 {d20,d21,d22,d23}, [R1]!

	// Store 128 bytes in four 32-byte bursts
	WORD	$0xF400020D	// VST1.8 {d0,d1,d2,d3},     [R0]!
	WORD	$0xF400420D	// VST1.8 {d4,d5,d6,d7},     [R0]!
	WORD	$0xF440020D	// VST1.8 {d16,d17,d18,d19}, [R0]!
	WORD	$0xF440420D	// VST1.8 {d20,d21,d22,d23}, [R0]!

	SUB.S	$1, R2
	BNE	copy_loop

done:
	RET
