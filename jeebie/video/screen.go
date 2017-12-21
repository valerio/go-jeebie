package video

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	renderScale = 3
	width       = 160
	height      = 144
)

// Screen encapsulates video output for the emulator
type Screen struct {
	window   *sdl.Window
	renderer *sdl.Renderer
}

// NewScreen initializes and returns a screen
func NewScreen() *Screen {
	var err error
	s := &Screen{}

	s.window, err = sdl.CreateWindow("go-jeebie",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		width*renderScale,
		height*renderScale,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)

	if err != nil {
		panic(err)
	}

	s.renderer, err = sdl.CreateRenderer(s.window, -1, sdl.RENDERER_ACCELERATED)

	if err != nil {
		panic(err)
	}

	return s
}

// Draw presents a new frame to the screen
func (s *Screen) Draw(buffer []uint32) {
	var err error

	surface, err := sdl.CreateRGBSurfaceFrom(
		unsafe.Pointer(&buffer[0]),
		width,
		height,
		32,
		4*width,
		0x000000FF,
		0x0000FF00,
		0x00FF0000,
		0xFF000000)

	if err != nil {
		panic(err)
	}

	defer surface.Free()

	surface.Lock()
	s.renderer.Clear()
	tex, err := s.renderer.CreateTextureFromSurface(surface)
	surface.Unlock()

	if err != nil {
		panic(err)
	}

	defer tex.Destroy()

	s.renderer.Copy(tex, nil, nil)
	s.renderer.Present()
}

// Destroy cleans up resources used by the screen
func (s *Screen) Destroy() {
	s.window.Destroy()
	s.renderer.Destroy()
}
