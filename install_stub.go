//go:build !(linux && (arm || arm64))

package main

import "fmt"

func runInstall() error {
	return fmt.Errorf("install command is only available on Linux ARM/ARM64 (reMarkable devices)")
}
