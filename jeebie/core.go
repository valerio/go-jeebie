package jeebie

import (
	"github.com/valep27/go-jeebie/jeebie/cpu"
	"github.com/valep27/go-jeebie/jeebie/memory"
	"github.com/valep27/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

// Emulator represents the root struct and entry point for running the emulation
type Emulator struct {
	cpu    *cpu.CPU
	mem    *memory.MMU
	screen *video.Screen
}

func (e *Emulator) init() {
	e.screen = video.NewScreen()
}

// New creates a new emulator instance
func New() *Emulator {
	e := &Emulator{}
	e.init()

	return e
}

// Run executes the main loop of the emulator
func (e *Emulator) Run() {
	for {
		e.screen.Draw()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

			switch t := event.(type) {
			case *sdl.KeyDownEvent:
				if t.Keysym.Sym == sdl.K_ESCAPE {
					defer e.screen.Destroy()
					defer sdl.Quit()
					return
				}
			case *sdl.KeyUpEvent:

			}
		}
	}
}
