//go:build linux && arm64

package remarkable

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// Maximum iterations to prevent infinite loops
	// Should never reach this in normal operation
	maxHeaderIterations = 100

	// Minimum valid header length
	minValidHeaderLength = 1024
)

// getFramePointer locates the framebuffer in memory for RMPP.
//
// RMPP uses a modern GPU/DRM display stack (/dev/dri/card0) rather than
// the classic framebuffer device. This requires a more complex algorithm:
// 1. Find the last /dev/dri/card0 mapping in /proc/[pid]/maps
// 2. Read memory headers to dynamically calculate the buffer offset
// 3. Iterate until the correct buffer size is found
//
// This differs from RM2's simpler /dev/fb0 approach due to the GPU architecture.
// Both devices now use BGRA format, but the underlying hardware architecture
// necessitates different pointer detection methods.
func getFramePointer(pid string) (int64, error) {
	// Find the memory range for the framebuffer
	startAddress, err := getMemoryRange(pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get memory range: %w", err)
	}

	// Calculate the correct starting address
	framePointer, err := calculateFramePointer(pid, startAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate frame pointer: %w", err)
	}

	return framePointer, nil
}

// getMemoryRange retrieves the end address of the last /dev/dri/card0 entry from /proc/[pid]/maps
func getMemoryRange(pid string) (int64, error) {
	mapsFilePath := fmt.Sprintf("/proc/%s/maps", pid)
	file, err := os.Open(mapsFilePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open maps file: %w", err)
	}
	defer file.Close()

	var memoryRange string
	scanner := bufio.NewScanner(file)

	// Find the last occurrence of /dev/dri/card0
	// We need the memory mapping for the display, which is located immediately
	// after the last /dev/dri/card0 mapping. Hence, we keep iterating through
	// the file and update memoryRange each time we encounter /dev/dri/card0.
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "/dev/dri/card0") {
			memoryRange = line
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading maps file: %w", err)
	}

	if memoryRange == "" {
		return 0, fmt.Errorf("no mapping found for /dev/dri/card0")
	}

	// Extract the end address of the last /dev/dri/card0 memory range
	// The range is in the format: "start-end permissions offset dev inode pathname"
	fields := strings.Fields(memoryRange)
	rangeField := fields[0]
	startEnd := strings.Split(rangeField, "-")
	if len(startEnd) != 2 {
		return 0, fmt.Errorf("invalid memory range format")
	}

	// We are interested in the end address because the next memory region,
	// starting from this end address, contains the frame buffer we need.
	end, err := strconv.ParseInt(startEnd[1], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse end address: %w", err)
	}

	return end, nil
}

// calculateFramePointer finds the frame pointer using the end address and memory file
func calculateFramePointer(pid string, startAddress int64) (int64, error) {
	memFilePath := fmt.Sprintf("/proc/%s/mem", pid)
	file, err := os.Open(memFilePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open memory file: %w", err)
	}
	defer file.Close()

	var offset int64
	length := 2
	iterationCount := 0

	// FIXED: Use GPU tile size instead of ScreenSizeBytes
	// The DRI driver allocates framebuffer memory in fixed-size tiles of 1,757,184 bytes.
	// Previous code used ScreenSizeBytes (calculated from screen dimensions), which broke
	// when firmware updates changed memory layout. Using the observable tile size makes
	// this robust across firmware versions.
	for length < GPUTileSize {
		iterationCount++
		if iterationCount > maxHeaderIterations {
			return 0, fmt.Errorf("exceeded maximum iterations (%d) searching for framebuffer - memory layout may have changed", maxHeaderIterations)
		}

		offset += int64(length - 2)

		if _, err := file.Seek(startAddress+offset+8, 0); err != nil {
			return 0, fmt.Errorf("failed to seek in memory file at offset %d: %w", offset, err)
		}

		header := make([]byte, 8)
		_, err := file.Read(header)
		if err != nil {
			return 0, fmt.Errorf("error reading memory header at offset %d: %w", offset, err)
		}

		// Extract the length from the header (4 bytes, little-endian)
		length = int(int64(header[0]) | int64(header[1])<<8 | int64(header[2])<<16 | int64(header[3])<<24)

		// Validation: detect corrupt/invalid header values
		if length < 0 {
			return 0, fmt.Errorf("invalid negative header length %d at offset %d", length, offset)
		}
		if length > 0 && length < minValidHeaderLength {
			return 0, fmt.Errorf("suspicious header length %d at offset %d - too small", length, offset)
		}
	}

	return startAddress + offset, nil
}
