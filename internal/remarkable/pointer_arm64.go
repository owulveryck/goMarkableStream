//go:build linux && arm64

package remarkable

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

	// Iterate to calculate the correct offset within the frame buffer memory
	// The memory header contains a length field (4 bytes) which we use to determine
	// how much memory to skip. We dynamically calculate the offset until the
	// buffer size (width x height x 4 bytes per pixel) is reached.
	for length < ScreenSizeBytes {
		offset += int64(length - 2)

		// Seek to the start address plus offset and read the header
		// The header helps identify the size of the subsequent memory block.
		file.Seek(startAddress+offset+8, 0)
		header := make([]byte, 8)
		_, err := file.Read(header)
		if err != nil {
			return 0, fmt.Errorf("error reading memory header: %w", err)
		}

		// Extract the length from the header (4 bytes at the beginning of the header)
		length = int(int64(header[0]) | int64(header[1])<<8 | int64(header[2])<<16 | int64(header[3])<<24)
	}

	// Return the calculated frame pointer address
	return startAddress + offset, nil
}
