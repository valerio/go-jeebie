package memory

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
	cart   *Cartridge
	memory []byte
}

// New creates a new memory unity with default data, i.e. nothing cartridge loaded.
// Equivalent to turning on a Gameboy without a cartridge in.
func New() *MMU {
	return &MMU{
		memory: make([]byte, 0x10000),
	}
}

// NewWithCartridge creates a new memory unit with the provided cartridge data loaded.
// Equivalent to turning on a Gameboy with a cartridge in.
func NewWithCartridge(cart *Cartridge) *MMU {
	mmu := New()
	mmu.cart = cart
	return mmu
}

func isBetween(addr, start, end uint16) bool {
	return addr >= start && addr <= end
}

// RequestInterrupt sets the interrupt flag (IF register) of the chosen interrupt to 1.
func (m *MMU) RequestInterrupt(interrupt addr.Interrupt) {
	interruptFlags := m.Read(addr.IF)
	m.Write(addr.IF, bit.Set(uint8(interrupt), interruptFlags))
}

func (m *MMU) ReadBit(index uint8, addr uint16) bool {
	return bit.IsSet(index, m.Read(addr))
}

func (m *MMU) SetBit(index uint8, addr uint16, set bool) {
	value := m.Read(addr)
	if set {
		bit.Set(index, value)
	} else {
		bit.Reset(index, value)
	}
	m.Write(addr, value)
}

func (m *MMU) Read(addr uint16) byte {
	// ROM
	if isBetween(addr, 0, 0x7FFF) {
		return m.cart.Read(addr)
	}

	// VRAM
	if isBetween(addr, 0x8000, 0x9FFF) {
		return m.memory[addr]
	}

	// external RAM
	if isBetween(addr, 0xA000, 0xBFFF) {
		return m.memory[addr]
	}

	// RAM
	if isBetween(addr, 0xC000, 0xDFFF) {
		return m.memory[addr]
	}

	// RAM mirror
	if isBetween(addr, 0xE000, 0xFDFF) {
		mirroredAddr := addr - 0x2000
		return m.memory[mirroredAddr]
	}

	// OAM
	if isBetween(addr, 0xFE00, 0xFE9F) {
		return m.memory[addr]
	}

	// Unused
	if isBetween(addr, 0xFEA0, 0xFEFF) {
		return m.memory[addr]
	}

	// IO registers
	if isBetween(addr, 0xFF00, 0xFF7F) {
		return m.memory[addr]
	}

	// Zero Page RAM & I/O registers
	if isBetween(addr, 0xFF80, 0xFFFF) {
		return m.memory[addr]
	}

	panic(fmt.Sprintf("Attempted read at unused/unmapped address: 0x%X", addr))
}

func (m *MMU) Write(addr uint16, value byte) {

	// ROM
	if isBetween(addr, 0, 0x7FFF) {
		m.cart.Write(addr, value)
		return
	}

	// VRAM
	if isBetween(addr, 0x8000, 0x9FFF) {
		m.memory[addr] = value
		return
	}

	// external RAM
	if isBetween(addr, 0xA000, 0xBFFF) {
		m.memory[addr] = value
		return
	}

	// RAM
	if isBetween(addr, 0xC000, 0xDFFF) {
		m.memory[addr] = value
		return
	}

	// RAM mirror
	if isBetween(addr, 0xE000, 0xFDFF) {
		mirroredAddr := addr - 0x2000
		m.memory[mirroredAddr] = value
		return
	}

	// OAM
	if isBetween(addr, 0xFE00, 0xFE9F) {
		m.memory[addr] = value
		return
	}

	// Unused
	if isBetween(addr, 0xFEA0, 0xFEFF) {
		m.memory[addr] = value
		return
	}

	// IO registers
	if isBetween(addr, 0xFF00, 0xFF7F) {
		m.memory[addr] = value
		return
	}

	// Zero Page RAM + I/O registers
	if isBetween(addr, 0xFF80, 0xFFFF) {
		m.memory[addr] = value
		return
	}

	panic(fmt.Sprintf("Attempted write at unused/unmapped address: 0x%X", addr))
}
