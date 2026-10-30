package delta

// rgb565LUT maps every 16-bit little-endian RGB565 value (lo | hi<<8) to the
// single-byte color-coded palette format used by reMarkable 1 streaming:
//
//	0-25  → grayscale (client renders value*10 per channel; 6,8,12,20,24 skipped)
//	6     → red
//	12    → blue
//	20    → green
//	24    → yellow
//	30    → transparent (white paper background)
//
// Precomputed at package init (64 KB) for O(1) lookup on non-NEON platforms and
// tail pixels. The arithmetic matches rgb565_arm.s bit-for-bit across all 65,536
// uint16 values.
var rgb565LUT [65536]byte

func init() {
	for px := range 65536 {
		rgb565LUT[px] = computeRGB565ColorByte(byte(px), byte(px>>8))
	}
}

// computeRGB565ColorByte converts one RGB565 LE pixel (lo, hi) to its 1-byte
// palette value using exact integer arithmetic identical to the ARMv7 NEON SIMD
// implementation in rgb565_arm.s.
func computeRGB565ColorByte(lo, hi byte) byte {
	// Expand 5-6-5 channels to 8-bit via bit replication:
	//   r = (r5 << 3) | (r5 >> 2)
	//   g = (g6 << 2) | (g6 >> 4)
	//   b = (b5 << 3) | (b5 >> 2)
	r := (hi & 0xF8) | (hi >> 5)
	g6 := ((hi & 0x07) << 3) | (lo >> 5)
	g := (g6 << 2) | (g6 >> 4)
	b5 := lo & 0x1F
	b := (b5 << 3) | (b5 >> 2)

	// Pen color detection (pairwise disjoint regions matching RM1 palette)
	if r > 184 && g < 80 && b < 80 {
		return 6 // Red
	}
	if r < 80 && g < 80 && b > 184 {
		return 12 // Blue
	}
	if g > 120 && r < 160 && b < 120 && g > r && g > b {
		return 20 // Green
	}
	if r > 184 && g > 180 && b < 120 {
		return 24 // Yellow
	}

	// Grayscale sum s = r + g + b in [0..765].
	// s > 737 corresponds to (r + g + b) / 3 > 245 (white paper → transparent).
	s := int(r) + int(g) + int(b)
	if s > 737 {
		return 30
	}

	// Fixed-point scaling matching VQDMULH.S16(s, 1115):
	// (2 * s * 1115) >> 16 maps [0..737] → [0..25] (≈ s * 25 / (3 * 245)).
	scaled := byte((2 * s * 1115) >> 16)
	switch scaled {
	case 6, 8, 12, 20, 24:
		scaled++
	}
	return scaled
}

// RGB565ToColorByte converts one little-endian RGB565 pixel (lo, hi) to the
// 1-byte RM1 color-coded value via table lookup.
func RGB565ToColorByte(lo, hi byte) byte {
	return rgb565LUT[uint16(lo)|uint16(hi)<<8]
}

// ConvertRGB565InPlace converts RGB565 little-endian pixel data in-place for
// the legacy stride-2 RLE encoder, writing the converted byte to each even index.
func ConvertRGB565InPlace(data []byte) {
	n := len(data) &^ 1
	i := 0
	for ; i+8 <= n; i += 8 {
		p0 := uint16(data[i]) | uint16(data[i+1])<<8
		p1 := uint16(data[i+2]) | uint16(data[i+3])<<8
		p2 := uint16(data[i+4]) | uint16(data[i+5])<<8
		p3 := uint16(data[i+6]) | uint16(data[i+7])<<8
		data[i] = rgb565LUT[p0]
		data[i+2] = rgb565LUT[p1]
		data[i+4] = rgb565LUT[p2]
		data[i+6] = rgb565LUT[p3]
	}
	for ; i < n; i += 2 {
		data[i] = rgb565LUT[uint16(data[i])|uint16(data[i+1])<<8]
	}
}
