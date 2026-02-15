//go:build !trace

package trace

import "time"

// Enabled is always false when tracing is disabled
var Enabled bool = false

// BuildTagEnabled indicates tracing was not compiled in
const BuildTagEnabled = false

// Config stub - same structure as real implementation
type Config struct {
	Enabled   bool
	Mode      string
	Dir       string
	MaxSizeMB int
	MaxFiles  int
	AutoStart bool
}

// Status stub - same structure as real implementation
type Status struct {
	Enabled     bool       `json:"enabled"`
	Active      bool       `json:"active"`
	Mode        string     `json:"mode"`
	RuntimeFile string     `json:"runtime_file"`
	RuntimeSize int64      `json:"runtime_size"`
	SpansFile   string     `json:"spans_file"`
	SpansSize   int64      `json:"spans_size"`
	TraceFiles  []FileInfo `json:"trace_files"`
}

// FileInfo stub
type FileInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Type         string    `json:"type"`
	SizeReadable string    `json:"size_readable"`
}

// Span stub - empty struct
type Span struct{}

// No-op function stubs
func Initialize(cfg Config) error                        { return nil }
func Start(mode string) error                            { return nil }
func Stop() ([]string, error)                            { return nil, nil }
func IsActive() bool                                     { return false }
func GetStatus() Status                                  { return Status{} }
func ListFiles() ([]FileInfo, error)                     { return nil, nil }
func DeleteFile(name string) error                       { return nil }
func GetFilePath(name string) (string, error)            { return "", nil }
func BeginSpan(operation string) *Span                   { return nil }
func EndSpan(span *Span, metadata map[string]any)        {}
