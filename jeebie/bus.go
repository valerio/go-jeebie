package jeebie

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// BusInterface defines the interface for component communication
type BusInterface interface {
	Read(address uint16) byte
	Write(address uint16, value byte)
	RequestInterrupt(interrupt addr.Interrupt)
}

// Bus provides centralized component communication
type Bus struct {
	CPU *cpu.CPU
	MMU *memory.MMU
	GPU *video.GPU
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) Read(address uint16) byte {
	return b.MMU.Read(address)
}

func (b *Bus) Write(address uint16, value byte) {
	b.MMU.Write(address, value)
}

// TickInstruction executes one CPU instruction and ticks all components
// Returns the number of cycles consumed
func (b *Bus) TickInstruction() int {
	cycles := b.CPU.Tick()
	b.MMU.Tick(cycles)
	b.GPU.Tick(cycles)
	b.MMU.APU.Tick(cycles)
	return cycles
}

func (b *Bus) RequestInterrupt(interrupt addr.Interrupt) {
	b.MMU.RequestInterrupt(interrupt)
}

func (b *Bus) ReadBit(index uint8, address uint16) bool {
	return b.MMU.ReadBit(index, address)
}
