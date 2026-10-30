//go:build arm && !nosimd

package delta

import "unsafe"

// convertRGB565ToColor8SIMD converts groups16*16 little-endian RGB565 pixels
// (groups16*32 bytes) from src into groups16*16 Color8 bytes at dst using
// ARMv7 NEON SIMD instructions.
//
//go:noescape
func convertRGB565ToColor8SIMD(dst, src unsafe.Pointer, groups16 int)

// ConvertRGB565ToColor8 converts len(src)/2 little-endian RGB565 pixels from src
// into packed 1-byte-per-pixel Color8 values in dst.
// Uses ARMv7 NEON SIMD for 16-pixel chunks and table lookup for any remainder.
func ConvertRGB565ToColor8(dst, src []byte) {
	npixels := len(src) / 2
	if npixels == 0 {
		return
	}
	_ = dst[npixels-1]

	groups16 := npixels / 16
	if groups16 > 0 {
		convertRGB565ToColor8SIMD(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), groups16)
	}

	for p := groups16 * 16; p < npixels; p++ {
		dst[p] = rgb565LUT[uint16(src[p*2])|uint16(src[p*2+1])<<8]
	}
}
