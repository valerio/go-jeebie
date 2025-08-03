package memory

import "time"

// MBC represents a Memory Bank Controller interface that all MBC types must implement
type MBC interface {
	// Read reads a byte from the specified address
	Read(addr uint16) uint8
	// Write writes a byte to the specified address, returns the written value
	Write(addr uint16, value uint8) uint8
}

// NoMBC represents cartridges with no memory banking capabilities.
// These are typically smaller games (32KB or less) that fit entirely in the
// base memory region. The cartridge ROM is directly mapped to 0x0000-0x7FFF
// and cannot be banked/switched. These cartridges cannot have external RAM.
type NoMBC struct {
	rom []uint8 // ROM data
}

// NewNoMBC creates a new NoMBC controller
func NewNoMBC(romData []uint8) *NoMBC {
	return &NoMBC{
		rom: romData,
	}
}

func (m *NoMBC) Read(addr uint16) uint8 {
	// For NoMBC, we just read directly from ROM
	return m.rom[addr]
}

func (m *NoMBC) Write(addr uint16, value uint8) uint8 {
	// NoMBC doesn't support writing to ROM
	return 0
}

// MBC1 is the first and most common MBC chip. Features include:
// - Supports up to 2MB ROM (125 16KB banks)
// - Up to 32KB RAM (4 8KB banks)
// - Bank 0 always mapped to 0x0000-0x3FFF
// - Switchable ROM bank at 0x4000-0x7FFF
// - Optional RAM banking at 0xA000-0xBFFF
// - Two banking modes:
//   - Mode 0 (ROM): Allows access to full ROM but only 8KB RAM
//   - Mode 1 (RAM): Restricts ROM banking but allows full RAM access
// - Optional battery backup for RAM persistence
type MBC1 struct {
	rom          []uint8
	ram          []uint8
	romBank      uint8
	ramBank      uint8
	ramEnabled   bool
	bankingMode  uint8
	hasBattery   bool
	ramBankCount uint8
}

// NewMBC1 creates a new MBC1 controller
func NewMBC1(romData []uint8, hasBattery bool, ramBankCount uint8) *MBC1 {
	ramSize := uint32(ramBankCount) * 0x2000 // 8KB per RAM bank
	return &MBC1{
		rom:          romData,
		ram:          make([]uint8, ramSize),
		romBank:      1,
		ramBank:      0,
		ramEnabled:   false,
		bankingMode:  0,
		hasBattery:   hasBattery,
		ramBankCount: ramBankCount,
	}
}

func (m *MBC1) Read(addr uint16) uint8 {
	switch {
	case addr <= 0x3FFF:
		// ROM Bank 0
		return m.rom[addr]
	case addr >= 0x4000 && addr <= 0x7FFF:
		// Switchable ROM Bank
		offset := uint32(m.romBank) * 0x4000
		if offset >= uint32(len(m.rom)) {
			// If bank would be out of bounds, wrap around
			offset = offset % uint32(len(m.rom))
		}
		return m.rom[offset+uint32(addr-0x4000)]
	case addr >= 0xA000 && addr <= 0xBFFF:
		// RAM Bank
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(m.ramBank) * 0x2000
		if offset >= uint32(len(m.ram)) {
			// If bank would be out of bounds, wrap around
			offset = offset % uint32(len(m.ram))
		}
		return m.ram[offset+uint32(addr-0xA000)]
	default:
		return 0xFF
	}
}

func (m *MBC1) Write(addr uint16, value uint8) uint8 {
	switch {
	case addr <= 0x1FFF:
		// RAM Enable
		m.ramEnabled = (value & 0x0F) == 0x0A
	case addr >= 0x2000 && addr <= 0x3FFF:
		// ROM Bank Number (lower 5 bits)
		bank := value & 0x1F
		if bank == 0 {
			bank = 1
		}
		m.romBank = (m.romBank & 0x60) | bank
	case addr >= 0x4000 && addr <= 0x5FFF:
		// RAM Bank Number or Upper ROM Bank Number
		if m.bankingMode == 0 {
			// ROM Banking mode - value goes to upper bits of ROM bank
			m.romBank = (m.romBank & 0x1F) | ((value & 0x03) << 5)
		} else {
			// RAM Banking mode - value goes to RAM bank
			m.ramBank = value & 0x03
		}
	case addr >= 0x6000 && addr <= 0x7FFF:
		// Banking Mode Select
		m.bankingMode = value & 0x01
		if m.bankingMode == 1 {
			// When switching to RAM banking mode, clear the upper bits of ROM bank
			m.romBank &= 0x1F
		}
	case addr >= 0xA000 && addr <= 0xBFFF:
		// RAM Bank
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(m.ramBank) * 0x2000
		if offset >= uint32(len(m.ram)) {
			offset = (offset % uint32(len(m.ram)))
		}
		m.ram[offset+uint32(addr-0xA000)] = value
	}
	return value
}

// MBC2 is a simpler MBC chip with built-in RAM. Features include:
// - Supports up to 256KB ROM (16 16KB banks)
// - Built-in 512x4 bits RAM (not external)
// - RAM does not require enabling (always accessible)
// - ROM banking similar to MBC1 but simpler
// - The least significant bit of the upper address byte selects between
//   ROM banking and RAM access
// - RAM is limited to 4-bit values (upper 4 bits are ignored)
// - Optional battery backup for the built-in RAM
type MBC2 struct {
	rom        []uint8
	ram        []uint8 // 512x4 bits RAM
	romBank    uint8
	ramEnabled bool
}

// NewMBC2 creates a new MBC2 controller
func NewMBC2(romData []uint8) *MBC2 {
	return &MBC2{
		rom:        romData,
		ram:        make([]uint8, 512),
		romBank:    1,
		ramEnabled: false,
	}
}

func (m *MBC2) Read(addr uint16) uint8 {
	switch {
	case addr <= 0x3FFF:
		// ROM Bank 0
		return m.rom[addr]
	case addr >= 0x4000 && addr <= 0x7FFF:
		// Switchable ROM Bank
		offset := uint32(m.romBank) * 0x4000
		if offset >= uint32(len(m.rom)) {
			offset = offset % uint32(len(m.rom))
		}
		return m.rom[offset+uint32(addr-0x4000)]
	case addr >= 0xA000 && addr <= 0xA1FF:
		// Built-in RAM
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(addr-0xA000) / 2
		return m.ram[offset]
	default:
		return 0xFF
	}
}

func (m *MBC2) Write(addr uint16, value uint8) uint8 {
	switch {
	case addr <= 0x1FFF:
		// RAM Enable
		if addr&0x0100 == 0 {
			// Only the lower 4 bits of the address are used
			m.ramEnabled = (value & 0x0F) == 0x0A
		}
	case addr >= 0x2000 && addr <= 0x3FFF:
		// ROM Bank Number
		if addr&0x0100 == 1 {
			// Only the lower 4 bits of the address are used
			m.romBank = value & 0x0F
			if m.romBank == 0 {
				m.romBank = 1
			}
		}
	case addr >= 0xA000 && addr <= 0xA1FF:
		// Built-in RAM
		// It's a 512x4 bits RAM, so only the lower 4 bits are used
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(addr-0xA000) / 2
		m.ram[offset] = value & 0x0F
	case addr >= 0x1A00 && addr <= 0x1FFF:
		// Commands $1A to $1F are stubs
		// These commands are used to control the RTC, but MBC2 does not have RTC support
	}
	return value
}

type Clock interface {
	Now() time.Time
}

type systemClockFunc func() time.Time

func (s systemClockFunc) Now() time.Time {
	return s()
}

// MBC3 is an advanced MBC chip with RTC support. Features include:
// - Supports up to 2MB ROM (128 16KB banks)
// - Up to 32KB RAM (4 8KB banks)
// - Real-Time Clock (RTC) functionality
// - RTC has 5 registers: Seconds, Minutes, Hours, Days (lower), Days (upper)/Flags
// - Similar banking to MBC1 but with different register layout
// - RAM and RTC can be battery backed
// - Used in games that needed to track real time (e.g. Pokémon Gold/Silver)
type MBC3 struct {
	rom        []uint8
	ram        []uint8
	rtc        [5]uint8 // RTC registers
	romBank    uint8
	ramBank    uint8
	ramEnabled bool
	hasRTC     bool
	rtcLatch   bool      // Flag to indicate if RTC data is latched
	clock      Clock     // Clock interface for RTC functionality
	rtcTime    time.Time // Time when RTC was last updated
}

// NewMBC3 creates a new MBC3 controller
func NewMBC3(romData []uint8, ramBankCount uint8, hasRTC bool, clock Clock) *MBC3 {
	if hasRTC && clock == nil {
		// default to system clock if no clock is provided
		clock = systemClockFunc(time.Now)
	}

	ramSize := uint32(ramBankCount) * 0x2000
	return &MBC3{
		rom:        romData,
		ram:        make([]uint8, ramSize),
		romBank:    1,
		ramEnabled: false,
		hasRTC:     hasRTC,
		rtcLatch:   false,
		clock:      clock,
		rtcTime:    clock.Now(),
	}
}

func (m *MBC3) Read(addr uint16) uint8 {
	switch {
	case addr <= 0x3FFF:
		return m.rom[addr]
	case addr >= 0x4000 && addr <= 0x7FFF:
		offset := uint32(m.romBank) * 0x4000
		if offset >= uint32(len(m.rom)) {
			offset = offset % uint32(len(m.rom))
		}
		return m.rom[offset+uint32(addr-0x4000)]
	case addr >= 0xA000 && addr <= 0xBFFF:
		if !m.ramEnabled {
			return 0xFF
		}
		if m.ramBank <= 0x03 {
			offset := uint32(m.ramBank) * 0x2000
			if offset >= uint32(len(m.ram)) {
				offset = offset % uint32(len(m.ram))
			}
			return m.ram[offset+uint32(addr-0xA000)]
		} else if m.hasRTC && m.ramBank >= 0x08 && m.ramBank <= 0x0C {
			if m.rtcLatch {
				m.updateRTC()
				m.rtcLatch = false
			}
			return m.rtc[m.ramBank-0x08]
		}
		return 0xFF
	default:
		return 0xFF
	}
}

func (m *MBC3) Write(addr uint16, value uint8) uint8 {
	switch {
	case addr <= 0x1FFF:
		m.ramEnabled = (value & 0x0F) == 0x0A
	case addr >= 0x2000 && addr <= 0x3FFF:
		bank := value & 0x7F
		if bank == 0 {
			bank = 1
		}
		m.romBank = bank
	case addr >= 0x4000 && addr <= 0x5FFF:
		m.ramBank = value
	case addr >= 0x6000 && addr <= 0x7FFF:
		if value == 0x00 {
			m.rtcLatch = true
		}
	case addr >= 0xA000 && addr <= 0xBFFF:
		if !m.ramEnabled {
			return 0xFF
		}
		if m.ramBank <= 0x03 {
			offset := uint32(m.ramBank) * 0x2000
			if offset >= uint32(len(m.ram)) {
				offset = offset % uint32(len(m.ram))
			}
			m.ram[offset+uint32(addr-0xA000)] = value
		} else if m.hasRTC && m.ramBank >= 0x08 && m.ramBank <= 0x0C {
			m.rtc[m.ramBank-0x08] = value
		}
	case addr >= 0x1A00 && addr <= 0x1FFF:
		// Commands $1A to $1F are stubs
	}
	return value
}

func (m *MBC3) updateRTC() {
	now := m.clock.Now()
	duration := now.Sub(m.rtcTime)
	m.rtcTime = now

	seconds := m.rtc[0] + uint8(duration.Seconds())
	minutes := m.rtc[1] + uint8(duration.Minutes())
	hours := m.rtc[2] + uint8(duration.Hours())

	m.rtc[0] = seconds % 60
	m.rtc[1] = minutes % 60
	m.rtc[2] = hours % 24
	// Days are split into two bytes
	// Handle days overflow
	daysLow := m.rtc[3] + uint8(duration.Hours()/24)
	daysHigh := m.rtc[4]

	daysHigh += daysLow / 255
	daysLow %= 255

	m.rtc[3] = daysLow
	m.rtc[4] = daysHigh
}

// MBC5 is the most advanced MBC chip. Features include:
// - Supports up to 8MB ROM (512 16KB banks)
// - Up to 128KB RAM (16 8KB banks)
// - Simple ROM/RAM banking with no quirks (unlike MBC1)
// - 9-bit ROM bank number (allows all 512 banks to be directly accessed)
// - Optional rumble motor support
// - Used in Game Boy Color games that needed more ROM/RAM
// - Backwards compatible with Game Boy
type MBC5 struct {
	rom        []uint8
	ram        []uint8
	romBank    uint16 // MBC5 supports up to 512 ROM banks
	ramBank    uint8
	ramEnabled bool
	hasRumble  bool
}

// NewMBC5 creates a new MBC5 controller
func NewMBC5(romData []uint8, hasRumble bool, ramBankCount uint8) *MBC5 {
	ramSize := uint32(ramBankCount) * 0x2000
	return &MBC5{
		rom:        romData,
		ram:        make([]uint8, ramSize),
		romBank:    1,
		ramEnabled: false,
		hasRumble:  hasRumble,
	}
}

func (m *MBC5) Read(addr uint16) uint8 {
	switch {
	case addr <= 0x3FFF:
		return m.rom[addr]
	case addr >= 0x4000 && addr <= 0x7FFF:
		offset := uint32(m.romBank) * 0x4000
		if offset >= uint32(len(m.rom)) {
			offset = offset % uint32(len(m.rom))
		}
		return m.rom[offset+uint32(addr-0x4000)]
	case addr >= 0xA000 && addr <= 0xBFFF:
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(m.ramBank) * 0x2000
		if offset >= uint32(len(m.ram)) {
			offset = offset % uint32(len(m.ram))
		}
		return m.ram[offset+uint32(addr-0xA000)]
	default:
		return 0xFF
	}
}

func (m *MBC5) Write(addr uint16, value uint8) uint8 {
	switch {
	case addr <= 0x1FFF:
		m.ramEnabled = (value & 0x0F) == 0x0A
	case addr >= 0x2000 && addr <= 0x2FFF:
		m.romBank = (m.romBank & 0x100) | uint16(value)
	case addr >= 0x3000 && addr <= 0x3FFF:
		m.romBank = (m.romBank & 0xFF) | (uint16(value&0x01) << 8)
	case addr >= 0x4000 && addr <= 0x5FFF:
		m.ramBank = value & 0x0F
	case addr >= 0xA000 && addr <= 0xBFFF:
		if !m.ramEnabled {
			return 0xFF
		}
		offset := uint32(m.ramBank) * 0x2000
		if offset >= uint32(len(m.ram)) {
			offset = offset % uint32(len(m.ram))
		}
		m.ram[offset+uint32(addr-0xA000)] = value
	}
	return value
}
