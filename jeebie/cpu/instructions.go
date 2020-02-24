package cpu

import (
	"math/bits"

	"github.com/valerio/go-jeebie/jeebie/bit"
)

func (c *CPU) pushStack(r uint16) {
	c.sp--
	c.memory.Write(c.sp, bit.High(r))
	c.sp--
	c.memory.Write(c.sp, bit.Low(r))
}

func (c *CPU) popStack() uint16 {
	low := c.memory.Read(c.sp)
	c.sp++
	high := c.memory.Read(c.sp)
	c.sp++

	return bit.Combine(high, low)
}

func (c *CPU) inc(r *uint8) {
	*r++
	value := *r

	c.setFlagToCondition(zeroFlag, value == 0)
	c.setFlagToCondition(halfCarryFlag, (value&0xF) == 0)
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

	// set carry if bit 7 was set
	c.setFlagToCondition(carryFlag, bit.IsSet(7, value))
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = bits.RotateLeft8(value, 1)

	// if rotating the A register, don't set the zero flag.
	if r != &c.a {
		c.setFlagToCondition(zeroFlag, value == 0)
	}

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

	// if rotating the A register, don't set the zero flag.
	if r != &c.a {
		c.setFlagToCondition(zeroFlag, value == 0)
	}

	*r = value
}

func (c *CPU) rrc(r *uint8) {
	value := *r

	// set carry if bit 0 was set
	c.setFlagToCondition(carryFlag, bit.IsSet(0, value))
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = bits.RotateLeft8(value, -1)

	// if rotating the A register, don't set the zero flag.
	if r != &c.a {
		c.setFlagToCondition(zeroFlag, value == 0)
	}

	*r = value
}

func (c *CPU) rr(r *uint8) {
	value := *r
	carry := c.flagToBit(carryFlag) << 7

	c.setFlagToCondition(carryFlag, bit.IsSet(0, value))
	c.resetFlag(zeroFlag)
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value = (value >> 1) | carry

	// if rotating the A register, don't set the zero flag.
	if r != &c.a {
		c.setFlagToCondition(zeroFlag, value == 0)
	}

	*r = value
}

func (c *CPU) sla(r *uint8) {
	value := *r

	c.setFlagToCondition(carryFlag, bit.IsSet(7, value))
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value <<= 1
	c.setFlagToCondition(zeroFlag, value == 0)
	*r = value
}

func (c *CPU) sra(r *uint8) {
	value := *r

	c.setFlagToCondition(carryFlag, bit.IsSet(0, value))
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	// preserve the MSB
	if bit.IsSet(7, value) {
		value = (value >> 1) | 0x80
	} else {
		value >>= 1
	}

	c.setFlagToCondition(zeroFlag, value == 0)

	*r = value
}

func (c *CPU) srl(r *uint8) {
	value := *r

	c.setFlagToCondition(carryFlag, bit.IsSet(0, value))
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)

	value >>= 1
	c.setFlagToCondition(zeroFlag, value == 0)
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

// adc sets the result of adding an 8 bit register and the carry value to A.
func (c *CPU) adc(value uint8) {
	carry := c.flagToBit(carryFlag)
	a := c.a
	result := a + value + carry

	shouldSetCarry := (uint16(a) + uint16(value) + uint16(carry)) > 0xFF
	shouldSetHalfCarry := (a&0xF)+(value&0xF)+carry > 0xF

	c.setFlagToCondition(zeroFlag, result == 0)
	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, shouldSetCarry)
	c.setFlagToCondition(halfCarryFlag, shouldSetHalfCarry)

	c.a = result
}

// addToHL sets the result of adding a 16 bit register to HL, while setting relevant flags.
func (c *CPU) addToHL(reg uint16) {
	hl := c.getHL()
	result := uint32(hl) + uint32(reg)

	c.resetFlag(subFlag)
	c.setFlagToCondition(carryFlag, (result&0x10000) != 0)
	c.setFlagToCondition(halfCarryFlag, (hl^reg^uint16(result))&0x1000 != 0)

	c.setHL(uint16(result))
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
	carry := c.flagToBit(carryFlag)

	result := int(c.a) - int(value) - int(carry)
	c.a = uint8(result)

	c.setFlagToCondition(zeroFlag, c.a == 0)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, result < 0)
	c.setFlagToCondition(halfCarryFlag, (int(a)&0xF)-(int(value)&0xF)-int(carry) < 0)
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

// Implements the compare (CP) instruction.
func (c *CPU) cp(value uint8) {
	c.setFlagToCondition(zeroFlag, c.a == value)
	c.setFlag(subFlag)
	c.setFlagToCondition(carryFlag, c.a < value)
	c.setFlagToCondition(halfCarryFlag, (c.a-value)&0xF > c.a&0xF)
}

// Implements SWAP, which swaps the upper and lower nibbles (4 bits) of the 8-bit argument.
func (c *CPU) swap(r *uint8) {
	result := *r>>4 + *r<<4

	c.setFlagToCondition(zeroFlag, result == 0)
	c.resetFlag(subFlag)
	c.resetFlag(carryFlag)
	c.resetFlag(halfCarryFlag)

	*r = result
}

// Implements DAA (Decimal Adjust Accumulator).
// It adjusts the A register so that it is valid Binary Coded Decimal (BCD).
func (c *CPU) daa() {
	// use a 16-bit integer to detect overflows and set carry accordingly
	a := uint16(c.a)

	if c.isSetFlag(subFlag) {
		if c.isSetFlag(halfCarryFlag) {
			a = (a - 0x06) & 0xFF
		}
		if c.isSetFlag(carryFlag) {
			a -= 0x60
		}
	} else {
		if c.isSetFlag(halfCarryFlag) || (a&0x0F) > 9 {
			a += 0x06
		}
		if c.isSetFlag(carryFlag) || a > 0x9F {
			a += 0x60
		}

	}

	c.resetFlag(halfCarryFlag)
	c.setFlagToCondition(zeroFlag, a == 0)

	// detect overflow
	if (a & 0x100) == 0x100 {
		c.setFlag(carryFlag)
	}

	a &= 0xFF

	c.a = uint8(a)
}

// cpl complements the A register (flips all bits).
func (c *CPU) cpl() {
	c.a ^= 0xFF
	c.setFlag(subFlag)
	c.setFlag(halfCarryFlag)
}

// ccf complements the carry flag.
func (c *CPU) ccf() {
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)
	// flip the carry flag
	c.setFlagToCondition(carryFlag, !c.isSetFlag(carryFlag))
}

// scf sets the carry flag
func (c *CPU) scf() {
	c.resetFlag(subFlag)
	c.resetFlag(halfCarryFlag)
	c.setFlag(carryFlag)
}

// bit (BIT) tests if the bit b in register r is set or not.
func (c *CPU) bit(b, r uint8) {
	isSet := bit.IsSet(b, r)

	if !isSet {
		c.setFlag(zeroFlag)
	}
	c.resetFlag(subFlag)
	c.setFlag(halfCarryFlag)
}

// set (SET) sets bit b in register r
func (c *CPU) set(b uint8, r *uint8) {
	*r = bit.Set(b, *r)
}

// res (RES) resets bit b in register r
func (c *CPU) res(b uint8, r *uint8) {
	*r = bit.Reset(b, *r)
}

func (c *CPU) jr() {
	addr := uint16(int(c.pc) + int(c.readSignedImmediate()))
	c.pc = addr + 1
}
