//go:build !arm64

package remarkable

const (
	// Model defines the current device model being used
	Model = Remarkable2

	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404

	// ScreenSizeBytes is the total memory size of the screen buffer in bytes
	ScreenSizeBytes = ScreenWidth * ScreenHeight * 2

	// MaxXValue represents the maximum X coordinate value from /dev/input/event1 (ABS_X)
	MaxXValue = 15725
	// MaxYValue represents the maximum Y coordinate value from /dev/input/event1 (ABS_Y)
	MaxYValue = 20966

	// PenInputDevice ...
	PenInputDevice = "/dev/input/event1"
	// TouchInputDevice ...
	TouchInputDevice = "/dev/input/event2"
)
