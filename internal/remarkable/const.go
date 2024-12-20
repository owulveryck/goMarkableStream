//go:build !arm64

package remarkable

const (
	Model = Remarkable2

	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404

	ScreenSize = ScreenWidth * ScreenHeight * 2

	// These values are from Max values of /dev/input/event1 (ABS_X and ABS_Y)
	MaxXValue = 15725
	MaxYValue = 20966

	PenInputDevice   = "/dev/input/event1"
	TouchInputDevice = "/dev/input/event2"
)
