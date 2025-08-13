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

// RenderFrameToHalfBlocks converts a frame buffer to half-block text representation
// Returns a slice of strings, one per text row (72 rows for 144 pixel rows)
func RenderFrameToHalfBlocks(frame []uint32, width, height int) []string {
	if len(frame) < width*height {
		// Handle incomplete frame buffer
		return []string{}
	}

	textHeight := height / 2
	if height%2 != 0 {
		textHeight++ // Add extra row if height is odd
	}

	lines := make([]string, textHeight)

	// Process two pixel rows at a time
	for textRow := 0; textRow < textHeight; textRow++ {
		line := make([]rune, width)

		for x := 0; x < width; x++ {
			pixelRow1 := textRow * 2
			pixelRow2 := pixelRow1 + 1

			// Get top pixel
			topPixel := uint32(0xFFFFFFFF) // Default to white
			if pixelRow1 < height {
				topPixel = frame[pixelRow1*width+x]
			}

			// Get bottom pixel
			bottomPixel := uint32(0xFFFFFFFF) // Default to white
			if pixelRow2 < height {
				bottomPixel = frame[pixelRow2*width+x]
			}

			// Convert to shades
			topShade := PixelToShade(topPixel)
			bottomShade := PixelToShade(bottomPixel)

			// Get appropriate character
			line[x] = GetHalfBlockChar(topShade, bottomShade)
		}

		lines[textRow] = string(line)
	}

	return lines
}
