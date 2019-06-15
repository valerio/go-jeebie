package cpu

import "github.com/valerio/go-jeebie/jeebie/util"

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
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
	c.resetFlag(subFlag)
}

func (c *CPU) dec(r *util.Register8) {
	r.Decr()
	value := r.Get()

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
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

// jr performs a jump using the immediate value (byte)
func (c *CPU) jr() {
	n := uint16(c.peekImmediate())
	c.pc.Set(c.pc.Get() + n)
}

// jp performs a jump using the immediate value (16 bit word)
func (c *CPU) jp() {
	nn := c.peekImmediateWord()
	c.pc.Set(c.pc.Get() + nn)
}
