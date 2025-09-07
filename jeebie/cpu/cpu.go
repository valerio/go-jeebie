package cpu

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// Bus provides the interface for component communication
type Bus interface {
	Read(address uint16) byte
	Write(address uint16, value byte)
	RequestInterrupt(interrupt addr.Interrupt)
	Tick(cycles int)
}

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag       Flag = 0x40
	halfCarryFlag Flag = 0x20
	carryFlag     Flag = 0x10
)

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
	cycles            uint64
	halted            bool

	// haltBug indicates the next instruction should execute with the
	// HALT bug semantics (skip first opcode-byte increment; operands still
	// advance PC). Set by HALT, cleared after the affected instruction.
	haltBug bool

	bus Bus
}

func initializeMemory(bus Bus) {
	bus.Write(addr.P1, 0xCF)
	bus.Write(addr.TIMA, 0x00)
	bus.Write(addr.TMA, 0x00)
	bus.Write(addr.TAC, 0x00)
	bus.Write(addr.LCDC, 0x91)
	bus.Write(addr.SCY, 0x00)
	bus.Write(addr.SCX, 0x00)
	bus.Write(addr.LYC, 0x00)
	bus.Write(addr.BGP, 0xFC)
	bus.Write(addr.OBP0, 0xFF)
	bus.Write(addr.OBP1, 0xFF)
	bus.Write(addr.WY, 0x00)
	bus.Write(addr.WX, 0x00)
	bus.Write(addr.IE, 0x00)

	// TODO: make the audio registers constant
	bus.Write(0xFF10, 0x80) //    ; NR10
	bus.Write(0xFF11, 0xBF) //    ; NR11
	bus.Write(0xFF12, 0xF3) //    ; NR12
	bus.Write(0xFF14, 0xBF) //    ; NR14
	bus.Write(0xFF16, 0x3F) //    ; NR21
	bus.Write(0xFF17, 0x00) //    ; NR22
	bus.Write(0xFF19, 0xBF) //    ; NR24
	bus.Write(0xFF1A, 0x7F) //    ; NR30
	bus.Write(0xFF1B, 0xFF) //    ; NR31
	bus.Write(0xFF1C, 0x9F) //    ; NR32
	bus.Write(0xFF1E, 0xBF) //    ; NR33
	bus.Write(0xFF20, 0xFF) //    ; NR41
	bus.Write(0xFF21, 0x00) //    ; NR42
	bus.Write(0xFF22, 0x00) //    ; NR43
	bus.Write(0xFF23, 0xBF) //    ; NR30
	bus.Write(0xFF24, 0x77) //    ; NR50
	bus.Write(0xFF25, 0xF3) //    ; NR51
	bus.Write(0xFF26, 0xF1) //    ; NR52  -- should be 0xF0 on SGB
}

// New returns an initialized CPU instance
func New(bus Bus) *CPU {
	initializeMemory(bus)

	cpu := &CPU{
		bus: bus,
	}

	cpu.setAF(0x01B0)
	cpu.setBC(0x0013)
	cpu.setDE(0x00D8)
	cpu.setHL(0x014D)
	cpu.sp = 0xFFFE
	cpu.pc = 0x0100

	return cpu
}

// Exec executes a single CPU instruction without ticking components.
// Returns the amount of cycles that execution has taken.
func (c *CPU) Exec() int {
	// Check for interrupts - this may wake from HALT
	interruptPending := c.handleInterrupts()

	if c.halted {
		// Check if we should wake from HALT (IE & IF != 0)
		if interruptPending {
			// Waking from HALT: do NOT trigger the HALT bug here.
			// The HALT bug only occurs when executing the HALT instruction
			// while IME=0 and an interrupt is pending.
			c.halted = false
		} else {
			// Still halted, consume cycles
			// Note: When halted, we need to tick components manually
			c.bus.Tick(4)
			return 4
		}
	}

	instruction := Decode(c)

	// Previous instruction triggered the halt bug, we have to skip the first PC increment,
	// then, after running the instruction, we clear the halt bug flag.
	skipFirstPCInc := c.haltBug
	if !skipFirstPCInc {
		c.pc++
	}
	if bit.High(c.currentOpcode) == 0xCB {
		c.pc++
	}

	cycles := instruction(c)
	c.cycles += uint64(cycles)

	// Clear halt bug flag IF we skipped the first PC increment this instruction.
	if skipFirstPCInc {
		c.haltBug = false
	}

	// Handle EI delay: enable interrupts after this instruction
	if c.eiPending {
		c.eiPending = false
		c.interruptsEnabled = true
	}

	return cycles
}

// handleInterrupts checks for an interrupt and handles it if necessary.
// Returns true if there are pending interrupts (IE & IF != 0).
func (c *CPU) handleInterrupts() bool {
	// retrieve the two masks
	enabledInterruptsMask := c.bus.Read(addr.IE)
	firedInterrupts := c.bus.Read(addr.IF)

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
			c.bus.Write(addr.IF, bit.Clear(i, firedInterrupts))

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
	n := c.bus.Read(c.pc)
	return n
}

// peekImmediateWord returns the two bytes at the memory address pointed by PC and PC+1
// this value is known as immediate ('nn' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediateWord() uint16 {
	low := c.bus.Read(c.pc)
	high := c.bus.Read(c.pc + 1)
	return bit.Combine(high, low)
}

// peekSignedImmediate returns signed byte value at the memory address pointed by PC
// this value is known as immediate ('*' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekSignedImmediate() int8 {
	return int8(c.peekImmediate())
}

// readImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readImmediate() uint8 {
	var n uint8

	// During the halt bug, the first operand byte re-reads the opcode byte (offset 0).
	if c.haltBug {
		offset := uint16(0)
		n = c.bus.Read(c.pc + offset)
		// Even under the halt bug, operand reads still advance PC
		c.pc++
	} else {
		n = c.peekImmediate()
		c.pc++
	}

	return n
}

// readImmediateWord acts similarly as its peek counterpart, but increments the PC twice after reading
func (c *CPU) readImmediateWord() uint16 {
	var nn uint16

	// During the halt bug, the first operand byte re-reads the opcode byte.
	if c.haltBug {
		low := c.bus.Read(c.pc + 0)
		high := c.bus.Read(c.pc + 1)
		nn = bit.Combine(high, low)
		// Even under the halt bug, operand reads still advance PC
		c.pc += 2
	} else {
		nn = c.peekImmediateWord()
		c.pc += 2
	}

	return nn
}

// readSignedImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readSignedImmediate() int8 {
	var n int8

	// During the halt bug, the first operand byte re-reads the opcode byte.
	if c.haltBug {
		n = int8(c.bus.Read(c.pc + 0))
		// Even under the halt bug, operand reads still advance PC
		c.pc++
	} else {
		n = c.peekSignedImmediate()
		c.pc++
	}

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
func (c *CPU) GetIE() uint8   { return c.bus.Read(0xFFFF) }
func (c *CPU) GetIF() uint8   { return c.bus.Read(0xFF0F) }

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
