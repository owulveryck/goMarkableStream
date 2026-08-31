package remarkable

import "fmt"

func copyVisibleRows(dst, src []byte, visibleRowBytes, storageRowBytes, height int) error {
	if visibleRowBytes <= 0 || storageRowBytes < visibleRowBytes || height <= 0 {
		return fmt.Errorf(
			"invalid framebuffer layout: visible row %d, storage row %d, height %d",
			visibleRowBytes,
			storageRowBytes,
			height,
		)
	}
	wantDst := visibleRowBytes * height
	wantSrc := storageRowBytes * height
	if len(dst) != wantDst || len(src) < wantSrc {
		return fmt.Errorf(
			"invalid framebuffer buffers: destination %d (want %d), source %d (want at least %d)",
			len(dst),
			wantDst,
			len(src),
			wantSrc,
		)
	}

	for y := 0; y < height; y++ {
		srcStart := y * storageRowBytes
		dstStart := y * visibleRowBytes
		copy(dst[dstStart:dstStart+visibleRowBytes], src[srcStart:srcStart+visibleRowBytes])
	}
	return nil
}
