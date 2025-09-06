package memory

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/audio"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/serial"
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

// JoypadKey represents a key on the Gameboy joypad
type JoypadKey uint8

const (
	JoypadRight JoypadKey = iota
	JoypadLeft
	JoypadUp
	JoypadDown
	JoypadA
	JoypadB
	JoypadSelect
	JoypadStart
)

// SerialPort is the minimal interface for a serial device connected to SB/SC.
// Implementations MUST only accept reads/writes to addr.SB and addr.SC.
type SerialPort interface {
	Write(address uint16, value byte)
	Read(address uint16) byte
	Tick(cycles int)
	Reset()
}

// MMU allows access to all memory mapped I/O and data/registers
type MMU struct {
	cart      *Cartridge
	mbc       MBC
	memory    []byte
	APU       *audio.APU
	regionMap [256]memRegion

	joypadButtons uint8 // Actual state of buttons A/B/Start/Select, mapped to low bits of P1
	joypadDpad    uint8 // Actual state of d-pad directions, mapped to low bits of P1

	serial SerialPort
	timer  Timer
}

// New creates a new memory unity with default data, i.e. nothing cartridge loaded.
// Equivalent to turning on a Gameboy without a cartridge in.
func New() *MMU {
	mmu := &MMU{
		memory:        make([]byte, 0x10000),
		cart:          NewCartridge(),
		APU:           audio.New(),
		joypadButtons: 0x0F,
		joypadDpad:    0x0F,
	}
	mmu.serial = serial.NewLogSink(func() { mmu.RequestInterrupt(addr.SerialInterrupt) })
	mmu.timer.TimerInterruptHandler = func() { mmu.RequestInterrupt(addr.TimerInterrupt) }
	initRegionMap(mmu)
	return mmu
}

// Tick advances any i/o that needs it, if any.
func (m *MMU) Tick(cycles int) {
	m.timer.Tick(cycles)
	if m.serial != nil {
		m.serial.Tick(cycles)
	}
}

// SetTimerSeed initializes the internal timer divider seed and DIV register.
func (m *MMU) SetTimerSeed(seed uint16) {
	m.timer.SetSeed(seed)
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
		if address == addr.SB || address == addr.SC {
			return m.serial.Read(address)
		}
		if address == addr.DIV || address == addr.TIMA || address == addr.TMA || address == addr.TAC {
			return m.timer.Read(address)
		}
		if address >= 0xFF10 && address <= 0xFF3F {
			return m.APU.ReadRegister(address)
		}
		// Just in case, we always read the upper 3 bits of IF as 1.
		// They're not used, but have caused me some headaches when checking for
		// when the halt bug triggers (IF != 0).
		if address == addr.IF {
			return m.memory[address] | 0xE0
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
		if address == addr.P1 {
			m.writeJoypad(value)
			return
		}
		if address == addr.SB || address == addr.SC {
			m.serial.Write(address, value)
			return
		}
		if address == addr.DIV || address == addr.TIMA || address == addr.TMA || address == addr.TAC {
			m.timer.Write(address, value)
			return
		}
		if address >= 0xFF10 && address <= 0xFF3F {
			m.APU.WriteRegister(address, value)
			return
		}
		if address == addr.IF {
			// This goddamn register has its upper 3 bits always set as 1...
			// Beware if you're trying to match halt bug behavior.
			m.memory[address] = value | 0xE0
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

// updateJoypadRegister sets the joypad register (P1) according to selection bits
// and hardware (buttons) status.
//
// In real hw, this register is actually just a selector (bits 5-6) that control
// to which set of buttons the low bits (0-3) are mapped to.
//
// The mapping:
//   - if bit 4 is set, bits 0-3 are mapped to the 4 d-pad directions
//   - if bit 5 is set, bits 0-3 are mapped to A, B, Start, Select
//   - if both are set, hw does an AND of both button sets
//   - if neither are set, return 0x0F (high impedence state)
//
// This function is called whenever:
//   - there is a write to the P1 register (only set bits 4-5)
//   - a button is pressed or released (tracked separately)
//
// Note that 1 -> button released, 0 -> button pressed.
// Bits 6-7 are unused, they always read as 1 on real hardware.
func (m *MMU) updateJoypadRegister() {
	p1 := m.memory[addr.P1]
	result := uint8(0b11000000) // Bits 6-7 are always read as 1
	result |= p1 & 0b00110000   // Keep selection bits 4-5

	// A button group is selected if the corresponding bit is 0
	selectDpad := !bit.IsSet(4, p1)
	selectButtons := !bit.IsSet(5, p1)

	switch {
	case selectButtons && !selectDpad:
		result |= m.joypadButtons & 0x0F
	case selectDpad && !selectButtons:
		result |= m.joypadDpad & 0x0F
	case selectButtons && selectDpad:
		result |= m.joypadButtons & m.joypadDpad & 0x0F
	default:
		// no selection
		result |= 0x0F
	}

	m.memory[addr.P1] = result
}

func (m *MMU) writeJoypad(value uint8) {
	// Only bits 4-5 are writable (selection bits)
	m.memory[addr.P1] = value & 0b00110000
	m.updateJoypadRegister()
}

func (m *MMU) HandleKeyPress(key JoypadKey) {
	oldButtons := m.joypadButtons
	oldDpad := m.joypadDpad

	switch key {
	case JoypadRight:
		m.joypadDpad = bit.Reset(0, m.joypadDpad)
	case JoypadLeft:
		m.joypadDpad = bit.Reset(1, m.joypadDpad)
	case JoypadUp:
		m.joypadDpad = bit.Reset(2, m.joypadDpad)
	case JoypadDown:
		m.joypadDpad = bit.Reset(3, m.joypadDpad)
	case JoypadA:
		m.joypadButtons = bit.Reset(0, m.joypadButtons)
	case JoypadB:
		m.joypadButtons = bit.Reset(1, m.joypadButtons)
	case JoypadSelect:
		m.joypadButtons = bit.Reset(2, m.joypadButtons)
	case JoypadStart:
		m.joypadButtons = bit.Reset(3, m.joypadButtons)
	}

	buttonTransitions := oldButtons & ^m.joypadButtons
	dpadTransitions := oldDpad & ^m.joypadDpad
	if buttonTransitions|dpadTransitions != 0 {
		m.RequestInterrupt(addr.JoypadInterrupt)
	}

	m.updateJoypadRegister()
}

func (m *MMU) HandleKeyRelease(key JoypadKey) {
	switch key {
	case JoypadRight:
		m.joypadDpad = bit.Set(0, m.joypadDpad)
	case JoypadLeft:
		m.joypadDpad = bit.Set(1, m.joypadDpad)
	case JoypadUp:
		m.joypadDpad = bit.Set(2, m.joypadDpad)
	case JoypadDown:
		m.joypadDpad = bit.Set(3, m.joypadDpad)
	case JoypadA:
		m.joypadButtons = bit.Set(0, m.joypadButtons)
	case JoypadB:
		m.joypadButtons = bit.Set(1, m.joypadButtons)
	case JoypadSelect:
		m.joypadButtons = bit.Set(2, m.joypadButtons)
	case JoypadStart:
		m.joypadButtons = bit.Set(3, m.joypadButtons)
	}

	m.updateJoypadRegister()
}
