//go:build sdl2

package sdl2

import (
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	windowWidth  = display.DefaultWindowWidth
	windowHeight = display.DefaultWindowHeight
	pixelScale   = display.DefaultPixelScale
)

// Backend implements the Backend interface using SDL2 bindings
// Note: building this requires SDL2 development libraries installed.
// Default builds skip this and use a stubbed renderer, see build tags (sdl2)
type Backend struct {
	window    *sdl.Window
	renderer  *sdl.Renderer
	texture   *sdl.Texture
	running   bool
	callbacks backend.BackendCallbacks
	config    backend.BackendConfig

	// Test pattern state
	testPatternFrame *video.FrameBuffer
	testPatternType  int
	testFrameCount   int

	// Snapshot state
	currentFrame *video.FrameBuffer

	// Debug window
	debugWindow *DebugWindow

	// Input management
	inputManager *input.Manager
}

// New creates a new SDL2 backend
func New() *Backend {
	return &Backend{
		debugWindow: NewDebugWindow(),
	}
}

// Init initializes the SDL2 backend
func (s *Backend) Init(config backend.BackendConfig) error {
	s.config = config
	s.callbacks = config.Callbacks
	s.inputManager = config.InputManager

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

	// Initialize debug window if ShowDebug is enabled
	if config.ShowDebug {
		if err := s.debugWindow.Init(); err != nil {
			slog.Warn("Failed to initialize debug window", "error", err)
		}
	}

	// Register SDL2-specific callbacks
	s.setupCallbacks()

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
func (s *Backend) Update(frame *video.FrameBuffer) error {
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

	// Render debug window if visible
	if s.debugWindow != nil {
		s.debugWindow.Render()
	}

	return nil
}

// Cleanup cleans up SDL2 resources
func (s *Backend) Cleanup() error {
	slog.Info("Cleaning up SDL2 backend")

	if s.debugWindow != nil {
		s.debugWindow.Cleanup()
	}
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

func (s *Backend) handleEvent(event sdl.Event) {
	switch e := event.(type) {
	case *sdl.QuitEvent:
		s.running = false
		if s.callbacks.OnQuit != nil {
			s.callbacks.OnQuit()
		}

	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			s.handleKeyDown(e.Keysym.Sym, e.Repeat)
		} else if e.Type == sdl.KEYUP {
			s.handleKeyUp(e.Keysym.Sym)
		}
	}

	// Pass event to debug window
	if s.debugWindow != nil {
		s.debugWindow.ProcessEvent(event)
	}
}

// keyMapping maps SDL2 keys to actions
var keyMapping = map[sdl.Keycode]action.Action{
	// Emulator controls
	sdl.K_F10:    action.EmulatorDebugUpdate,
	sdl.K_F11:    action.EmulatorDebugToggle,
	sdl.K_F12:    action.EmulatorSnapshot,
	sdl.K_ESCAPE: action.EmulatorQuit,
	sdl.K_SPACE:  action.EmulatorPauseToggle,
	sdl.K_t:      action.EmulatorTestPatternCycle,

	// Game Boy controls
	sdl.K_RETURN: action.GBButtonStart,
	sdl.K_a:      action.GBButtonA,
	sdl.K_s:      action.GBButtonB,
	sdl.K_q:      action.GBButtonSelect,
	sdl.K_UP:     action.GBDPadUp,
	sdl.K_DOWN:   action.GBDPadDown,
	sdl.K_LEFT:   action.GBDPadLeft,
	sdl.K_RIGHT:  action.GBDPadRight,
}

func (s *Backend) setupCallbacks() {
	// Register callbacks for actions that need backend-specific handling
	if s.inputManager == nil {
		slog.Warn("No input manager available, callbacks not registered")
		return
	}

	s.inputManager.On(action.EmulatorQuit, event.Press, func() {
		s.running = false
	})

	s.inputManager.On(action.EmulatorDebugToggle, event.Press, func() {
		s.ToggleDebugWindow()
	})
}

func (s *Backend) handleKeyDown(key sdl.Keycode, repeat uint8) {
	// Ignore key repeat events
	if repeat != 0 {
		return
	}

	if act, exists := keyMapping[key]; exists {
		// Handle backend-specific actions that need direct access to backend state
		if act == action.EmulatorSnapshot {
			debug.TakeSnapshot(s.currentFrame, s.config.TestPattern, s.testPatternType)
			return
		}
		if act == action.EmulatorTestPatternCycle && s.config.TestPattern {
			s.testPatternType = (s.testPatternType + 1) % display.TestPatternCount
			s.generateTestPattern(s.testPatternType)
			patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
			slog.Info("Switched to test pattern", "pattern", patternNames[s.testPatternType])
			return
		}
		// Pass all actions to input manager for proper handling and debouncing
		if s.inputManager != nil {
			s.inputManager.Trigger(act, event.Press)
		}
	}
}

func (s *Backend) handleKeyUp(key sdl.Keycode) {
	if s.inputManager == nil {
		return
	}
	if act, exists := keyMapping[key]; exists {
		// Only trigger Release events for Game Boy controls
		switch act {
		case action.GBButtonA, action.GBButtonB, action.GBButtonStart, action.GBButtonSelect,
			action.GBDPadUp, action.GBDPadDown, action.GBDPadLeft, action.GBDPadRight:
			s.inputManager.Trigger(act, event.Release)
		}
	}
}

func (s *Backend) renderFrame(frame *video.FrameBuffer) {
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
func (s *Backend) gbColorToRGBA(gbColor uint32) (r, g, b, a uint8) {
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
func (s *Backend) generateTestPattern(patternType int) {
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
func (s *Backend) animateTestPattern() {
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

// UpdateDebugData updates the debug window with current emulator state
func (s *Backend) UpdateDebugData(oam *debug.OAMData, vram *debug.VRAMData) {
	if s.debugWindow != nil {
		s.debugWindow.UpdateData(oam, vram)
	}
}

// ToggleDebugWindow shows/hides the debug window
func (s *Backend) ToggleDebugWindow() {
	if s.debugWindow == nil {
		slog.Warn("Debug window is nil")
		return
	}

	if !s.debugWindow.IsInitialized() {
		slog.Debug("Initializing debug window")
		if err := s.debugWindow.Init(); err != nil {
			slog.Warn("Failed to initialize debug window", "error", err)
			return
		}
	}
	wasVisible := s.debugWindow.IsVisible()
	s.debugWindow.SetVisible(!wasVisible)
	slog.Debug("Debug window visibility changed", "was_visible", wasVisible, "now_visible", !wasVisible)

	// If we're showing the window, trigger a debug data update
	if !wasVisible && s.callbacks.OnDebugMessage != nil {
		slog.Debug("Triggering debug data update")
		s.callbacks.OnDebugMessage("debug:update_window")
	}
}
