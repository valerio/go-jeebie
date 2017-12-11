package cpu

import "github.com/valep27/go-jeebie/jeebie/memory"
import (
	"github.com/valep27/GChip8/src/util"
	"github.com/valep27/go-jeebie/jeebie/bit"
)

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag            = 0x40
	halfCarryFlag      = 0x20
	carryFlag          = 0x10
)

// Interrupt is the value of one of the 5 possible interrupts for the Z80.
type Interrupt uint8

const (
	vBlankInterrupt     Interrupt = 0x40
	lcdcStatusInterrupt           = 0x48
	timerInterrupt                = 0x50
	serialInterrupt               = 0x58
	joypadInterrupt               = 0x60
)

const BaseInterruptAddress uint16 = 0x40

const (
	InterruptEnableAddress uint16 = 0xFFFF
	InterruptFlagAddress   uint16 = 0xFF0F
)

// CPU is the main struct holding Z80 state
type CPU struct {
	memory *memory.MMU
	af     Register16
	bc     Register16
	de     Register16
	hl     Register16
	sp     Register16
	pc     Register16

	interruptsEnabled bool
}

// New returns an uninitialized CPU instance
func New() *CPU {
	return &CPU{}
}

// Tick emulates a single step during the main loop for the cpu.
func (c *CPU) Tick() {

}

// handleInterrupts checks for an interrupt and handles it if necessary.
func (c *CPU) handleInterrupts() {
	if c.interruptsEnabled == false {
		return
	}

	// retrieve the two masks
	enabledInterruptsMask := c.memory.ReadByte(InterruptEnableAddress)
	firedInterrupts := c.memory.ReadByte(InterruptFlagAddress)

	// if zero, no interrupts that are enabled were fired
	if (enabledInterruptsMask & firedInterrupts) == 0 {
		return
	}

	c.pushStack(c.pc)

	for i := uint8(0); i < 5; i++ {

		if bit.IsBitSet(i, firedInterrupts) {
			// interrupt handlers are offset by 8
			address := uint16(i)*8 + BaseInterruptAddress

			// mark as handled by clearing the bit at i
			c.memory.WriteByte(InterruptFlagAddress, bit.ClearBit(i, firedInterrupts))

			c.pc.set(address)
			c.interruptsEnabled = false

			// only handle one interrupt at a time
			return
		}
	}
}

// peekImmediate returns the byte at the memory address pointed by the PC
// this value is known as immediate ('n' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediate() uint8 {
	n := c.memory.ReadByte(c.pc.get())
	return n
}

// peekImmediateWord returns the two bytes at the memory address pointed by PC and PC+1
// this value is known as immediate ('nn' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekImmediateWord() uint16 {
	low := c.memory.ReadByte(c.pc.get())
	high := c.memory.ReadByte(c.pc.get() + 1)

	return bit.CombineBytes(low, high)
}

// peekSignedImmediate returns signed byte value at the memory address pointed by PC
// this value is known as immediate ('*' in mnemonics), some opcodes use it as a parameter
func (c CPU) peekSignedImmediate() int8 {
	return int8(c.peekImmediate())
}

// readImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readImmediate() uint8 {
	n := c.peekImmediate()
	c.pc.incr()
	return n
}

// readImmediateWord acts similarly as its peek counterpart, but increments the PC twice after reading
func (c *CPU) readImmediateWord() uint16 {
	nn := c.peekImmediateWord()
	c.pc.incr()
	return nn
}

// readSignedImmediate acts similarly as its peek counterpart, but increments the PC once after reading
func (c *CPU) readSignedImmediate() int8 {
	n := c.peekSignedImmediate()
	c.pc.incr()
	return n
}

func (c *CPU) setFlag(flag Flag) {
	c.af.setLow(uint8(flag))
}

func (c *CPU) resetFlag(flag Flag) {
	c.af.setLow(uint8(flag) ^ 0xFF)
}

func (c CPU) isSetFlag(flag Flag) bool {
	return c.af.getHigh()&uint8(flag) != 0
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

func (c *CPU) pushStack(r Register16) {
	c.sp.decr()
	c.memory.WriteByte(c.sp.get(), r.getHigh())
	c.sp.decr()
	c.memory.WriteByte(c.sp.get(), r.getLow())
}

func (c *CPU) popStack() uint16 {
	low := c.memory.ReadByte(c.sp.get())
	c.sp.incr()
	high := c.memory.ReadByte(c.sp.get())
	c.sp.incr()

	return util.CombineBytes(low, high)
}

func (c *CPU) inc(r *Register8) {
	r.incr()
	value := r.get()

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
	c.resetFlag(subFlag)
}

func (c *CPU) dec(r *Register8) {
	r.decr()
	value := r.get()

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
	c.setFlag(subFlag)
}

func (c *CPU) rlc(r *Register8) {
	value := r.get()

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | (value >> 7)
	r.set(value)
}

func (c *CPU) rl(r *Register8) {
	value := r.get()
	carry := c.flagToBit(carryFlag)

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | carry
	r.set(value)
}

func (c *CPU) rrc(r *Register8) {
	value := r.get()

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | ((value & 1) << 7)
	r.set(value)
}

func (c *CPU) rr(r *Register8) {
	value := r.get()
	carry := c.flagToBit(carryFlag) << 7

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | carry
	r.set(value)
}

// add sets the result of adding an 8 bit register to A, while setting all relevant flags.
func (c *CPU) addToA(value uint8) {
	dst := c.af.low
	result := dst.get() + value

	carry := (uint16(dst.get()) + uint16(value)) > 0xFF
	halfCarry := (dst.get()&0xF)+(value&0xF) > 0xF

	c.setFlagToCondition(zeroFlag, result == 0)
	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.af.setLow(result)
}

// addToHL sets the result of adding a 16 bit register to HL, while setting relevant flags.
func (c *CPU) addToHL(reg Register16) {
	dst := c.hl
	result := dst.get() + reg.get()

	carry := (uint32(dst.get()) + uint32(reg.get())) > 0xFFFF
	halfCarry := (dst.get()&0xFFF)+(reg.get()&0xFFF) > 0xFFF

	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.hl.set(result)
}

// sub will subtract the value from register A and set all relevant flags.
func (c *CPU) sub(value uint8) {
	a := c.af.getLow()
	result := a - value

	c.af.setLow(result)

	c.setFlagToCondition(zeroFlag, result == 0)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, a < value)
	c.setFlagToCondition(halfCarryFlag, (a&0xF)-(value&0xF) < 0)
}
