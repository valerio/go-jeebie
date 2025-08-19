//go:build sdl2

package backend

import (
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	windowWidth  = display.DefaultWindowWidth
	windowHeight = display.DefaultWindowHeight
	pixelScale   = display.DefaultPixelScale
)

// SDL2Backend implements the Backend interface using SDL2 bindings
// Note: building this requires SDL2 development libraries installed.
// Default builds skip this and use a stubbed renderer, see build tags (sdl2)
type SDL2Backend struct {
	window    *sdl.Window
	renderer  *sdl.Renderer
	texture   *sdl.Texture
	running   bool
	callbacks BackendCallbacks
	config    BackendConfig

	// Test pattern state
	testPatternFrame *video.FrameBuffer
	testPatternType  int
	testFrameCount   int

	// Snapshot state
	currentFrame *video.FrameBuffer
}

// NewSDL2Backend creates a new SDL2 backend
func NewSDL2Backend() *SDL2Backend {
	return &SDL2Backend{}
}

// Init initializes the SDL2 backend
func (s *SDL2Backend) Init(config BackendConfig) error {
	s.config = config
	s.callbacks = config.Callbacks

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		return fmt.Errorf("failed to initialize SDL2: %v", err)
	}

	window, err := sdl.CreateWindow(
		config.Title,
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		windowWidth,
		windowHeight,
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		sdl.Quit()
		return fmt.Errorf("failed to create window: %v", err)
	}
	s.window = window

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		window.Destroy()
		sdl.Quit()
		return fmt.Errorf("failed to create renderer: %v", err)
	}
	s.renderer = renderer

	// Create texture for Game Boy screen
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_RGBA8888,
		sdl.TEXTUREACCESS_STREAMING,
		video.FramebufferWidth,
		video.FramebufferHeight,
	)
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		sdl.Quit()
		return fmt.Errorf("failed to create texture: %v", err)
	}
	s.texture = texture

	s.running = true

	if config.TestPattern {
		s.testPatternFrame = video.NewFrameBuffer()
		s.generateTestPattern(0)
		slog.Info("SDL2 backend initialized in test pattern mode")
	} else {
		slog.Info("SDL2 backend initialized")
	}

	return nil
}

// Update renders a frame and processes events
func (s *SDL2Backend) Update(frame *video.FrameBuffer) error {
	if !s.running {
		return nil
	}

	// Process SDL events
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		s.handleEvent(event)
	}

	if !s.running {
		return nil
	}

	// Use test pattern frame if in test pattern mode
	renderFrame := frame
	if s.config.TestPattern {
		s.testFrameCount++
		if s.testFrameCount%display.TestPatternAnimationFrames == 0 {
			s.animateTestPattern()
		}
		renderFrame = s.testPatternFrame
	}

	// Store current frame for snapshots and render
	s.currentFrame = renderFrame
	s.renderFrame(renderFrame)

	return nil
}

// Cleanup cleans up SDL2 resources
func (s *SDL2Backend) Cleanup() error {
	slog.Info("Cleaning up SDL2 backend")

	if s.texture != nil {
		s.texture.Destroy()
	}
	if s.renderer != nil {
		s.renderer.Destroy()
	}
	if s.window != nil {
		s.window.Destroy()
	}
	sdl.Quit()

	return nil
}

func (s *SDL2Backend) handleEvent(event sdl.Event) {
	switch e := event.(type) {
	case *sdl.QuitEvent:
		s.running = false
		if s.callbacks.OnQuit != nil {
			s.callbacks.OnQuit()
		}

	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			s.handleKeyDown(e.Keysym.Sym)
		} else if e.Type == sdl.KEYUP {
			s.handleKeyUp(e.Keysym.Sym)
		}
	}
}

func (s *SDL2Backend) handleKeyDown(key sdl.Keycode) {
	if s.config.TestPattern {
		switch key {
		case sdl.K_t:
			s.testPatternType = (s.testPatternType + 1) % display.TestPatternCount
			s.generateTestPattern(s.testPatternType)
			patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
			slog.Info("Switched to test pattern", "pattern", patternNames[s.testPatternType])
		case sdl.K_ESCAPE:
			s.running = false
			if s.callbacks.OnQuit != nil {
				s.callbacks.OnQuit()
			}
		case sdl.K_F12:
			debug.TakeSnapshot(s.currentFrame, s.config.TestPattern, s.testPatternType)
		}
		return
	}

	// Normal emulator controls
	switch key {
	case sdl.K_ESCAPE:
		s.running = false
		if s.callbacks.OnQuit != nil {
			s.callbacks.OnQuit()
		}
	case sdl.K_RETURN:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadStart)
		}
	case sdl.K_RIGHT:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadRight)
		}
	case sdl.K_LEFT:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadLeft)
		}
	case sdl.K_UP:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadUp)
		}
	case sdl.K_DOWN:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadDown)
		}
	case sdl.K_a:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadA)
		}
	case sdl.K_s:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadB)
		}
	case sdl.K_q:
		if s.callbacks.OnKeyPress != nil {
			s.callbacks.OnKeyPress(memory.JoypadSelect)
		}
	case sdl.K_SPACE:
		if s.callbacks.OnDebugMessage != nil {
			s.callbacks.OnDebugMessage("debug:toggle_pause")
		}
	case sdl.K_F12:
		debug.TakeSnapshot(s.currentFrame, s.config.TestPattern, s.testPatternType)
	}
}

func (s *SDL2Backend) handleKeyUp(key sdl.Keycode) {
	if s.config.TestPattern {
		return
	}

	// Handle key releases for joypad
	switch key {
	case sdl.K_RIGHT:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadRight)
		}
	case sdl.K_LEFT:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadLeft)
		}
	case sdl.K_UP:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadUp)
		}
	case sdl.K_DOWN:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadDown)
		}
	case sdl.K_a:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadA)
		}
	case sdl.K_s:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadB)
		}
	case sdl.K_q:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadSelect)
		}
	case sdl.K_RETURN:
		if s.callbacks.OnKeyRelease != nil {
			s.callbacks.OnKeyRelease(memory.JoypadStart)
		}
	}
}

func (s *SDL2Backend) renderFrame(frame *video.FrameBuffer) {
	frameData := frame.ToSlice()

	// Convert to ABGR byte order for little-endian RGBA8888
	sdlPixels := make([]byte, video.FramebufferWidth*video.FramebufferHeight*display.RGBABytesPerPixel)

	for y := 0; y < video.FramebufferHeight; y++ {
		for x := 0; x < video.FramebufferWidth; x++ {
			srcIdx := y*video.FramebufferWidth + x
			dstIdx := srcIdx * display.RGBABytesPerPixel

			gbPixel := frameData[srcIdx]
			r, g, b, a := s.gbColorToRGBA(gbPixel)

			// ABGR byte order for little-endian RGBA8888
			sdlPixels[dstIdx] = byte(a)   // Alpha (first byte)
			sdlPixels[dstIdx+1] = byte(b) // Blue
			sdlPixels[dstIdx+2] = byte(g) // Green
			sdlPixels[dstIdx+3] = byte(r) // Red (last byte)
		}
	}

	// Update texture with SDL2 pixel data
	s.texture.Update(nil, unsafe.Pointer(&sdlPixels[0]), video.FramebufferWidth*display.RGBABytesPerPixel)

	// Clear renderer and draw texture scaled up
	s.renderer.SetDrawColor(display.GrayscaleBlack, display.GrayscaleBlack, display.GrayscaleBlack, display.FullAlpha)
	s.renderer.Clear()
	s.renderer.Copy(s.texture, nil, nil)
	s.renderer.Present()
}

// gbColorToRGBA converts a Game Boy color value to RGBA components
func (s *SDL2Backend) gbColorToRGBA(gbColor uint32) (r, g, b, a uint8) {
	// Always map to proper Game Boy grayscale colors first
	switch gbColor {
	case uint32(video.WhiteColor):
		return display.GrayscaleWhite, display.GrayscaleWhite, display.GrayscaleWhite, display.FullAlpha
	case uint32(video.LightGreyColor):
		return display.GrayscaleLightGray, display.GrayscaleLightGray, display.GrayscaleLightGray, display.FullAlpha
	case uint32(video.DarkGreyColor):
		return display.GrayscaleDarkGray, display.GrayscaleDarkGray, display.GrayscaleDarkGray, display.FullAlpha
	case uint32(video.BlackColor):
		return display.GrayscaleBlack, display.GrayscaleBlack, display.GrayscaleBlack, display.FullAlpha
	}

	// For any non-standard colors, extract the red channel and convert to grayscale
	// This handles test pattern gradients properly
	red := uint8((gbColor >> display.RGBARShift) & display.RGBAColorMask)

	// Convert to grayscale by using the red channel value
	return red, red, red, display.FullAlpha
}

// generateTestPattern creates different test patterns
func (s *SDL2Backend) generateTestPattern(patternType int) {
	switch patternType {
	case 0: // Checkerboard
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x/display.TestPatternTileSize)+(y/display.TestPatternTileSize))%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.BlackColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 1: // Gradient
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				// Map x position to one of the 4 Game Boy colors
				colorIndex := x * 4 / video.FramebufferWidth
				var color video.GBColor
				switch colorIndex {
				case 0:
					color = video.BlackColor
				case 1:
					color = video.DarkGreyColor
				case 2:
					color = video.LightGreyColor
				default:
					color = video.WhiteColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 2: // Vertical stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if (x/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Diagonal lines
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}

// animateTestPattern provides simple animation for test patterns
func (s *SDL2Backend) animateTestPattern() {
	frame := s.testFrameCount / display.TestPatternAnimationFrames
	switch s.testPatternType {
	case 2: // Animate stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+frame*display.TestPatternStripeSpeed)/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Animate diagonal
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y+frame*display.TestPatternDiagonalSpeed)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				s.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}
