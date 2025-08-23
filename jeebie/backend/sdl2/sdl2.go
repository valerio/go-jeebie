//go:build sdl2

package sdl2

import (
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/valerio/go-jeebie/jeebie/audio"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/display"
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
	window        *sdl.Window
	renderer      *sdl.Renderer
	texture       *sdl.Texture
	running       bool
	config        backend.BackendConfig
	debugProvider backend.DebugDataProvider // For extracting debug data

	// Test pattern state
	testPatternFrame *video.FrameBuffer
	testPatternType  int
	testFrameCount   int

	// Snapshot state
	currentFrame *video.FrameBuffer

	// Debug window
	debugWindow *DebugWindow

	// Audio
	audioDevice   sdl.AudioDeviceID
	audioProvider audio.Provider

	pixelBuffer []byte
	eventBuffer []backend.InputEvent
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
	s.debugProvider = config.DebugProvider
	s.audioProvider = config.AudioProvider

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS | sdl.INIT_AUDIO); err != nil {
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

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
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

	// Show the window
	s.window.Show()

	// Pre-allocate pixel buffer for rendering
	s.pixelBuffer = make([]byte, video.FramebufferWidth*video.FramebufferHeight*display.RGBABytesPerPixel)

	// Pre-allocate event buffer with reasonable capacity
	s.eventBuffer = make([]backend.InputEvent, 0, 10)

	s.running = true

	// Initialize audio if AudioProvider is available and not in test pattern mode
	if s.audioProvider != nil && !config.TestPattern {
		if err := s.initAudio(); err != nil {
			slog.Warn("Failed to initialize audio", "error", err)
		}
	}

	// Initialize debug window if ShowDebug is enabled
	if config.ShowDebug {
		if err := s.debugWindow.Init(); err != nil {
			slog.Warn("Failed to initialize debug window", "error", err)
		}
	}

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
func (s *Backend) Update(frame *video.FrameBuffer) ([]backend.InputEvent, error) {
	s.eventBuffer = s.eventBuffer[:0]

	// Collect events directly while processing SDL events
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		if inputEvents := s.handleEvent(event); inputEvents != nil {
			s.eventBuffer = append(s.eventBuffer, inputEvents...)
		}
	}

	if !s.running {
		return s.eventBuffer, nil
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

	// Queue audio samples if available
	if s.audioDevice != 0 && s.audioProvider != nil {
		s.queueAudioSamples()
	}

	return s.eventBuffer, nil
}

// Cleanup cleans up SDL2 resources
func (s *Backend) Cleanup() error {
	slog.Info("Cleaning up SDL2 backend")

	if s.audioDevice != 0 {
		sdl.CloseAudioDevice(s.audioDevice)
	}
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

func (s *Backend) handleEvent(evt sdl.Event) []backend.InputEvent {
	// Pass event to debug window first
	if s.debugWindow != nil {
		s.debugWindow.ProcessEvent(evt)
	}

	switch e := evt.(type) {
	case *sdl.QuitEvent:
		s.running = false
		return []backend.InputEvent{{Action: action.EmulatorQuit, Type: event.Press}}

	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			return s.handleKeyDown(e.Keysym.Sym, e.Repeat)
		} else if e.Type == sdl.KEYUP {
			return s.handleKeyUp(e.Keysym.Sym)
		}
	}

	return nil
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

	// Audio debugging
	sdl.K_F1: action.AudioToggleChannel1,
	sdl.K_F2: action.AudioToggleChannel2,
	sdl.K_F3: action.AudioToggleChannel3,
	sdl.K_F4: action.AudioToggleChannel4,
	sdl.K_F5: action.AudioSoloChannel1,
	sdl.K_F6: action.AudioSoloChannel2,
	sdl.K_F7: action.AudioSoloChannel3,
	sdl.K_F8: action.AudioSoloChannel4,
	sdl.K_d:  action.AudioShowStatus,

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

// saveSnapshot takes a screenshot
func (s *Backend) saveSnapshot() {
	debug.TakeSnapshot(s.currentFrame, s.config.TestPattern, s.testPatternType)
}

// cycleTestPattern switches to the next test pattern
func (s *Backend) cycleTestPattern() {
	if s.config.TestPattern {
		s.testPatternType = (s.testPatternType + 1) % display.TestPatternCount
		s.generateTestPattern(s.testPatternType)
		patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
		slog.Info("Switched to test pattern", "pattern", patternNames[s.testPatternType])
	}
}

func (s *Backend) handleKeyDown(key sdl.Keycode, repeat uint8) []backend.InputEvent {
	if act, exists := keyMapping[key]; exists {
		// For initial press, send Press event
		// For held keys (repeat > 0), send Hold event
		if repeat == 0 {
			return []backend.InputEvent{{Action: act, Type: event.Press}}
		} else {
			// Generate Hold event for held keys (not debounced)
			return []backend.InputEvent{{Action: act, Type: event.Hold}}
		}
	}
	return nil
}

func (s *Backend) handleKeyUp(key sdl.Keycode) []backend.InputEvent {
	if act, exists := keyMapping[key]; exists {
		// Only trigger Release events for Game Boy controls
		switch act {
		case action.GBButtonA, action.GBButtonB, action.GBButtonStart, action.GBButtonSelect,
			action.GBDPadUp, action.GBDPadDown, action.GBDPadLeft, action.GBDPadRight:
			return []backend.InputEvent{{Action: act, Type: event.Release}}
		}
	}
	return nil
}

func (s *Backend) renderFrame(frame *video.FrameBuffer) {
	frameData := frame.ToSlice()

	for y := 0; y < video.FramebufferHeight; y++ {
		for x := 0; x < video.FramebufferWidth; x++ {
			srcIdx := y*video.FramebufferWidth + x
			dstIdx := srcIdx * display.RGBABytesPerPixel

			gbPixel := frameData[srcIdx]
			r, g, b, a := s.gbColorToRGBA(gbPixel)

			// ABGR byte order for little-endian RGBA8888
			s.pixelBuffer[dstIdx] = byte(a)   // Alpha (first byte)
			s.pixelBuffer[dstIdx+1] = byte(b) // Blue
			s.pixelBuffer[dstIdx+2] = byte(g) // Green
			s.pixelBuffer[dstIdx+3] = byte(r) // Red (last byte)
		}
	}

	// Update texture with SDL2 pixel data
	s.texture.Update(nil, unsafe.Pointer(&s.pixelBuffer[0]), video.FramebufferWidth*display.RGBABytesPerPixel)

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
	red := uint8((gbColor >> display.RGBARShift) & display.RGBAColorMask)
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

// HandleBackendAction processes backend-specific actions after debouncing
func (s *Backend) HandleBackendAction(act action.Action) {
	switch act {
	case action.EmulatorSnapshot:
		s.saveSnapshot()
	case action.EmulatorTestPatternCycle:
		if s.config.TestPattern {
			s.cycleTestPattern()
		}
	case action.EmulatorDebugToggle:
		s.ToggleDebugWindow()
	case action.EmulatorDebugUpdate:
		// Update debug window data if it's visible
		if s.debugWindow != nil && s.debugWindow.IsVisible() && s.debugProvider != nil {
			debugData := s.debugProvider.ExtractDebugData()
			if debugData != nil && debugData.OAM != nil && debugData.VRAM != nil {
				s.UpdateDebugData(debugData.OAM, debugData.VRAM)
			}
		}
	// Audio debugging actions
	case action.AudioToggleChannel1:
		if s.audioProvider != nil {
			s.audioProvider.ToggleChannel(1)
			s.logAudioStatus("Toggled channel 1")
		}
	case action.AudioToggleChannel2:
		if s.audioProvider != nil {
			s.audioProvider.ToggleChannel(2)
			s.logAudioStatus("Toggled channel 2")
		}
	case action.AudioToggleChannel3:
		if s.audioProvider != nil {
			s.audioProvider.ToggleChannel(3)
			s.logAudioStatus("Toggled channel 3")
		}
	case action.AudioToggleChannel4:
		if s.audioProvider != nil {
			s.audioProvider.ToggleChannel(4)
			s.logAudioStatus("Toggled channel 4")
		}
	case action.AudioSoloChannel1:
		if s.audioProvider != nil {
			s.audioProvider.SoloChannel(1)
			s.logAudioStatus("Solo channel 1")
		}
	case action.AudioSoloChannel2:
		if s.audioProvider != nil {
			s.audioProvider.SoloChannel(2)
			s.logAudioStatus("Solo channel 2")
		}
	case action.AudioSoloChannel3:
		if s.audioProvider != nil {
			s.audioProvider.SoloChannel(3)
			s.logAudioStatus("Solo channel 3")
		}
	case action.AudioSoloChannel4:
		if s.audioProvider != nil {
			s.audioProvider.SoloChannel(4)
			s.logAudioStatus("Solo channel 4")
		}
	case action.AudioShowStatus:
		if s.audioProvider != nil {
			s.logAudioStatus("Audio status")
		}
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
	if !wasVisible {
		slog.Debug("Triggering debug data update")
		s.handleDebugMessage("debug:update_window")
	}
}

// handleDebugMessage processes debug messages internally
func (s *Backend) handleDebugMessage(message string) {
	switch message {
	case "debug:toggle_window":
		s.ToggleDebugWindow()
	case "debug:update_window":
		// Extract debug data through the minimal interface
		if s.debugProvider != nil {
			debugData := s.debugProvider.ExtractDebugData()
			if debugData != nil && debugData.OAM != nil && debugData.VRAM != nil {
				slog.Debug("Extracted debug data", "oam_entries", len(debugData.OAM.Sprites))
				s.UpdateDebugData(debugData.OAM, debugData.VRAM)
			}
		}
	case "debug:snapshot":
		// Handle snapshot - we already have currentFrame
		if s.currentFrame != nil {
			s.saveSnapshot()
		}
	case "debug:cycle_test_pattern":
		if s.config.TestPattern {
			s.cycleTestPattern()
		}
	default:
		// Ignore unhandled messages
	}
}

// logAudioStatus logs the current audio channel status
func (s *Backend) logAudioStatus(message string) {
	if s.audioProvider == nil {
		return
	}
	ch1, ch2, ch3, ch4 := s.audioProvider.GetChannelStatus()
	slog.Info(message,
		"ch1", ch1,
		"ch2", ch2,
		"ch3", ch3,
		"ch4", ch4,
	)
}

// queueAudioSamples gets samples from audio provider and queues them for playback
func (s *Backend) queueAudioSamples() {
	if s.audioProvider == nil || s.audioDevice == 0 {
		return
	}

	// Get queued audio size and queue more if needed
	queuedBytes := sdl.GetQueuedAudioSize(s.audioDevice)
	const targetBytes = 2048 * 4 // Target ~2048 stereo samples

	if queuedBytes < targetBytes {
		samplesToGet := (targetBytes - queuedBytes) / 4
		samples := s.audioProvider.GetSamples(int(samplesToGet))

		if len(samples) > 0 {
			// Convert mono to stereo
			stereoSamples := make([]int16, len(samples)*2)
			for i, sample := range samples {
				stereoSamples[i*2] = sample
				stereoSamples[i*2+1] = sample
			}

			// Queue the audio
			sliceHeader := (*[1 << 30]byte)(unsafe.Pointer(&stereoSamples[0]))[: len(stereoSamples)*2 : len(stereoSamples)*2]
			sdl.QueueAudio(s.audioDevice, sliceHeader)
		}
	}
}

// initAudio initializes SDL2 audio subsystem
func (s *Backend) initAudio() error {
	spec := &sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_S16LSB,
		Channels: 2,
		Samples:  512,
	}

	obtained := &sdl.AudioSpec{}
	audioDevice, err := sdl.OpenAudioDevice("", false, spec, obtained, 0)
	if err != nil {
		return fmt.Errorf("failed to open audio device: %v", err)
	}

	s.audioDevice = audioDevice
	sdl.PauseAudioDevice(s.audioDevice, false)

	slog.Info("Audio initialized", "freq", obtained.Freq, "samples", obtained.Samples)
	return nil
}
