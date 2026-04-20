package remarkable

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrXochitlNotFound = errors.New("xochitl process not found - is the reMarkable software running?")

func findXochitlPID() (string, error) {
	base := "/proc"
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		pid := entry.Name()
		if !entry.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(base, entry.Name()))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				orig, err := os.Readlink(filepath.Join(base, pid, entry.Name()))
				if err != nil {
					continue
				}
				if orig == "/usr/bin/xochitl" {
					return pid, nil
				}
			}
		}
	}
	return "", ErrXochitlNotFound
}
