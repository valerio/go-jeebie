//go:build !sdl2

package backend

import (
	"fmt"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// SDL2Backend stub for when SDL2 is not available
type SDL2Backend struct{}

func NewSDL2Backend() *SDL2Backend {
	return &SDL2Backend{}
}

func (s *SDL2Backend) Init(config BackendConfig) error {
	return fmt.Errorf("SDL2 backend not available - compile with -tags sdl2 and install SDL2 development libraries")
}

func (s *SDL2Backend) Update(frame *video.FrameBuffer) error {
	return fmt.Errorf("SDL2 backend not available")
}

func (s *SDL2Backend) Cleanup() error {
	return nil
}
