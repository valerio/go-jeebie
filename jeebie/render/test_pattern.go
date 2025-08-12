package render

import (
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/valerio/go-jeebie/jeebie/video"
)

const (
	// Test pattern constants
	testPatternCount = 4
	targetFPS        = 60
	animationFrames  = 30
	
	// Pattern generation constants
	checkerboardTileSize = 8
	stripeWidth          = 4
	diagonalTileSize     = 8
	
	// Display positioning
	displayOffsetX = 5
	displayOffsetY = 2
	verticalScale  = 2  // Skip every other line
	
	// Color thresholds for shade mapping
	shade1Threshold = 64
	shade2Threshold = 128
	shade3Threshold = 192
	maxColorValue   = 255
	
	// Animation speeds
	stripeAnimationSpeed   = 2
	diagonalAnimationSpeed = 4
)

// RunTestPattern displays a test pattern to verify the rendering pipeline
func RunTestPattern() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}

	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	screen.Clear()

	slog.Info("Starting test pattern display")

	// Create a test framebuffer
	fb := video.NewFrameBuffer()
	
	// Fill with checkerboard pattern
	for y := 0; y < video.FramebufferHeight; y++ {
		for x := 0; x < video.FramebufferWidth; x++ {
			var color uint32
			if ((x/checkerboardTileSize)+(y/checkerboardTileSize))%2 == 0 {
				color = uint32(video.WhiteColor)
			} else {
				color = uint32(video.BlackColor)
			}
			fb.SetPixel(uint(x), uint(y), video.GBColor(color))
		}
	}

	// Main loop
	running := true
	patternType := 0
	frameCount := 0
	
	go func() {
		for running {
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					running = false
					return
				case tcell.KeyRune:
					switch ev.Rune() {
					case ' ':
						// Cycle through patterns
						patternType = (patternType + 1) % testPatternCount
						updatePattern(fb, patternType)
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}()

	ticker := time.NewTicker(time.Second / targetFPS)
	defer ticker.Stop()

	for running {
		select {
		case <-ticker.C:
			frameCount++
			
			// Animate the pattern
			if frameCount%animationFrames == 0 {
				animatePattern(fb, patternType, frameCount/animationFrames)
			}
			
			// Draw the framebuffer
			drawTestFramebuffer(screen, fb)
			
			// Draw info text
			termWidth, termHeight := screen.Size()
			info := "Test Pattern Mode - Press SPACE to change pattern, ESC to exit"
			for i, ch := range info {
				if i < termWidth {
					screen.SetContent(i, termHeight-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorYellow))
				}
			}
			
			patternName := []string{"Checkerboard", "Gradient", "Stripes", "Noise"}[patternType]
			status := "Pattern: " + patternName + " | Frame: " + string(rune(frameCount))
			for i, ch := range status {
				if i < termWidth {
					screen.SetContent(i, 0, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorGreen))
				}
			}
			
			screen.Show()
		}
	}

	return nil
}

func drawTestFramebuffer(screen tcell.Screen, fb *video.FrameBuffer) {
	frame := fb.ToSlice()
	
	for y := 0; y < video.FramebufferHeight; y++ {
		for x := 0; x < video.FramebufferWidth; x++ {
			pixel := frame[y*video.FramebufferWidth+x]
			
			// Convert to shade character
			shade := 0
			switch pixel {
			case uint32(video.BlackColor):
				shade = 0
			case uint32(video.DarkGreyColor):
				shade = 1
			case uint32(video.LightGreyColor):
				shade = 2
			case uint32(video.WhiteColor):
				shade = 3
			default:
				// For test pattern, map intermediate values
				r := (pixel >> 24) & 0xFF
				if r < shade1Threshold {
					shade = 0
				} else if r < shade2Threshold {
					shade = 1
				} else if r < shade3Threshold {
					shade = 2
				} else {
					shade = 3
				}
			}
			
			char := shadeChars[shade]
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			
			// Draw at position with some offset for visibility
			screenX := x + displayOffsetX
			screenY := y/verticalScale + displayOffsetY // Compress vertically for terminal
			
			if y%verticalScale == 0 { // Skip every other line for terminal aspect ratio
				screen.SetContent(screenX, screenY, char, nil, style)
			}
		}
	}
}

func updatePattern(fb *video.FrameBuffer, patternType int) {
	switch patternType {
	case 0: // Checkerboard
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color uint32
				if ((x/checkerboardTileSize)+(y/checkerboardTileSize))%2 == 0 {
					color = uint32(video.WhiteColor)
				} else {
					color = uint32(video.BlackColor)
				}
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	case 1: // Gradient
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				gray := uint32(x * maxColorValue / video.FramebufferWidth)
				color := (gray << 24) | (gray << 16) | (gray << 8) | maxColorValue
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	case 2: // Vertical stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color uint32
				if (x/stripeWidth)%2 == 0 {
					color = uint32(video.WhiteColor)
				} else {
					color = uint32(video.DarkGreyColor)
				}
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	case 3: // Diagonal lines
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color uint32
				if ((x+y)/diagonalTileSize)%2 == 0 {
					color = uint32(video.LightGreyColor)
				} else {
					color = uint32(video.DarkGreyColor)
				}
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	}
}

func animatePattern(fb *video.FrameBuffer, patternType int, frame int) {
	// Simple animation based on pattern type
	switch patternType {
	case 2: // Animate stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color uint32
				if ((x+frame*stripeAnimationSpeed)/stripeWidth)%2 == 0 {
					color = uint32(video.WhiteColor)
				} else {
					color = uint32(video.DarkGreyColor)
				}
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	case 3: // Animate diagonal
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color uint32
				if ((x+y+frame*diagonalAnimationSpeed)/diagonalTileSize)%2 == 0 {
					color = uint32(video.LightGreyColor)
				} else {
					color = uint32(video.DarkGreyColor)
				}
				fb.SetPixel(uint(x), uint(y), video.GBColor(color))
			}
		}
	}
}