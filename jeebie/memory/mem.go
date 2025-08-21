package memory

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

type memRegion uint8

const (
	regionROM memRegion = iota
	regionVRAM
	regionExtRAM
	regionWRAM
	regionEcho
	regionOAM
	regionUnused
	regionIO
	regionHRAM
)

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
	cart      *Cartridge
	mbc       MBC
	memory    []byte
	joypad    *Joypad
	regionMap [256]memRegion
}

// New creates a new memory unity with default data, i.e. nothing cartridge loaded.
// Equivalent to turning on a Gameboy without a cartridge in.
func New() *MMU {
	mmu := &MMU{
		memory: make([]byte, 0x10000),
		cart:   NewCartridge(),
		joypad: NewJoypad(),
	}
	initRegionMap(mmu)
	return mmu
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

func initRegionMap(m *MMU) {
	// ROM: 0x0000-0x7FFF
	for i := 0x00; i <= 0x7F; i++ {
		m.regionMap[i] = regionROM
	}
	// VRAM: 0x8000-0x9FFF
	for i := 0x80; i <= 0x9F; i++ {
		m.regionMap[i] = regionVRAM
	}
	// External RAM: 0xA000-0xBFFF
	for i := 0xA0; i <= 0xBF; i++ {
		m.regionMap[i] = regionExtRAM
	}
	// Work RAM: 0xC000-0xDFFF
	for i := 0xC0; i <= 0xDF; i++ {
		m.regionMap[i] = regionWRAM
	}
	// Echo RAM: 0xE000-0xFDFF
	for i := 0xE0; i <= 0xFD; i++ {
		m.regionMap[i] = regionEcho
	}
	// OAM: 0xFE00-0xFE9F, Unused: 0xFEA0-0xFEFF
	m.regionMap[0xFE] = regionOAM
	// IO + HRAM: 0xFF00-0xFFFF
	m.regionMap[0xFF] = regionIO
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
	switch m.regionMap[address>>8] {
	case regionROM, regionExtRAM:
		if m.mbc == nil {
			slog.Warn("Reading from ROM/external RAM with no cartridge", "addr", fmt.Sprintf("0x%04X", address))
			return 0xFF
		}
		return m.mbc.Read(address)
	case regionVRAM, regionWRAM:
		return m.memory[address]
	case regionEcho:
		if address <= 0xFDFF {
			return m.memory[address-0x2000]
		}
		return m.memory[address-0x2000]
	case regionOAM:
		if address <= 0xFE9F {
			return m.memory[address]
		}
		// Unused area 0xFEA0-0xFEFF
		return m.memory[address]
	case regionIO:
		if address == 0xFF00 {
			return m.joypad.Read()
		}
		if address >= 0xFF80 {
			// HRAM
			return m.memory[address]
		}
		// Other IO registers
		return m.memory[address]
	default:
		panic(fmt.Sprintf("Attempted read at unmapped address: 0x%X", address))
	}
}

func (m *MMU) Write(address uint16, value byte) {
	switch m.regionMap[address>>8] {
	case regionROM:
		if m.mbc == nil {
			slog.Warn("Writing to ROM with no cartridge", "addr", fmt.Sprintf("0x%04X", address), "value", fmt.Sprintf("0x%02X", value))
			return
		}
		m.mbc.Write(address, value)
	case regionVRAM:
		m.memory[address] = value
	case regionExtRAM:
		if m.mbc == nil {
			slog.Warn("Writing to external RAM with no cartridge", "addr", fmt.Sprintf("0x%04X", address), "value", fmt.Sprintf("0x%02X", value))
			return
		}
		m.mbc.Write(address, value)
	case regionWRAM:
		m.memory[address] = value
	case regionEcho:
		if address <= 0xFDFF {
			m.memory[address-0x2000] = value
		}
	case regionOAM:
		if address <= 0xFE9F {
			m.memory[address] = value
		} else {
			// Unused area 0xFEA0-0xFEFF
			m.memory[address] = value
		}
	case regionIO:
		if address == 0xFF00 {
			m.joypad.Write(value)
			return
		}
		if address == addr.DMA {
			sourceAddr := uint16(value) << 8
			// DMA transfer copies 160 bytes from source to OAM
			for i := range uint16(160) {
				m.memory[0xFE00+i] = m.Read(sourceAddr + i)
			}
			m.memory[address] = value
			return
		}
		if address >= 0xFF80 {
			// HRAM
			m.memory[address] = value
			return
		}
		// Other IO registers
		m.memory[address] = value
	default:
		panic(fmt.Sprintf("Attempted write at unmapped address: 0x%X", address))
	}
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
