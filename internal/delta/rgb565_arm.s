//go:build arm && !nosimd

#include "textflag.h"

// func convertRGB565ToColor8SIMD(dst, src unsafe.Pointer, groups16 int)
//
// Vectorized RGB565 (little-endian) → 1-byte Color8 converter for ARMv7 NEON,
// specifically tuned for the reMarkable 1's Cortex-A9 + NEON MPE:
//   - Processes 16 pixels (32 bytes → 16 bytes) per iteration
//   - Uses VLD2.8 hardware de-interleaving to split lo/hi RGB565 bytes in one load
//   - 100% branchless SIMD (VBSL / VCGT / VCEQ / VQDMULH): zero branch
//     mispredictions on anti-aliased strokes and zero NEON→ARM VMOV stalls
//   - Preserves callee-saved VFP registers d8-d15 (Q4-Q7) completely untouched
//   - Matches computeRGB565ColorByte bit-for-bit across all 65,536 uint16 values
//
// NEON register allocation (Q4-Q7 / d8-d15 untouched):
//   Q0  (d0,d1):   lo bytes / scratch predicate
//   Q1  (d2,d3):   hi bytes / scratch predicate
//   Q2  (d4,d5):   8-bit R channel
//   Q3  (d6,d7):   8-bit G channel
//   Q8  (d16,d17): 8-bit B channel
//   Q9  (d18,d19): 16-bit sum low / scaled / color accumulator
//   Q10 (d20,d21): 16-bit sum high / predicates / final 16-pixel output
//   Q11 (d22,d23): white-paper mask & grayscale/white result
//   Q12 (d24,d25): scratch mask
//   Q13 (d26,d27): constant 1115 (int16x8) for VQDMULH grayscale scaling
//   Q14 (d28,d29): constant 737  (uint16x8) for white-paper threshold
//   Q15 (d30,d31): constant 0xF8 (uint8x16) for R5 mask
//
// ARM register allocation:
//   R0 = dst pointer (advances by 16 per iteration)
//   R1 = src pointer (advances by 32 per iteration)
//   R2 = groups16 counter (decrements to 0)
TEXT ·convertRGB565ToColor8SIMD(SB), NOSPLIT|NOFRAME, $0-12
	MOVW	dst+0(FP), R0
	MOVW	src+4(FP), R1
	MOVW	groups16+8(FP), R2

	CMP	$0, R2
	BEQ	done

	// Hoist loop-invariant SIMD constants
	WORD	$0xF3C7EE58	// VMOV.I8  Q15, #0xF8
	WORD	$0xF3C6C851	// VMOV.I16 Q14, #225
	WORD	$0xF2C0CB52	// VORR.I16 Q14, #0x0200  (225 | 512 = 737)
	WORD	$0xF2C5A85B	// VMOV.I16 Q13, #91
	WORD	$0xF2C0AB54	// VORR.I16 Q13, #0x0400  (91 | 1024 = 1115)

rgb_loop:
	// Prefetch source cache lines (32B line size on Cortex-A9 / PL310)
	WORD	$0xF5D1F100	// PLD [R1, #256]  — near (L2→L1)
	WORD	$0xF5D1F200	// PLD [R1, #512]  — far  (mem→L2)

	// Load 16 RGB565 LE pixels (32 bytes), de-interleaving into Q0 (lo) and Q1 (hi)
	WORD	$0xF421030D	// VLD2.8 {d0, d1, d2, d3}, [R1]!

	// Extract 8-bit R = (hi & 0xF8) | (hi >> 5) into Q2
	WORD	$0xF202417E	// VAND    Q2, Q1, Q15
	WORD	$0xF38B4154	// VSRA.U8 Q2, Q2, #5

	// Extract 8-bit G = (g6 << 2) | (g6 >> 4) into Q3
	WORD	$0xF28D6552	// VSHL.I8 Q3, Q1, #5
	WORD	$0xF3CB0050	// VSHR.U8 Q8, Q0, #5
	WORD	$0xF2CA0570	// VSHL.I8 Q8, Q8, #2
	WORD	$0xF2266170	// VORR    Q3, Q3, Q8
	WORD	$0xF38A6156	// VSRA.U8 Q3, Q3, #6

	// Extract 8-bit B = ((lo & 0x1F) << 3) | ((lo & 0x1F) >> 2) into Q8
	WORD	$0xF2CB0550	// VSHL.I8 Q8, Q0, #3
	WORD	$0xF3CB0170	// VSRA.U8 Q8, Q8, #5

	// 16-bit channel sum s = R + G + B in Q9 (low 8 px) and Q10 (high 8 px)
	WORD	$0xF3C42006	// VADDL.U8 Q9, d4, d6
	WORD	$0xF3C221A0	// VADDW.U8 Q9, Q9, d16
	WORD	$0xF3C54007	// VADDL.U8 Q10, d5, d7
	WORD	$0xF3C441A1	// VADDW.U8 Q10, Q10, d17

	// White-paper mask: is_white = (s > 737) narrowed to 16 bytes in Q11
	WORD	$0xF35263EC	// VCGT.U16  Q11, Q9, Q14
	WORD	$0xF35483EC	// VCGT.U16  Q12, Q10, Q14
	WORD	$0xF3F26226	// VMOVN.I16 d22, Q11
	WORD	$0xF3F27228	// VMOVN.I16 d23, Q12

	// Grayscale scaling: scaled = (2 * s * 1115) >> 16 narrowed to 16 bytes in Q9
	WORD	$0xF2522BEA	// VQDMULH.S16 Q9, Q9, Q13
	WORD	$0xF2544BEA	// VQDMULH.S16 Q10, Q10, Q13
	WORD	$0xF3F22222	// VMOVN.I16   d18, Q9
	WORD	$0xF3F23224	// VMOVN.I16   d19, Q10

	// Increment scaled by 1 if it collides with reserved color codes {6,8,12,20,24}.
	// VCEQ produces 0xFF (-1 in int8), so VSUB.I8 adds +1 on matching lanes.
	WORD	$0xF2C04E56	// VMOV.I8 Q10, #6
	WORD	$0xF34288F4	// VCEQ.I8 Q12, Q9, Q10
	WORD	$0xF2C04E58	// VMOV.I8 Q10, #8
	WORD	$0xF34248F4	// VCEQ.I8 Q10, Q9, Q10
	WORD	$0xF26881F4	// VORR    Q12, Q12, Q10
	WORD	$0xF2C04E5C	// VMOV.I8 Q10, #12
	WORD	$0xF34248F4	// VCEQ.I8 Q10, Q9, Q10
	WORD	$0xF26881F4	// VORR    Q12, Q12, Q10
	WORD	$0xF2C14E54	// VMOV.I8 Q10, #20
	WORD	$0xF34248F4	// VCEQ.I8 Q10, Q9, Q10
	WORD	$0xF26881F4	// VORR    Q12, Q12, Q10
	WORD	$0xF2C14E58	// VMOV.I8 Q10, #24
	WORD	$0xF34248F4	// VCEQ.I8 Q10, Q9, Q10
	WORD	$0xF26881F4	// VORR    Q12, Q12, Q10
	WORD	$0xF34228E8	// VSUB.I8 Q9, Q9, Q12

	// Select white paper (30) vs grayscale: Q11 = is_white ? 30 : scaled
	WORD	$0xF2C14E5E	// VMOV.I8 Q10, #30
	WORD	$0xF35461F2	// VBSL    Q11, Q10, Q9

	// Shared pen-color channel predicates
	WORD	$0xF3C38E58	// VMOV.I8 Q12, #184
	WORD	$0xF3040368	// VCGT.U8 Q0, Q2, Q12   (R > 184)
	WORD	$0xF30023E8	// VCGT.U8 Q1, Q8, Q12   (B > 184)
	WORD	$0xF2C58E50	// VMOV.I8 Q12, #80
	WORD	$0xF34823C4	// VCGT.U8 Q9, Q12, Q2   (R < 80)
	WORD	$0xF34843C6	// VCGT.U8 Q10, Q12, Q3  (G < 80)
	WORD	$0xF34883E0	// VCGT.U8 Q12, Q12, Q8  (B < 80)

	// Blue (12): R < 80 && G < 80 && B > 184
	WORD	$0xF24221F4	// VAND    Q9, Q9, Q10
	WORD	$0xF24221D2	// VAND    Q9, Q9, Q1
	WORD	$0xF2802E5C	// VMOV.I8 Q1, #12
	WORD	$0xF3522176	// VBSL    Q9, Q1, Q11

	// Red (6): R > 184 && G < 80 && B < 80
	WORD	$0xF24881F4	// VAND    Q12, Q12, Q10
	WORD	$0xF24881D0	// VAND    Q12, Q12, Q0
	WORD	$0xF2802E56	// VMOV.I8 Q1, #6
	WORD	$0xF3528172	// VBSL    Q12, Q1, Q9

	// Predicates B < 120 (Q1) and G > 120 (Q10)
	WORD	$0xF2C74E58	// VMOV.I8 Q10, #120
	WORD	$0xF30423E0	// VCGT.U8 Q1, Q10, Q8   (B < 120)
	WORD	$0xF3464364	// VCGT.U8 Q10, Q3, Q10  (G > 120)

	// Yellow (24): R > 184 && G > 180 && B < 120
	WORD	$0xF3C32E54	// VMOV.I8 Q9, #180
	WORD	$0xF3462362	// VCGT.U8 Q9, Q3, Q9    (G > 180)
	WORD	$0xF24221D0	// VAND    Q9, Q9, Q0
	WORD	$0xF24221D2	// VAND    Q9, Q9, Q1
	WORD	$0xF2810E58	// VMOV.I8 Q0, #24
	WORD	$0xF3502178	// VBSL    Q9, Q0, Q12

	// Green (20): G > 120 && R < 160 && B < 120 && G > R && G > B
	WORD	$0xF3820E50	// VMOV.I8 Q0, #160
	WORD	$0xF3000344	// VCGT.U8 Q0, Q0, Q2    (R < 160)
	WORD	$0xF24441D0	// VAND    Q10, Q10, Q0
	WORD	$0xF24441D2	// VAND    Q10, Q10, Q1
	WORD	$0xF3060344	// VCGT.U8 Q0, Q3, Q2    (G > R)
	WORD	$0xF24441D0	// VAND    Q10, Q10, Q0
	WORD	$0xF3060360	// VCGT.U8 Q0, Q3, Q8    (G > B)
	WORD	$0xF24441D0	// VAND    Q10, Q10, Q0
	WORD	$0xF2810E54	// VMOV.I8 Q0, #20
	WORD	$0xF3504172	// VBSL    Q10, Q0, Q9

	// Store 16 Color8 bytes from Q10 ({d20, d21})
	WORD	$0xF4404A0D	// VST1.8 {d20, d21}, [R0]!

	SUB.S	$1, R2
	BNE	rgb_loop

done:
	RET
