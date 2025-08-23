package backend

import (
	"github.com/valerio/go-jeebie/jeebie/audio"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// InputEvent represents an input event from a backend
type InputEvent struct {
	Action action.Action
	Type   event.Type
}

// Backend represents a complete emulator platform (rendering + input + audio)
// Backends are responsible for:
// - Rendering frames to their specific output (terminal, SDL window, etc.)
// - Capturing platform-specific input events and returning them as InputEvents
// - Handling backend-specific features (snapshots, test patterns, debug windows)
type Backend interface {
	// Init configures the backend with the provided configuration.
	// This is a required step before calling Update.
	Init(config BackendConfig) error

	// Update handles rendering the frame and collecting platform events.
	// Backends should:
	// 1. Poll for platform-specific events (keyboard, window events, etc.)
	// 2. Translate events to InputEvents and return them
	// 3. Render the provided frame (or test pattern if configured)
	// 4. Handle backend-specific features (debug windows, snapshots, etc.)
	// Returns a slice of InputEvents that occurred during this update
	Update(frame *video.FrameBuffer) ([]InputEvent, error)

	// Cleanup resources when shutting down
	Cleanup() error
}

// DebugDataProvider is a minimal interface for backends that need debug information
// This avoids exposing the entire Emulator interface to backends
type DebugDataProvider interface {
	// ExtractDebugData returns complete debug data for visualization
	// Returns nil if no debug data is available
	ExtractDebugData() *debug.CompleteDebugData
}

// BackendConfig holds configuration for backends
type BackendConfig struct {
	Title         string
	Scale         int
	VSync         bool
	Fullscreen    bool
	ShowDebug     bool              // Backends may ignore unsupported features
	TestPattern   bool              // Display test pattern instead of emulation
	DebugProvider DebugDataProvider // Optional: For backends with debug features
	APU           *audio.APU        // Optional: For backends with audio support
}
