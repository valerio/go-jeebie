package render

// SharedRenderUtils contains common rendering utilities for both terminal and snapshot rendering

// PixelToShade converts a pixel value to a shade level (0-3)
func PixelToShade(pixel uint32) int {
	switch pixel {
	case 0x000000FF:
		return 0 // Black
	case 0x4C4C4CFF:
		return 1 // Dark gray
	case 0x989898FF:
		return 2 // Light gray
	case 0xFFFFFFFF:
		return 3 // White
	default:
		return 0
	}
}

// GetHalfBlockChar returns the appropriate half-block character for two shades
// Returns the character and a description of what it represents
func GetHalfBlockChar(topShade, bottomShade int) rune {
	if topShade == bottomShade {
		// Both pixels same shade - use full block
		return '█'
	} else if topShade == 3 && bottomShade != 3 {
		// Top white, bottom not - use lower half block
		return '▄'
	} else if topShade != 3 && bottomShade == 3 {
		// Top not white, bottom white - use upper half block
		return '▀'
	} else {
		// Mixed shades - use upper half block with appropriate colors
		return '▀'
	}
}
