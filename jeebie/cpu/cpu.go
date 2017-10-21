package cpu

import "github.com/valep27/go-jeebie/jeebie/memory"
import "github.com/valep27/go-jeebie/jeebie/bit"

// Flag is one of the 4 possible flags used in the flag register (high part of AF)
type Flag uint8

const (
	zeroFlag      Flag = 0x80
	subFlag            = 0x40
	halfCarryFlag      = 0x20
	carryFlag          = 0x10
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
}

// New returns an uninitialized CPU instance
func New() CPU {
	return CPU{}
}

//Tick emulates a single step during the main loop for the cpu.
func (c *CPU) Tick() {

}

func (c *CPU) getImmediate() uint8 {
	n := c.memory.ReadByte(c.pc.get())
	c.pc.incr()
	return n
}

func (c *CPU) getImmediateWord() uint16 {
	low := c.getImmediate()
	high := c.getImmediate()

	return bit.CombineBytes(low, high)
}

func (c *CPU) getImmediateSigned() int8 {
	return int8(c.getImmediate())
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
