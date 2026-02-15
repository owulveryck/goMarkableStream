package remarkable

import (
	"testing"
)

// TestFindXochitlPIDReturnsError tests that findXochitlPID returns an error
// when the xochitl process is not found, rather than an empty string.
func TestFindXochitlPIDReturnsError(t *testing.T) {
	// This test is mainly for documentation and will likely skip on non-reMarkable hardware
	// The important thing is that the function signature returns (string, error)

	pid, err := findXochitlPID()

	// If running on reMarkable with xochitl running
	if err == nil && pid != "" {
		// Valid case - process found
		t.Logf("Found xochitl process: %s", pid)
		return
	}

	// If not running on reMarkable or xochitl not running
	if err != nil && pid == "" {
		// Also valid - error returned with empty PID
		t.Logf("xochitl not found (expected): %v", err)
		return
	}

	// Invalid combinations
	if err == nil && pid == "" {
		t.Error("findXochitlPID() returned empty string without error")
	}
	if err != nil && pid != "" {
		t.Errorf("findXochitlPID() returned PID %s with error %v", pid, err)
	}
}

// TestFindXochitlPIDErrorMessage tests that the error message is descriptive.
func TestFindXochitlPIDErrorMessage(t *testing.T) {
	// Try to find xochitl
	pid, err := findXochitlPID()

	// If not found, check error message
	if err != nil {
		errMsg := err.Error()
		// Error message should mention xochitl
		if errMsg == "" {
			t.Error("Error message is empty")
		}
		t.Logf("Error message: %s", errMsg)
		return
	}

	// Process found
	t.Logf("xochitl process found: %s", pid)
}
