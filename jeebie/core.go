package jeebie

import (
	"io/ioutil"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// Emulator represents the root struct and entry point for running the emulation
type Emulator struct {
	cpu *cpu.CPU
	gpu *video.GPU
	mem *memory.MMU
}

func (e *Emulator) init(mem *memory.MMU) {
	e.cpu = cpu.New(mem)
	e.gpu = video.NewGpu(mem)
	e.mem = mem
}

// New creates a new emulator instance
func New() *Emulator {
	e := &Emulator{}
	e.init(memory.NewWithCartridge(memory.NewCartridge()))

	return e
}

// NewWithFile creates a new emulator instance and loads the file specified into it.
func NewWithFile(path string) (*Emulator, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	slog.Debug("Loaded ROM data", "size", len(data))

	e := &Emulator{}
	e.init(memory.NewWithCartridge(memory.NewCartridgeWithData(data)))

	return e, nil
}

func (e *Emulator) RunUntilFrame() {
	total := 0
	for {
		cycles := e.cpu.Tick()
		e.gpu.Tick(cycles)

		total += cycles

		if total >= 70224 {
			return
		}
	}
}

func (e *Emulator) GetCurrentFrame() *video.FrameBuffer {
	return e.gpu.GetFrameBuffer()
}
