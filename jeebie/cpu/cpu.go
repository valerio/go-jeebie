package cpu

import (
	"github.com/valep27/go-jeebie/jeebie/memory"
	"github.com/valep27/go-jeebie/jeebie/util"
)

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag            = 0x40
	halfCarryFlag      = 0x20
	carryFlag          = 0x10
)

const (
	baseInterruptAddress   uint16 = 0x40
	interruptEnableAddress        = 0xFFFF
	interruptFlagAddress          = 0xFF0F
)

// CPU is the main struct holding Z80 state
type CPU struct {
	memory *memory.MMU
	af     util.Register16
	bc     util.Register16
	de     util.Register16
	hl     util.Register16
	sp     util.Register16
	pc     util.Register16

	interruptsEnabled bool
	currentOpcode     uint16
}

// New returns an uninitialized CPU instance
func New(memory *memory.MMU) *CPU {
	return &CPU{
		memory: memory,
		pc:     util.Register16{High: 0x01},
	}
}

// Tick emulates a single step during the main loop for the cpu.
// Returns the amount of cycles that execution has taken.
func (c *CPU) Tick() int {
	c.handleInterrupts()

	instruction := Decode(c)
	cycles := instruction(c)

	return cycles
}

// handleInterrupts checks for an interrupt and handles it if necessary.
func (c *CPU) handleInterrupts() {
	if c.interruptsEnabled == false {
		return
	}

	// retrieve the two masks
	enabledInterruptsMask := c.memory.ReadByte(interruptEnableAddress)
	firedInterrupts := c.memory.ReadByte(interruptFlagAddress)

	// if zero, no interrupts that are enabled were fired
	if (enabledInterruptsMask & firedInterrupts) == 0 {
		return
	}

	c.pushStack(c.pc)

	for i := uint8(0); i < 5; i++ {

		if util.IsBitSet(i, firedInterrupts) {
			// interrupt handlers are offset by 8
			address := uint16(i)*8 + baseInterruptAddress

			// mark as handled by clearing the bit at i
			c.memory.WriteByte(interruptFlagAddress, util.ClearBit(i, firedInterrupts))

			c.pc.Set(address)
			c.interruptsEnabled = false

			// only handle one interrupt at a time
			return
		}
	}
}

// peekImmediate returns the byte at the memory address pointed by the PC
// this value is known as immediate ('n' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediate() uint8 {
	n := c.memory.ReadByte(c.pc.Get())
	return n
}

// peekImmediateWord returns the two bytes at the memory address pointed by PC and PC+1
// this value is known as immediate ('nn' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediateWord() uint16 {
	low := c.memory.ReadByte(c.pc.Get())
	high := c.memory.ReadByte(c.pc.Get() + 1)

	return util.CombineBytes(low, high)
}

// peekSignedImmediate returns signed byte value at the memory address pointed by PC
// this value is known as immediate ('*' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekSignedImmediate() int8 {
	return int8(c.peekImmediate())
}

// readImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readImmediate() uint8 {
	n := c.peekImmediate()
	c.pc.Incr()
	return n
}

// readImmediateWord acts similarly as its peek counterpart, but increments the PC twice after reading
func (c *CPU) readImmediateWord() uint16 {
	nn := c.peekImmediateWord()
	c.pc.Incr()
	c.pc.Incr()
	return nn
}

// readSignedImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readSignedImmediate() int8 {
	n := c.peekSignedImmediate()
	c.pc.Incr()
	return n
}

func (c *CPU) setFlag(flag Flag) {
	c.af.SetLow(uint8(flag))
}

func (c *CPU) resetFlag(flag Flag) {
	c.af.SetLow(uint8(flag) ^ 0xFF)
}

func (c CPU) isSetFlag(flag Flag) bool {
	return c.af.GetHigh()&uint8(flag) != 0
}

// flagToBit will return 1 if the passed flag is set, 0 otherwise
func (c CPU) flagToBit(flag Flag) uint8 {
	if c.isSetFlag(flag) {
		return 1
	}

	return 0
}

func (c *CPU) setFlagToCondition(flag Flag, condition bool) {
	if condition {
		c.setFlag(flag)
	} else {
		c.resetFlag(flag)
	}
}

func (c *CPU) pushStack(r util.Register16) {
	c.sp.Decr()
	c.memory.WriteByte(c.sp.Get(), r.GetHigh())
	c.sp.Decr()
	c.memory.WriteByte(c.sp.Get(), r.GetLow())
}

func (c *CPU) popStack() uint16 {
	low := c.memory.ReadByte(c.sp.Get())
	c.sp.Incr()
	high := c.memory.ReadByte(c.sp.Get())
	c.sp.Incr()

	return util.CombineBytes(low, high)
}

func (c *CPU) inc(r *util.Register8) {
	r.Incr()
	value := r.Get()

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value & 0xF) == 0xF)
	c.resetFlag(subFlag)
}

func (c *CPU) dec(r *util.Register8) {
	r.Decr()
	value := r.Get()

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value & 0xF) == 0xF)
	c.setFlag(subFlag)
}

func (c *CPU) rlc(r *util.Register8) {
	value := r.Get()

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | (value >> 7)
	r.Set(value)
}

func (c *CPU) rl(r *util.Register8) {
	value := r.Get()
	carry := c.flagToBit(carryFlag)

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | carry
	r.Set(value)
}

func (c *CPU) rrc(r *util.Register8) {
	value := r.Get()

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | ((value & 1) << 7)
	r.Set(value)
}

func (c *CPU) rr(r *util.Register8) {
	value := r.Get()
	carry := c.flagToBit(carryFlag) << 7

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | carry
	r.Set(value)
}

// add sets the result of adding an 8 bit register to A, while setting all relevant flags.
func (c *CPU) addToA(value uint8) {
	a := c.af.GetLow()
	result := a + value

	carry := (uint16(a) + uint16(value)) > 0xFF
	halfCarry := (a&0xF)+(value&0xF) > 0xF

	c.setFlagToCondition(zeroFlag, result == 0)
	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.af.SetLow(result)
}

// addToHL sets the result of adding a 16 bit register to HL, while setting relevant flags.
func (c *CPU) addToHL(reg util.Register16) {
	dst := c.hl
	result := dst.Get() + reg.Get()

	carry := (uint32(dst.Get()) + uint32(reg.Get())) > 0xFFFF
	halfCarry := (dst.Get()&0xFFF)+(reg.Get()&0xFFF) > 0xFFF

	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.hl.Set(result)
}

// sub will subtract the value from register A and set all relevant flags.
func (c *CPU) sub(value uint8) {
	a := c.af.GetLow()
	result := a - value

	c.af.SetLow(result)

	c.setFlagToCondition(zeroFlag, result == 0)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, a < value)
	c.setFlagToCondition(halfCarryFlag, (a&0xF)-(value&0xF) < 0)
}
