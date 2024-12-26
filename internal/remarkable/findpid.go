package remarkable

import (
	"log"
	"os"
	"path/filepath"
)

func findXochitlPID() string {
	base := "/proc"
	entries, err := os.ReadDir(base)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		pid := entry.Name()
		if !entry.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(base, entry.Name()))
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				log.Fatal(err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				orig, err := os.Readlink(filepath.Join(base, pid, entry.Name()))
				if err != nil {
					continue
				}
				if orig == "/usr/bin/xochitl" {
					return pid
				}
			}
		}
	}
	return ""
}
