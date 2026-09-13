package remarkable

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type memoryRange struct {
	start int64
	end   int64
}

func parseFramebufferMemoryRange(r io.Reader) (memoryRange, error) {
	var (
		lastDRIEnd int64
		candidate  memoryRange
		foundDRI   bool
	)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		bounds := strings.SplitN(fields[0], "-", 2)
		if len(bounds) != 2 {
			continue
		}
		start, startErr := strconv.ParseInt(bounds[0], 16, 64)
		end, endErr := strconv.ParseInt(bounds[1], 16, 64)
		if startErr != nil || endErr != nil || start >= end {
			continue
		}

		if strings.Contains(line, "/dev/dri/card0") {
			lastDRIEnd = end
			foundDRI = true
			candidate = memoryRange{}
			continue
		}

		if foundDRI && start == lastDRIEnd && strings.HasPrefix(fields[1], "r") {
			candidate = memoryRange{start: start, end: end}
		}
	}
	if err := scanner.Err(); err != nil {
		return memoryRange{}, fmt.Errorf("read maps: %w", err)
	}
	if !foundDRI {
		return memoryRange{}, fmt.Errorf("no mapping found for /dev/dri/card0")
	}
	if candidate.start == 0 {
		return memoryRange{}, fmt.Errorf("no readable mapping immediately follows /dev/dri/card0")
	}
	return candidate, nil
}
