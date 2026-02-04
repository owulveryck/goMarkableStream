// Package debug provides debug logging functionality controlled by the RK_DEBUG environment variable.
package debug

import "log"

// Enabled controls whether debug logging is active.
var Enabled = false

// Log prints a debug message if debug logging is enabled.
func Log(format string, v ...any) {
	if Enabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}
