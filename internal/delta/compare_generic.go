//go:build !arm64 && (!arm || nosimd)

package delta

import "unsafe"

func compareAndCopyBlocks(dst, src unsafe.Pointer, mask []byte, nblocks int) {
	for i := range nblocks {
		off := i * 64
		changed := byte(0)
		for j := 0; j < 64; j += 8 {
			curr := *(*uint64)(unsafe.Add(src, off+j))
			prev := *(*uint64)(unsafe.Add(dst, off+j))
			if curr != prev {
				changed = 1
				*(*uint64)(unsafe.Add(dst, off+j)) = curr
			}
		}
		mask[i] = changed
	}
}
