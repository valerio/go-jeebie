package video

// SpritePriorityBuffer manages sprite-to-pixel priority assignment in
// DMG (non-color) rendering, see https://gbdev.io/pandocs/OAM.html#drawing-priority.
//
// In this mode, the PPU enforces strict priority rules:
//   - sprites with lower X coordinates have priority
//   - when X coordinates match, lower OAM indices win.
//
// Example 1: overlap with different X coordinates
//
//	Pixels:     0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17
//	Sprite 0:                  [-----A-----]                    (X=5, OAM=0)
//	Sprite 1:                           [-----B-----]           (X=10, OAM=1)
//	Result:                    [-----A-----]--B-----]
//
// Sprite 0 wins all its pixels because it has lower X coordinate.
//
// Example 2: overlap with same X coordinates
//
//	Pixels:    10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25
//	Sprite 1:           [-----D-----]                          (X=12, OAM=1)
//	Sprite 3:           [-----C-----]                          (X=12, OAM=3)
//	Sprite 5:  [-----E-----]                                   (X=10, OAM=5)
//	Result:    [-----E-----]--D-----]
//
// - Pixels 10-17: Sprite 5 wins (lowest X=10, beats both Sprites 1 and 3)
// - Pixels 18-19: Sprite 1 wins (same X=12, lower OAM than Sprite 3)
//
// How the priority buffer works:
//
// Instead of sorting sprites by priority, we use a per-pixel priority model:
//
// 1. Initialize: Clear buffer, marking all pixels as unowned (-1)
// 2. Selection phase: For each sprite (in OAM order):
//
//		For each pixel the sprite covers (8 pixels wide):
//		  	Check current priority holder for that pixel
//	  		If unowned OR this sprite has higher priority:
//	  			Claim the pixel (store sprite index and X coordinate)
//
// 3. Render phase:
//
//		For each sprite:
//	 		Only draw pixels that this sprite has priority for
//	  		Skip transparent pixels and background priority checks
//
// A simpler solution would be to collect sprites by looking at their Y coord
// in a first loop (0 to 40, selection priority), then, before drawing, sorting
// them by (X, OAM index) and drawing in that order. This buffer instead avoids
// sorts by precomputing priority during the selection phase.
type SpritePriorityBuffer struct {
	// priority tracks which sprite (by OAM index) has priority for each pixel
	// -1 means no sprite has priority for this pixel
	priority [FramebufferWidth]int

	// priorityX tracks the X coordinate of the sprite that has priority for each pixel
	// used for priority comparison when multiple sprites overlap
	priorityX [FramebufferWidth]int
}

// Clear resets the buffer for a new scanline
func (s *SpritePriorityBuffer) Clear() {
	for i := range FramebufferWidth {
		s.priority[i] = -1
		s.priorityX[i] = 0xFF // max value ensures any sprite wins initially
	}
}

// TryClaimPixel attempts to claim priority for a pixel for a sprite.
// Returns true if the sprite wins priority and claims the pixel.
// Priority rules:
//  1. If no sprite has priority for the pixel, this sprite wins
//  2. If this sprite has a lower X coordinate, it wins
//  3. If X coordinates match, lower OAM index wins
func (s *SpritePriorityBuffer) TryClaimPixel(pixelX, spriteIndex, spriteX int) bool {
	if pixelX < 0 || pixelX >= FramebufferWidth {
		return false
	}

	current := s.priority[pixelX]

	// 1: no priority holder yet, this sprite wins
	if current == -1 {
		s.priority[pixelX] = spriteIndex
		s.priorityX[pixelX] = spriteX
		return true
	}

	currentX := s.priorityX[pixelX]

	// 2: lower X coordinate wins
	if spriteX < currentX {
		s.priority[pixelX] = spriteIndex
		s.priorityX[pixelX] = spriteX
		return true
	}

	// 3: same X, lower OAM index wins
	if spriteX == currentX && spriteIndex < current {
		s.priority[pixelX] = spriteIndex
		s.priorityX[pixelX] = spriteX
		return true
	}

	return false
}

// GetPriority returns the sprite index that has priority for a pixel, or -1 if none
func (s *SpritePriorityBuffer) GetPriority(pixelX int) int {
	if pixelX < 0 || pixelX >= FramebufferWidth {
		return -1
	}
	return s.priority[pixelX]
}
