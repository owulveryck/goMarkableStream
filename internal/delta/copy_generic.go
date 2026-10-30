//go:build !arm || nosimd

package delta

// CopyFrame copies min(len(dst), len(src)) bytes from src to dst.
func CopyFrame(dst, src []byte) int {
	return copy(dst, src)
}
