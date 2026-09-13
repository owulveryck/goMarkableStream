package remarkable

import (
	"fmt"
	"os"
	"strings"
)

const chiappaMachine = "reMarkable Chiappa"

func detectChiappa(machinePath, deviceTreePath string) (string, error) {
	paths := []string{machinePath, deviceTreePath}
	var errors []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		name := strings.TrimSpace(strings.TrimRight(string(data), "\x00"))
		if name == "" {
			errors = append(errors, fmt.Sprintf("%s: empty", path))
			continue
		}
		if !strings.EqualFold(name, chiappaMachine) {
			return name, fmt.Errorf("unsupported device %q; Move build requires %q", name, chiappaMachine)
		}
		return name, nil
	}
	return "", fmt.Errorf("cannot detect device (%s)", strings.Join(errors, "; "))
}
