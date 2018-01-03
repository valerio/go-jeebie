package memory

import (
	"github.com/valep27/go-jeebie/jeebie"
	"fmt"
)

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
	cart   *jeebie.Cartridge
	memory []byte
}

func New() *MMU {
	return &MMU{
		memory: make([]byte, 0x10000),
	}
}

func NewWithCartridge(cart *jeebie.Cartridge) *MMU {
	mmu := New()
	mmu.cart = cart
	return mmu
}

func isBetween(addr, start, end uint16) bool {
	return addr >= start && addr <= end
}

func (m *MMU) ReadByte(addr uint16) byte {

	// ROM
	if isBetween(addr, 0, 0x7FFF) {
		return m.cart.ReadByte(addr)
	}

	// VRAM
	if isBetween(addr, 0x8000, 0x9FFF) {
		return m.memory[addr]
	}

	// external RAM
	if isBetween(addr, 0xA000, 0xBFFF) {
		panic(fmt.Sprintf("Attempted read at unused/unmapped address: %v", addr))
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
		panic(fmt.Sprintf("Attempted read at unused/unmapped address: %v", addr))
	}

	// IO registers
	if isBetween(addr, 0xFF00, 0xFF7F) {
		return m.memory[addr]
	}

	// Zero Page RAM
	if isBetween(addr, 0xFF80, 0xFFFE) {
		return m.memory[addr]
	}

	/* Interrupt Enable register */
	if addr == 0xFFFF {
		return m.memory[addr]
	}

	panic(fmt.Sprintf("Attempted read at unused/unmapped address: %v", addr))
}

func (m *MMU) WriteByte(addr uint16, value byte) {

	// ROM
	if isBetween(addr, 0, 0x7FFF) {
		m.cart.WriteByte(addr, value)
		return
	}

	// VRAM
	if isBetween(addr, 0x8000, 0x9FFF) {
		m.memory[addr] = value
		return
	}

	// external RAM
	if isBetween(addr, 0xA000, 0xBFFF) {
		panic(fmt.Sprintf("Attempted write at unused/unmapped address: %v", addr))
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
		panic(fmt.Sprintf("Attempted write at unused/unmapped address: %v", addr))
	}

	// IO registers
	if isBetween(addr, 0xFF00, 0xFF7F) {
		m.memory[addr] = value
		return
	}

	// Zero Page RAM
	if isBetween(addr, 0xFF80, 0xFFFE) {
		m.memory[addr] = value
		return
	}

	/* Interrupt Enable register */
	if addr == 0xFFFF {
		m.memory[addr] = value
		return
	}

	panic(fmt.Sprintf("Attempted write at unused/unmapped address: %v", addr))
}
