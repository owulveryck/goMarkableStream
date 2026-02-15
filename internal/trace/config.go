//go:build trace

package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config holds tracing configuration
type Config struct {
	Enabled   bool
	Mode      string // "runtime", "spans", or "both"
	Dir       string
	MaxSizeMB int
	MaxFiles  int
	AutoStart bool
}

// FileInfo represents information about a trace file
type FileInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Type         string    `json:"type"` // "runtime" or "spans"
	SizeReadable string    `json:"size_readable"`
}

var config Config

// ensureDir creates the trace directory if it doesn't exist
func ensureDir() error {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create trace directory: %w", err)
	}
	return nil
}

// generateFilename creates a timestamped filename for trace files
func generateFilename(traceType string) string {
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	ext := ".trace"
	if traceType == "spans" {
		ext = ".jsonl"
	}
	return filepath.Join(config.Dir, fmt.Sprintf("%s-%s%s", traceType, timestamp, ext))
}

// listTraceFiles returns all trace files in the directory
func listTraceFiles() ([]FileInfo, error) {
	entries, err := os.ReadDir(config.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only include trace and jsonl files
		if !strings.HasSuffix(name, ".trace") && !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		traceType := "runtime"
		if strings.HasSuffix(name, ".jsonl") {
			traceType = "spans"
		}

		files = append(files, FileInfo{
			Name:         name,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			Type:         traceType,
			SizeReadable: formatSize(info.Size()),
		})
	}

	// Sort by modification time, newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	return files, nil
}

// cleanupOldFiles removes old trace files beyond the configured limit
func cleanupOldFiles(traceType string) error {
	files, err := listTraceFiles()
	if err != nil {
		return err
	}

	// Filter by type
	var typeFiles []FileInfo
	for _, f := range files {
		if f.Type == traceType {
			typeFiles = append(typeFiles, f)
		}
	}

	// Remove files beyond the limit
	if len(typeFiles) > config.MaxFiles {
		for i := config.MaxFiles; i < len(typeFiles); i++ {
			path := filepath.Join(config.Dir, typeFiles[i].Name)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove old trace file %s: %w", typeFiles[i].Name, err)
			}
		}
	}

	return nil
}

// checkFileSize returns true if the file exceeds the max size
func checkFileSize(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	maxBytes := int64(config.MaxSizeMB) * 1024 * 1024
	return info.Size() >= maxBytes, nil
}

// formatSize formats bytes into a human-readable size string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
