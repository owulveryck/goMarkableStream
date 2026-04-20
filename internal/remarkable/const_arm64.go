//go:build arm64

package remarkable

const (
	Model = RemarkablePaperPro

	// ScreenWidth of the remarkable paper pro
	ScreenWidth = 1632
	// ScreenHeight of the remarkable paper pro
	ScreenHeight = 2154

	ScreenSizeBytes = ScreenWidth * ScreenHeight * 4

	// GPU Tile Size: DRI driver allocates framebuffer memory in fixed-size tiles
	// Observed from /proc/[pid]/maps: each /dev/dri/card0 mapping = 1,757,184 bytes
	// This value is stable across firmware versions, unlike ScreenSizeBytes which
	// caused infinite loops when firmware changed memory layout.
	// Used by calculateFramePointer() for robust framebuffer detection.
	GPUTileSize = 1757184

	// These values are from Max values of /dev/input/event2 (ABS_X and ABS_Y)
	MaxXValue = 11180
	MaxYValue = 15340

	PenInputDevice   = "/dev/input/event2"
	TouchInputDevice = "/dev/input/event3"
)
