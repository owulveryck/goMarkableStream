//go:build arm64 && rmmove

package remarkable

const (
	// The Paper Pro Move uses the same ARM64 framebuffer discovery path as the
	// Paper Pro, with a narrower visible framebuffer and Move-specific input
	// ranges.
	Model = RemarkablePaperProMove

	ScreenWidth  = moveScreenWidth
	ScreenHeight = moveScreenHeight

	ScreenSizeBytes = moveVisibleSizeBytes

	// Chiappa stores each 954-pixel visible row in a 960-pixel-aligned row.
	FramebufferStorageWidth = moveStorageWidth

	MaxXValue = moveMaxXValue
	MaxYValue = moveMaxYValue

	PenInputDevice   = "/dev/input/event2"
	TouchInputDevice = "/dev/input/event3"
)
