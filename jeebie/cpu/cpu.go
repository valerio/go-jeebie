package cpu

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag            = 0x40
	halfCarryFlag      = 0x20
	carryFlag          = 0x10
)

var timerFrequencies = map[uint8]int{
	0: 1024,
	1: 16,
	2: 64,
	3: 256,
}

const (
	baseInterruptAddress uint16 = 0x40
)

// CPU is the main struct holding Z80 state
type CPU struct {
	// registers
	a  uint8
	f  uint8
	b  uint8
	c  uint8
	d  uint8
	e  uint8
	h  uint8
	l  uint8
	sp uint16
	pc uint16

	// metadata
	interruptsEnabled bool
	eiPending         bool // EI delay: interrupts enable after next instruction
	currentOpcode     uint16
	stopped           bool
	halted            bool
	haltBug           bool // HALT bug: PC doesn't increment properly
	cycles            uint64
	divCycles         int
	timaCycles        int

	memory *memory.MMU
}

func initializeMemory(mmu *memory.MMU) {
	mmu.Write(addr.TIMA, 0x00)
	mmu.Write(addr.TMA, 0x00)
	mmu.Write(addr.TAC, 0x00)
	mmu.Write(addr.LCDC, 0x91)
	mmu.Write(addr.SCY, 0x00)
	mmu.Write(addr.SCX, 0x00)
	mmu.Write(addr.LYC, 0x00)
	mmu.Write(addr.BGP, 0xFC)
	mmu.Write(addr.OBP0, 0xFF)
	mmu.Write(addr.OBP1, 0xFF)
	mmu.Write(addr.WY, 0x00)
	mmu.Write(addr.WX, 0x00)
	mmu.Write(addr.IE, 0x00)

	// TODO: make the audio registers constant
	mmu.Write(0xFF10, 0x80) //    ; NR10
	mmu.Write(0xFF11, 0xBF) //    ; NR11
	mmu.Write(0xFF12, 0xF3) //    ; NR12
	mmu.Write(0xFF14, 0xBF) //    ; NR14
	mmu.Write(0xFF16, 0x3F) //    ; NR21
	mmu.Write(0xFF17, 0x00) //    ; NR22
	mmu.Write(0xFF19, 0xBF) //    ; NR24
	mmu.Write(0xFF1A, 0x7F) //    ; NR30
	mmu.Write(0xFF1B, 0xFF) //    ; NR31
	mmu.Write(0xFF1C, 0x9F) //    ; NR32
	mmu.Write(0xFF1E, 0xBF) //    ; NR33
	mmu.Write(0xFF20, 0xFF) //    ; NR41
	mmu.Write(0xFF21, 0x00) //    ; NR42
	mmu.Write(0xFF22, 0x00) //    ; NR43
	mmu.Write(0xFF23, 0xBF) //    ; NR30
	mmu.Write(0xFF24, 0x77) //    ; NR50
	mmu.Write(0xFF25, 0xF3) //    ; NR51
	mmu.Write(0xFF26, 0xF1) //    ; NR52  -- should be 0xF0 on SGB
}

// New returns an initialized CPU instance
func New(memory *memory.MMU) *CPU {
	initializeMemory(memory)

	cpu := &CPU{
		memory: memory,
	}

	cpu.setAF(0x01B0)
	cpu.setBC(0x0013)
	cpu.setDE(0x00D8)
	cpu.setHL(0x014D)
	cpu.sp = 0xFFFE
	cpu.pc = 0x0100

	return cpu
}

// Tick emulates a single step during the main loop for the cpu.
// Returns the amount of cycles that execution has taken.
func (c *CPU) Tick() int {
	// Check for interrupts - this may wake from HALT
	interruptPending := c.handleInterrupts()

	if c.halted {
		// Check if we should wake from HALT (IE & IF != 0)
		if interruptPending {
			c.halted = false
			// If IME=0 and interrupt pending, we have the HALT bug scenario
			if !c.interruptsEnabled {
				c.haltBug = true
			}
		} else {
			// Still halted, consume cycles
			return 4
		}
	}

	// Handle HALT bug: don't increment PC for next instruction
	if c.haltBug {
		c.haltBug = false
		// Execute instruction without incrementing PC first
		// This causes the instruction after HALT to be read twice
	}

	instruction := Decode(c)
	cycles := instruction(c)
	c.cycles += uint64(cycles)

	// Handle EI delay: enable interrupts after this instruction
	if c.eiPending {
		c.eiPending = false
		c.interruptsEnabled = true
	}

	c.updateTimers(cycles)

	return cycles
}

// handleInterrupts checks for an interrupt and handles it if necessary.
// Returns true if there are pending interrupts (IE & IF != 0).
func (c *CPU) handleInterrupts() bool {
	// retrieve the two masks
	enabledInterruptsMask := c.memory.Read(addr.IE)
	firedInterrupts := c.memory.Read(addr.IF)

	// check if any enabled interrupts are pending
	pendingInterrupts := (enabledInterruptsMask & firedInterrupts) != 0

	if !c.interruptsEnabled {
		return pendingInterrupts
	}

	if !pendingInterrupts {
		return false
	}

	// service interrupts in priority order (bit 0 = highest)
	for i := uint8(0); i < 5; i++ {
		if bit.IsSet(i, firedInterrupts) && bit.IsSet(i, enabledInterruptsMask) {
			// interrupt handlers are offset by 8
			// 0x40 - 0x48 - 0x50 - 0x58 - 0x60
			address := uint16(i)*8 + baseInterruptAddress

			// mark as handled by clearing the bit at i
			c.memory.Write(addr.IF, bit.Clear(i, firedInterrupts))

			// move PC to interrupt handler address
			c.pushStack(c.pc)
			c.pc = address

			// add cycles equivalent to a JMP.
			c.cycles += 20

			// disable interrupts
			c.interruptsEnabled = false

			// return on the first served interrupt.
			return true
		}
	}

	return pendingInterrupts
}

// peekImmediate returns the byte at the memory address pointed by the PC
// this value is known as immediate ('n' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediate() uint8 {
	n := c.memory.Read(c.pc)
	return n
}

// peekImmediateWord returns the two bytes at the memory address pointed by PC and PC+1
// this value is known as immediate ('nn' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediateWord() uint16 {
	low := c.memory.Read(c.pc)
	high := c.memory.Read(c.pc + 1)

	return bit.Combine(high, low)
}

// peekSignedImmediate returns signed byte value at the memory address pointed by PC
// this value is known as immediate ('*' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekSignedImmediate() int8 {
	return int8(c.peekImmediate())
}

// readImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readImmediate() uint8 {
	n := c.peekImmediate()
	c.pc++
	return n
}

// readImmediateWord acts similarly as its peek counterpart, but increments the PC twice after reading
func (c *CPU) readImmediateWord() uint16 {
	nn := c.peekImmediateWord()
	c.pc += 2
	return nn
}

// readSignedImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readSignedImmediate() int8 {
	n := c.peekSignedImmediate()
	c.pc++
	return n
}

func (c *CPU) setFlag(flag Flag) {
	c.f |= uint8(flag)
}

func (c *CPU) resetFlag(flag Flag) {
	c.f &= uint8(flag ^ 0xFF)
}

func (c CPU) isSetFlag(flag Flag) bool {
	return c.f&uint8(flag) != 0
}

// flagToBit will return 1 if the passed flag is set, 0 otherwise
func (c CPU) flagToBit(flag Flag) uint8 {
	if c.isSetFlag(flag) {
		return 1
	}

	return 0
}

func (c *CPU) setFlagToCondition(flag Flag, condition bool) {
	if !condition {
		c.resetFlag(flag)
		return
	}

	c.setFlag(flag)
}

func (c *CPU) setBC(value uint16) {
	c.b = bit.High(value)
	c.c = bit.Low(value)
}

func (c CPU) getBC() uint16 {
	return bit.Combine(c.b, c.c)
}

func (c *CPU) setDE(value uint16) {
	c.d = bit.High(value)
	c.e = bit.Low(value)
}

func (c CPU) getDE() uint16 {
	return bit.Combine(c.d, c.e)
}

func (c *CPU) setHL(value uint16) {
	c.h = bit.High(value)
	c.l = bit.Low(value)
}

func (c CPU) getHL() uint16 {
	return bit.Combine(c.h, c.l)
}

func (c *CPU) setAF(value uint16) {
	c.a = bit.High(value)
	// F register lower 4 bits must be 0
	c.f = bit.Low(value) & 0xF0
}

func (c CPU) getAF() uint16 {
	return bit.Combine(c.a, c.f)
}

func (c *CPU) updateTimers(cycles int) {
	c.divCycles += cycles
	if c.divCycles >= 256 {
		c.divCycles -= 256
		c.memory.Write(addr.DIV, c.memory.Read(addr.DIV)+1)
	}

	tac := c.memory.Read(addr.TAC)
	if !bit.IsSet(2, tac) {
		return
	}

	c.timaCycles += cycles
	frequency := timerFrequencies[tac&0x03]

	for c.timaCycles >= frequency {
		c.timaCycles -= frequency
		tima := c.memory.Read(addr.TIMA)
		if tima == 0xFF {
			tma := c.memory.Read(addr.TMA)
			c.memory.Write(addr.TIMA, tma)
			c.memory.RequestInterrupt(addr.TimerInterrupt)
		} else {
			c.memory.Write(addr.TIMA, tima+1)
		}
	}
}

// Debug getter methods for register display
func (c *CPU) GetA() uint8       { return c.a }
func (c *CPU) GetF() uint8       { return c.f }
func (c *CPU) GetB() uint8       { return c.b }
func (c *CPU) GetC() uint8       { return c.c }
func (c *CPU) GetD() uint8       { return c.d }
func (c *CPU) GetE() uint8       { return c.e }
func (c *CPU) GetH() uint8       { return c.h }
func (c *CPU) GetL() uint8       { return c.l }
func (c *CPU) GetSP() uint16     { return c.sp }
func (c *CPU) GetPC() uint16     { return c.pc }
func (c *CPU) GetCycles() uint64 { return c.cycles }

// Interrupt state getters
func (c *CPU) GetIME() bool   { return c.interruptsEnabled }
func (c *CPU) IsHalted() bool { return c.halted }
func (c *CPU) GetIE() uint8   { return c.memory.Read(0xFFFF) }
func (c *CPU) GetIF() uint8   { return c.memory.Read(0xFF0F) }

// GetPendingInterrupts returns which interrupts are both enabled and requested
func (c *CPU) GetPendingInterrupts() uint8 {
	ie := c.GetIE()
	iFlag := c.GetIF()
	return ie & iFlag & 0x1F
}

// GetFlagString returns a human-readable representation of the flag register
func (c *CPU) GetFlagString() string {
	flags := ""
	if c.f&uint8(zeroFlag) != 0 {
		flags += "Z"
	} else {
		flags += "-"
	}
	if c.f&uint8(subFlag) != 0 {
		flags += "N"
	} else {
		flags += "-"
	}
	if c.f&uint8(halfCarryFlag) != 0 {
		flags += "H"
	} else {
		flags += "-"
	}
	if c.f&uint8(carryFlag) != 0 {
		flags += "C"
	} else {
		flags += "-"
	}
	return flags
}

// ResetTimerCycles resets the TIMA timer cycle counter when TIMA is written to
func (c *CPU) ResetTimerCycles() {
	c.timaCycles = 0
}
