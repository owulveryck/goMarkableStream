//go:build arm64

package remarkable

const (
	// ScreenWidth of the remarkable paper pro
	ScreenWidth = 2154
	// ScreenHeight of the remarkable paper pro
	ScreenHeight = 1624

	ScreenSize = ScreenWidth * ScreenHeight * 4

	PenInputDevice   = "/dev/input/event2"
	TouchInputDevice = "/dev/input/event3"
)
