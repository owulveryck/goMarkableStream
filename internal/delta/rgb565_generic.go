//go:build !arm || nosimd

package delta

import "encoding/binary"

// ConvertRGB565ToColor8 converts len(src)/2 little-endian RGB565 pixels from src
// into packed 1-byte-per-pixel Color8 values in dst.
// Caller must ensure len(dst) >= len(src)/2.
func ConvertRGB565ToColor8(dst, src []byte) {
	npixels := len(src) / 2
	if npixels == 0 {
		return
	}
	_ = dst[npixels-1]

	// Process 4 pixels (8 bytes) per iteration via 64-bit little-endian loads
	// to minimize bounds checks and byte-load instructions.
	p := 0
	for ; p+4 <= npixels; p += 4 {
		w := binary.LittleEndian.Uint64(src[p*2 : p*2+8])
		dst[p] = rgb565LUT[uint16(w)]
		dst[p+1] = rgb565LUT[uint16(w>>16)]
		dst[p+2] = rgb565LUT[uint16(w>>32)]
		dst[p+3] = rgb565LUT[uint16(w>>48)]
	}
	for ; p < npixels; p++ {
		dst[p] = rgb565LUT[uint16(src[p*2])|uint16(src[p*2+1])<<8]
	}
}
