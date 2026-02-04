//go:build linux && !arm64

package remarkable

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// FramebufferFormat represents the type of framebuffer format in use
type FramebufferFormat int

const (
	// FormatLegacy is the pre-3.24 firmware format (gray16le, 1872x1404, 2 bytes/pixel)
	FormatLegacy FramebufferFormat = iota
	// FormatNew is the 3.24+ firmware format (ABGR32, 1404x1872, 4 bytes/pixel)
	FormatNew
)

const (
	// Firmware version file location
	firmwareVersionPath = "/etc/version"

	// New format constants (firmware 3.24+)
	newFormatWidth         = 1404
	newFormatHeight        = 1872
	newFormatBytesPerPixel = 4
	newFormatPointerOffset = int64(2629632)
)

// DetectFirmwareFormat detects which framebuffer format is in use based on firmware version.
// Returns FormatNew for firmware >= 3.24, FormatLegacy otherwise.
func DetectFirmwareFormat() FramebufferFormat {
	major, minor, err := parseFirmwareVersion(firmwareVersionPath)
	if err != nil {
		log.Printf("Could not detect firmware version: %v, using legacy format", err)
		return FormatLegacy
	}

	log.Printf("Detected firmware version: %d.%d", major, minor)

	// Check if firmware is >= 3.24
	if major > 3 || (major == 3 && minor >= 24) {
		return FormatNew
	}

	return FormatLegacy
}

// parseFirmwareVersion reads and parses the firmware version from the given file path.
// The version file typically contains a string like "3.24.0.1234"
func parseFirmwareVersion(path string) (major, minor int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ".")
		if len(parts) >= 2 {
			major, err = strconv.Atoi(parts[0])
			if err != nil {
				return 0, 0, err
			}
			minor, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, 0, err
			}
			return major, minor, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return 0, 0, os.ErrNotExist
}

// initConfigForFirmware updates the runtime Config based on detected firmware format.
// This should be called during initialization on RM2 devices.
func initConfigForFirmware() {
	format := DetectFirmwareFormat()

	switch format {
	case FormatNew:
		log.Println("Using new framebuffer format (firmware 3.24+)")
		Config = FramebufferConfig{
			Width:          newFormatWidth,
			Height:         newFormatHeight,
			BytesPerPixel:  newFormatBytesPerPixel,
			SizeBytes:      newFormatWidth * newFormatHeight * newFormatBytesPerPixel,
			PointerOffset:  newFormatPointerOffset,
			UseBGRA:        true,
			TextureFlipped: true,
		}
	case FormatLegacy:
		log.Println("Using legacy framebuffer format (pre-3.24)")
		Config = FramebufferConfig{
			Width:          ScreenWidth,
			Height:         ScreenHeight,
			BytesPerPixel:  2,
			SizeBytes:      ScreenWidth * ScreenHeight * 2,
			PointerOffset:  0,
			UseBGRA:        false,
			TextureFlipped: false,
		}
	}
}
