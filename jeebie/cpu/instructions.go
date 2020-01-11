package cpu

import "github.com/valerio/go-jeebie/jeebie/bit"

func (c *CPU) pushStack(r uint16) {
	c.sp--
	c.memory.Write(c.sp, bit.Low(r))
	c.sp--
	c.memory.Write(c.sp, bit.High(r))
}

func (c *CPU) popStack() uint16 {
	high := c.memory.Read(c.sp)
	c.sp++
	low := c.memory.Read(c.sp)
	c.sp++

	return bit.Combine(high, low)
}

func (c *CPU) inc(r *uint8) {
	*r++
	value := *r

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
	c.resetFlag(subFlag)
}

func (c *CPU) dec(r *uint8) {
	*r--
	value := *r

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0xF)
	c.setFlag(subFlag)
}

func (c *CPU) rlc(r *uint8) {
	value := *r

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | (value >> 7)
	*r = value
}

func (c *CPU) rl(r *uint8) {
	value := *r
	carry := c.flagToBit(carryFlag)

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value << 1) | carry
	*r = value
}

func (c *CPU) rrc(r *uint8) {
	value := *r

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | ((value & 1) << 7)
	*r = value
}

func (c *CPU) rr(r *uint8) {
	value := *r
	carry := c.flagToBit(carryFlag) << 7

	c.setFlagToCondition(carryFlag, value > 0x7F)
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | carry
	*r = value
}

// add sets the result of adding an 8 bit register to A, while setting all relevant flags.
func (c *CPU) addToA(value uint8) {
	a := c.a
	result := a + value

	carry := (uint16(a) + uint16(value)) > 0xFF
	halfCarry := (a&0xF)+(value&0xF) > 0xF

	c.setFlagToCondition(zeroFlag, result == 0)
	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.a = result
}

// addToHL sets the result of adding a 16 bit register to HL, while setting relevant flags.
func (c *CPU) addToHL(reg uint16) {
	hl := bit.Combine(c.h, c.l)
	result := hl + reg

	carry := (uint32(hl) + uint32(reg)) > 0xFFFF
	halfCarry := (hl&0xFFF)+(reg&0xFFF) > 0xFFF

	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, carry)
	c.setFlagToCondition(halfCarryFlag, halfCarry)

	c.h = bit.High(result)
	c.l = bit.Low(result)
}

// sub will subtract the value from register A and set all relevant flags.
func (c *CPU) sub(value uint8) {
	a := c.a
	c.a = a - value

	c.setFlagToCondition(zeroFlag, c.a == 0)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, a < value)
	c.setFlagToCondition(halfCarryFlag, (int(a)&0xF)-(int(value)&0xF) < 0)
}

// sbc will subtract the value and carry (1 if set, 0 otherwise) from the register A.
func (c *CPU) sbc(value uint8) {
	a := c.a
	carry := 0
	if c.isSetFlag(carryFlag) {
		carry = 1
	}

	result := int(c.a) - int(value) - carry
	c.a = uint8(result)

	c.setFlagToCondition(zeroFlag, result == 0)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, result < 0)
	c.setFlagToCondition(halfCarryFlag, (int(a)&0xF)-(int(value)&0xF)-carry < 0)
}

func (c *CPU) and(value uint8) {
	c.a &= value
	c.setFlagToCondition(zeroFlag, c.a == 0)
	c.setFlag(halfCarryFlag)
}

func (c *CPU) or(value uint8) {
	c.a |= value
	c.setFlagToCondition(zeroFlag, c.a == 0)
	c.resetFlag(subFlag)
	c.resetFlag(carryFlag)
	c.resetFlag(halfCarryFlag)
}

func (c *CPU) xor(value uint8) {
	c.a ^= value
	c.setFlagToCondition(zeroFlag, c.a == 0)
	c.resetFlag(subFlag)
	c.resetFlag(carryFlag)
	c.resetFlag(halfCarryFlag)
}

// jr performs a jump using the immediate value (byte)
func (c *CPU) jr() {
	c.pc += uint16(c.peekImmediate())
}

// jp performs a jump using the immediate value (16 bit word)
func (c *CPU) jp() {
	c.pc += c.peekImmediateWord()
}
