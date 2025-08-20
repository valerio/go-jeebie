package backend

import (
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// Backend represents a complete emulator platform (rendering + input + audio)
type Backend interface {
	// Init configures the backend
	// This is a required step
	Init(config BackendConfig) error

	// Update handles rendering the frame and processing platform events
	// This is where each backend handles its specific input/audio systems
	Update(frame *video.FrameBuffer) error

	// Cleanup resources when shutting down
	Cleanup() error
}

// BackendConfig holds configuration for backends
type BackendConfig struct {
	Title        string
	Scale        int
	VSync        bool
	Fullscreen   bool
	ShowDebug    bool             // Backends may ignore unsupported features
	TestPattern  bool             // Display test pattern instead of emulation
	Callbacks    BackendCallbacks // Callbacks for backend communication
	InputManager *input.Manager   // Shared input manager for unified input handling
}

// BackendCallbacks allows backends to communicate with the emulator
type BackendCallbacks struct {
	// Input callbacks
	// TODO: move these to input manager in all backends

	OnKeyPress   func(key memory.JoypadKey)
	OnKeyRelease func(key memory.JoypadKey)

	// Control callbacks
	OnQuit func() // Backend requests shutdown (e.g., window close)

	// Debug callbacks (optional)
	OnDebugMessage func(message string) // Backend can send debug info to emulator
}
