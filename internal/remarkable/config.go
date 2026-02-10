package remarkable

const (
	// BytesPerPixelBGRA is the size in bytes for BGRA32 color format.
	// Used by RM2 firmware 3.24+ and RMPP.
	BytesPerPixelBGRA = 4

	// BytesPerPixelGray16 is the size in bytes for 16-bit grayscale format.
	// Used by RM2 firmware versions before 3.24.
	BytesPerPixelGray16 = 2
)

// FramebufferConfig holds runtime configuration for the framebuffer
// that varies based on device model and firmware version.
type FramebufferConfig struct {
	Width          int
	Height         int
	BytesPerPixel  int
	SizeBytes      int
	PointerOffset  int64
	UseBGRA        bool
	TextureFlipped bool
}

// Config holds the runtime framebuffer configuration.
// It is initialized at startup based on device model and firmware version.
var Config FramebufferConfig

func init() {
	// Default to compile-time constants for backward compatibility
	Config = FramebufferConfig{
		Width:          ScreenWidth,
		Height:         ScreenHeight,
		BytesPerPixel:  ScreenSizeBytes / (ScreenWidth * ScreenHeight),
		SizeBytes:      ScreenSizeBytes,
		PointerOffset:  0,
		UseBGRA:        Model == RemarkablePaperPro,
		TextureFlipped: Model == RemarkablePaperPro,
	}
}
