//go:build linux && arm64

package remarkable

import (
	"fmt"
	"os"
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
	memory, err := getMemoryRange(pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get memory range: %w", err)
	}

	// Calculate the correct starting address
	framePointer, err := calculateFramePointer(pid, memory)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate frame pointer: %w", err)
	}

	return framePointer, nil
}

// getMemoryRange retrieves the readable region immediately following the last
// /dev/dri/card0 mapping from /proc/[pid]/maps.
func getMemoryRange(pid string) (memoryRange, error) {
	mapsFilePath := fmt.Sprintf("/proc/%s/maps", pid)
	file, err := os.Open(mapsFilePath)
	if err != nil {
		return memoryRange{}, fmt.Errorf("cannot open maps file: %w", err)
	}
	defer file.Close()

	return parseFramebufferMemoryRange(file)
}

// calculateFramePointer finds the frame pointer using the end address and memory file
func calculateFramePointer(pid string, region memoryRange) (int64, error) {
	memFilePath := fmt.Sprintf("/proc/%s/mem", pid)
	file, err := os.Open(memFilePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open memory file: %w", err)
	}
	defer file.Close()

	requiredSize := FramebufferStorageWidth * ScreenHeight * BytesPerPixelBGRA
	return calculateFramePointerFrom(file, region, ScreenSizeBytes, requiredSize)
}
