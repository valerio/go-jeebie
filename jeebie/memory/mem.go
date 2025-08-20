package memory

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
	cart   *Cartridge
	mbc    MBC
	memory []byte
	joypad *Joypad
}

// New creates a new memory unity with default data, i.e. nothing cartridge loaded.
// Equivalent to turning on a Gameboy without a cartridge in.
func New() *MMU {
	return &MMU{
		memory: make([]byte, 0x10000),
		cart:   NewCartridge(),
		joypad: NewJoypad(),
	}
}

// NewWithCartridge creates a new memory unit with the provided cartridge data loaded.
// Equivalent to turning on a Gameboy with a cartridge in.
func NewWithCartridge(cart *Cartridge) *MMU {
	mmu := New()
	mmu.cart = cart

	switch cart.mbcType {
	case NoMBCType:
		mmu.mbc = NewNoMBC(cart.data)
	case MBC1Type:
		mmu.mbc = NewMBC1(cart.data, cart.hasBattery, cart.ramBankCount)
	case MBC1MultiType:
		mmu.mbc = NewMBC1(cart.data, cart.hasBattery, cart.ramBankCount) // FIXME: add support for multicart
	case MBC2Type:
		mmu.mbc = NewMBC2(cart.data)
	case MBC3Type:
		mmu.mbc = NewMBC3(cart.data, cart.ramBankCount, cart.hasRTC, nil)
	case MBC5Type:
		mmu.mbc = NewMBC5(cart.data, cart.hasRumble, cart.ramBankCount)
	case MBCUnknownType:
		panic("unsupported MBC type: unknown")
	default:
		panic(fmt.Sprintf("unsupported MBC type: %d", cart.mbcType))
	}

	return mmu
}

func isBetween(addr, start, end uint16) bool {
	return addr >= start && addr <= end
}

// RequestInterrupt sets the interrupt flag (IF register) of the chosen interrupt to 1.
func (m *MMU) RequestInterrupt(interrupt addr.Interrupt) {
	interruptFlags := m.Read(addr.IF)

	var bitPos uint8
	switch interrupt {
	case addr.VBlankInterrupt:
		bitPos = 0
	case addr.LCDSTATInterrupt:
		bitPos = 1
	case addr.TimerInterrupt:
		bitPos = 2
	case addr.SerialInterrupt:
		bitPos = 3
	case addr.JoypadInterrupt:
		bitPos = 4
	default:
		panic(fmt.Sprintf("Unknown interrupt: 0x%02X", uint8(interrupt)))
	}

	newFlags := bit.Set(bitPos, interruptFlags)

	m.Write(addr.IF, newFlags)
}

func (m *MMU) ReadBit(index uint8, address uint16) bool {
	return bit.IsSet(index, m.Read(address))
}

func (m *MMU) SetBit(index uint8, address uint16, set bool) {
	value := m.Read(address)
	if set {
		value = bit.Set(index, value)
	} else {
		value = bit.Reset(index, value)
	}
	m.Write(address, value)
}

func (m *MMU) Read(address uint16) byte {
	// ROM / RAM
	if isBetween(address, 0, 0x7FFF) || isBetween(address, 0xA000, 0xBFFF) {
		if m.mbc == nil {
			slog.Warn("Reading from ROM/external RAM with no cartridge", "addr", fmt.Sprintf("0x%04X", address))
			return 0xFF // simulate no cartridge behavior
		}
		return m.mbc.Read(address)
	}

	// VRAM
	if isBetween(address, 0x8000, 0x9FFF) {
		return m.memory[address]
	}

	// RAM
	if isBetween(address, 0xC000, 0xDFFF) {
		return m.memory[address]
	}

	// RAM mirror
	if isBetween(address, 0xE000, 0xFDFF) {
		mirroredAddr := address - 0x2000
		return m.memory[mirroredAddr]
	}

	// OAM
	if isBetween(address, 0xFE00, 0xFE9F) {
		return m.memory[address]
	}

	// Unused
	if isBetween(address, 0xFEA0, 0xFEFF) {
		return m.memory[address]
	}

	// IO registers
	if isBetween(address, 0xFF00, 0xFF7F) {
		if address == 0xFF00 {
			return m.joypad.Read()
		}
		return m.memory[address]
	}

	// Zero Page RAM & I/O registers
	if isBetween(address, 0xFF80, 0xFFFF) {
		return m.memory[address]
	}

	panic(fmt.Sprintf("Attempted read at unused/unmapped address: 0x%X", address))
}

func (m *MMU) Write(address uint16, value byte) {

	// ROM
	if isBetween(address, 0, 0x7FFF) {
		if m.mbc == nil {
			slog.Warn("Writing to ROM with no cartridge", "addr", fmt.Sprintf("0x%04X", address), "value", fmt.Sprintf("0x%02X", value))
			return
		}
		m.mbc.Write(address, value)
		return
	}

	// VRAM
	if isBetween(address, 0x8000, 0x9FFF) {
		m.memory[address] = value
		return
	}

	// external RAM
	if isBetween(address, 0xA000, 0xBFFF) {
		if m.mbc == nil {
			slog.Warn("Writing to external RAM with no cartridge", "addr", fmt.Sprintf("0x%04X", address), "value", fmt.Sprintf("0x%02X", value))
			return
		}
		m.mbc.Write(address, value)
		return
	}

	// RAM
	if isBetween(address, 0xC000, 0xDFFF) {
		m.memory[address] = value
		return
	}

	// RAM mirror
	if isBetween(address, 0xE000, 0xFDFF) {
		mirroredAddr := address - 0x2000
		m.memory[mirroredAddr] = value
		return
	}

	// OAM
	if isBetween(address, 0xFE00, 0xFE9F) {
		m.memory[address] = value
		return
	}

	// Unused
	if isBetween(address, 0xFEA0, 0xFEFF) {
		m.memory[address] = value
		return
	}

	// IO registers
	if isBetween(address, 0xFF00, 0xFF7F) {
		if address == 0xFF00 {
			m.joypad.Write(value)
			return
		}
		// handle DMA transfer
		if address == addr.DMA {
			sourceAddr := uint16(value) << 8
			slog.Debug("DMA transfer", "source", fmt.Sprintf("0x%04X", sourceAddr), "value", fmt.Sprintf("0x%02X", value))
			// DMA transfer copies 160 bytes from source to OAM
			for i := range uint16(160) {
				m.memory[0xFE00+i] = m.Read(sourceAddr + i)
			}
			// still store the value in the DMA register
			m.memory[address] = value
			// TODO: update timers according to DMA transfer timing, cpu should detect it most likely.
			return
		}
		m.memory[address] = value
		return
	}

	// Zero Page RAM + I/O registers
	if isBetween(address, 0xFF80, 0xFFFF) {
		m.memory[address] = value
		return
	}

	panic(fmt.Sprintf("Attempted write at unused/unmapped address: 0x%X", address))
}

func (m *MMU) HandleKeyPress(key JoypadKey) {
	m.joypad.Press(key)
}

func (m *MMU) HandleKeyRelease(key JoypadKey) {
	m.joypad.Release(key)
}

// GetJoypadState returns the raw joypad state for debugging
func (m *MMU) GetJoypadState() (uint8, uint8) {
	return m.joypad.buttons, m.joypad.dpad
}

// GetJoypad returns the joypad instance for direct access
func (m *MMU) GetJoypad() *Joypad {
	return m.joypad
}
