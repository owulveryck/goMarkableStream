//go:build linux && !arm64

package remarkable

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func init() {
	// Detect firmware version and update runtime config
	initConfigForFirmware()
}

// getFramePointer locates the framebuffer in memory for RM2.
//
// RM2 uses the classic Linux framebuffer device (/dev/fb0). This function
// scans /proc/[pid]/maps to find the memory mapping for /dev/fb0, then
// applies the configured PointerOffset to locate the actual pixel data.
//
// For firmware 3.24+, the offset is 2629632 bytes; for legacy firmware it's 0.
// This differs from RMPP's approach which uses the modern GPU/DRM stack.
func getFramePointer(pid string) (int64, error) {
	file, err := os.OpenFile("/proc/"+pid+"/maps", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return 0, fmt.Errorf("cannot open maps file: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanAddr := false
	var addr int64
	for scanner.Scan() {
		if scanAddr {
			hex := strings.Split(scanner.Text(), "-")[0]
			addr, err = strconv.ParseInt("0x"+hex, 0, 64)
			break
		}
		if scanner.Text() == `/dev/fb0` {
			scanAddr = true
		}
	}
	return addr + Config.PointerOffset + 8, err
}
