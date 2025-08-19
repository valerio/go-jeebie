//go:build !sdl2

package sdl2

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// Backend stub for when SDL2 is not available
type Backend struct{}

// New creates a stub SDL2 backend that returns an error
func New() *Backend {
	return &Backend{}
}

// Init returns an error indicating SDL2 is not available
func (s *Backend) Init(config backend.BackendConfig) error {
	return fmt.Errorf("SDL2 backend not available - build with -tags sdl2 to enable")
}

// Update returns an error
func (s *Backend) Update(frame *video.FrameBuffer) error {
	return fmt.Errorf("SDL2 backend not available")
}

// Cleanup does nothing
func (s *Backend) Cleanup() error {
	return nil
}

// UpdateDebugData does nothing
func (s *Backend) UpdateDebugData(oam interface{}, vram interface{}) {
	// No-op
}

// ToggleDebugWindow does nothing
func (s *Backend) ToggleDebugWindow() {
	// No-op
}
