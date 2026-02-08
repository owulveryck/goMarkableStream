//go:build !(linux && (arm || arm64))

package main

import "errors"

// runDownload is a stub for non-ARM platforms
func runDownload() error {
	return errors.New("download command is only available on Linux ARM/ARM64 (reMarkable devices)")
}
