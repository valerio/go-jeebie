package backend

import (
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// Backend represents a complete emulator platform (rendering + input + audio)
// Backends are responsible for:
// - Rendering frames to their specific output (terminal, SDL window, etc.)
// - Translating platform-specific input events to Actions via InputManager
// - Handling backend-specific features (snapshots, test patterns, debug windows)
type Backend interface {
	// Init configures the backend with the provided configuration.
	// The InputManager in config is used to translate platform events to actions.
	// This is a required step before calling Update.
	Init(config BackendConfig) error

	// Update handles rendering the frame and processing platform events.
	// Backends should:
	// 1. Poll for platform-specific events (keyboard, window events, etc.)
	// 2. Translate events to Actions and call InputManager.Trigger()
	// 3. Render the provided frame (or test pattern if configured)
	// 4. Handle backend-specific features (debug windows, snapshots, etc.)
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
	// Control callbacks
	OnQuit func() // Backend requests shutdown (e.g., window close)

	// Debug callbacks (optional)
	OnDebugMessage func(message string) // Backend can send debug info to emulator
}
