package remarkable

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	maxHeaderIterations = 100
	minHeaderLength     = 2
	headerReadOffset    = 8
	headerSize          = 8
)

func calculateFramePointerFrom(r io.ReaderAt, region memoryRange, targetSize, requiredSize int) (int64, error) {
	if region.start < 0 || region.end <= region.start {
		return 0, fmt.Errorf("invalid framebuffer memory range %#x-%#x", region.start, region.end)
	}
	if targetSize <= 0 || requiredSize < targetSize {
		return 0, fmt.Errorf("invalid framebuffer sizes: target=%d required=%d", targetSize, requiredSize)
	}

	var offset int64
	length := minHeaderLength
	for iteration := 0; length < targetSize; iteration++ {
		if iteration >= maxHeaderIterations {
			return 0, fmt.Errorf("framebuffer discovery exceeded %d header iterations", maxHeaderIterations)
		}

		offset += int64(length - minHeaderLength)
		headerAddress := region.start + offset + headerReadOffset
		if headerAddress < region.start || headerAddress+headerSize > region.end {
			return 0, fmt.Errorf("framebuffer header address %#x is outside %#x-%#x", headerAddress, region.start, region.end)
		}

		var header [headerSize]byte
		if _, err := r.ReadAt(header[:], headerAddress); err != nil {
			return 0, fmt.Errorf("read framebuffer header at %#x: %w", headerAddress, err)
		}
		length = int(binary.LittleEndian.Uint32(header[:4]))
		if length < minHeaderLength {
			return 0, fmt.Errorf("invalid framebuffer header length %d at %#x", length, headerAddress)
		}
		if int64(length) > region.end-region.start {
			return 0, fmt.Errorf("framebuffer header length %d exceeds memory range", length)
		}
	}

	framePointer := region.start + offset
	if framePointer < region.start || framePointer+int64(requiredSize) > region.end {
		return 0, fmt.Errorf("framebuffer %#x + %d bytes exceeds %#x-%#x", framePointer, requiredSize, region.start, region.end)
	}
	return framePointer, nil
}
