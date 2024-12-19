//go:build !arm64

package remarkable

const (
	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404

	ScreenSize = ScreenWidth * ScreenHeight * 2

	PenInputDevice   = "/dev/input/event1"
	TouchInputDevice = "/dev/input/event2"
)
