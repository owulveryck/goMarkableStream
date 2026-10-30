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
	// Firmware version file location (os-release contains IMG_VERSION)
	firmwareVersionPath = "/etc/os-release"

	// New format constants (firmware 3.24+)
	newFormatWidth         = 1404
	newFormatHeight        = 1872
	newFormatPointerOffset = int64(2629632)
)

func isRemarkable1() bool {
	data, err := os.ReadFile("/proc/device-tree/model")
	if err != nil {
		data, err = os.ReadFile("/sys/devices/soc0/machine")
		if err != nil {
			return false
		}
	}
	model := strings.TrimRight(string(data), "\x00")
	return strings.Contains(model, "Prototype 1") || strings.Contains(model, "reMarkable 1")
}

// DetectFirmwareFormat detects which framebuffer format is in use based on firmware version.
// Returns FormatNew for firmware >= 3.24, FormatLegacy otherwise.
func DetectFirmwareFormat() FramebufferFormat {
	major, minor, err := parseFirmwareVersion(firmwareVersionPath)
	if err != nil {
		log.Printf("Could not detect firmware version from IMG_VERSION in /etc/os-release: %v, using legacy format", err)
		return FormatLegacy
	}

	log.Printf("Detected firmware version: %d.%d", major, minor)

	// Check if firmware is >= 3.24
	if major > 3 || (major == 3 && minor >= 24) {
		return FormatNew
	}

	return FormatLegacy
}

// parseFirmwareVersion reads and parses the firmware version from /etc/os-release.
// It looks for IMG_VERSION="3.24.0.149" and extracts the major.minor version.
func parseFirmwareVersion(path string) (major, minor int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if value, found := strings.CutPrefix(line, "IMG_VERSION="); found {
			// Remove surrounding quotes if present
			value = strings.Trim(value, "\"")
			parts := strings.Split(value, ".")
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
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return 0, 0, os.ErrNotExist
}

// initConfigForFirmware updates the runtime Config based on detected device model and firmware format.
// This is called during initialization on !arm64 devices.
func initConfigForFirmware() {
	if isRemarkable1() {
		log.Println("Detected reMarkable 1")
		Model = Remarkable1
		ScreenWidth = 1408 // 1404 visible + 4 pixels stride padding
		ScreenHeight = 1872
		ScreenSizeBytes = ScreenWidth * ScreenHeight * 2
		MaxXValue = 15725
		MaxYValue = 20966
		PenInputDevice = "/dev/input/event0"
		TouchInputDevice = "/dev/input/event2"
		Config = FramebufferConfig{
			Width:          ScreenWidth,
			Height:         ScreenHeight,
			BytesPerPixel:  2,
			SizeBytes:      ScreenSizeBytes,
			PointerOffset:  0,
			UseBGRA:        false,
			TextureFlipped: false,
		}
		return
	}

	format := DetectFirmwareFormat()

	switch format {
	case FormatNew:
		log.Println("Using new framebuffer format (firmware 3.24+)")
		Config = FramebufferConfig{
			Width:          newFormatWidth,
			Height:         newFormatHeight,
			BytesPerPixel:  BytesPerPixelBGRA,
			SizeBytes:      newFormatWidth * newFormatHeight * BytesPerPixelBGRA,
			PointerOffset:  newFormatPointerOffset,
			UseBGRA:        true,
			TextureFlipped: true,
		}
	case FormatLegacy:
		log.Println("Using legacy framebuffer format (pre-3.24)")
		Config = FramebufferConfig{
			Width:          ScreenWidth,
			Height:         ScreenHeight,
			BytesPerPixel:  BytesPerPixelGray16,
			SizeBytes:      ScreenWidth * ScreenHeight * BytesPerPixelGray16,
			PointerOffset:  0,
			UseBGRA:        false,
			TextureFlipped: false,
		}
	}
}
