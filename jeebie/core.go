package jeebie

import (
	"io/ioutil"
	"log"

	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

// Emulator represents the root struct and entry point for running the emulation
type Emulator struct {
	cpu    *cpu.CPU
	gpu    *video.GPU
	mem    *memory.MMU
	screen *video.Screen
}

func (e *Emulator) init() {
	e.mem = memory.NewWithCartridge(memory.NewCartridge())
	e.screen = video.NewScreen()

	e.cpu = cpu.New(e.mem)
	e.gpu = video.NewGpu(e.screen, e.mem)
}

// New creates a new emulator instance
func New() *Emulator {
	e := &Emulator{}
	e.init()

	return e
}

// NewWithFile creates a new emulator instance and loads the file specified into it.
func NewWithFile(path string) (*Emulator, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded %v bytes of ROM data\n", len(data))

	e := &Emulator{}
	e.init()
	e.mem = memory.NewWithCartridge(memory.NewCartridgeWithData(data))

	return e, nil
}

func (e *Emulator) Tick() {
	cycles := e.cpu.Tick()
	e.gpu.Tick(cycles)
}

// Run executes the main loop of the emulator
func (e *Emulator) Run() {
	defer e.screen.Destroy()
	defer sdl.Quit()

	for {
		e.Tick()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

			switch t := event.(type) {
			case *sdl.KeyDownEvent:
				if t.Keysym.Sym == sdl.K_ESCAPE {
					return
				}
			case *sdl.KeyUpEvent:

			}
		}
	}
}
