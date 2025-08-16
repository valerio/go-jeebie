package display

// RGBA pixel format constants
const (
	// RGBABytesPerPixel is the number of bytes per pixel in RGBA format
	RGBABytesPerPixel = 4
	// RGBARShift is the bit shift for the red component in RGBA format
	RGBARShift = 24
	// RGBAGShift is the bit shift for the green component in RGBA format
	RGBAGShift = 16
	// RGBABShift is the bit shift for the blue component in RGBA format
	RGBABShift = 8
	// RGBAColorMask is the mask for extracting color components
	RGBAColorMask = 0xFF
)

// Backend scaling and window constants
const (
	// DefaultPixelScale is the default scaling factor for Game Boy pixels
	DefaultPixelScale = 4
	// DefaultWindowWidth is the default window width (GameBoy width * scale)
	DefaultWindowWidth = 160 * DefaultPixelScale // 640
	// DefaultWindowHeight is the default window height (GameBoy height * scale)
	DefaultWindowHeight = 144 * DefaultPixelScale // 576
)

// Test pattern constants
const (
	// TestPatternCount is the number of available test patterns
	TestPatternCount = 4
	// TestPatternTileSize is the size of tiles for checkerboard and diagonal patterns
	TestPatternTileSize = 8
	// TestPatternStripeWidth is the width of stripes in the stripe pattern
	TestPatternStripeWidth = 4
	// TestPatternAnimationFrames is the number of frames between test pattern animations
	TestPatternAnimationFrames = 30
	// TestPatternStripeSpeed is the animation speed for stripe patterns
	TestPatternStripeSpeed = 2
	// TestPatternDiagonalSpeed is the animation speed for diagonal patterns
	TestPatternDiagonalSpeed = 4
)

// Color mapping constants
const (
	// GrayscaleWhite is the RGB value for white in grayscale
	GrayscaleWhite = 255
	// GrayscaleLightGray is the RGB value for light gray in grayscale
	GrayscaleLightGray = 170
	// GrayscaleDarkGray is the RGB value for dark gray in grayscale
	GrayscaleDarkGray = 85
	// GrayscaleBlack is the RGB value for black in grayscale
	GrayscaleBlack = 0
	// FullAlpha is the alpha value for fully opaque pixels
	FullAlpha = 255
)
