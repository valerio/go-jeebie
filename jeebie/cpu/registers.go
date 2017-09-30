package cpu

// Register8 represents an 8-bit register in the CPU
type Register8 uint8

func (r Register8) get() uint8 {
	return uint8(r)
}

func (r *Register8) set(value uint8) {
	*r = Register8(value)
}

func (r *Register8) incr() {
	value := r.get() + 1
	*r = Register8(value)
}

func (r *Register8) decr() {
	value := r.get() - 1
	*r = Register8(value)
}


// Register16 represents a 16-bit register in the CPU
type Register16 uint16

func (r Register16) low() Register8 {
	return Register8(r & 0xFF)
}

func (r Register16) high() Register8 {
	return Register8((r >> 8) & 0xFF)
}

func (r Register16) getLow() uint8 {
	return uint8(r.low())
}

func (r Register16) getHigh() uint8 {
	return uint8(r.high())
}

func (r *Register16) set(value uint16) {
	*r = Register16(value)
}

func (r *Register16) setHigh(high uint8) {
	result := uint16(high)<<8 + uint16(*r)&0x00FF
	*r = Register16(result)
}

func (r *Register16) setLow(low uint8) {
	result := uint16(*r)&0xFF00 + uint16(low)
	*r = Register16(result)
}

func (r Register16) get() uint16 {
	return uint16(r)
}

func (r *Register16) incr() {
	value := r.get() + 1
	*r = Register16(value)
}

func (r *Register16) decr() {
	value := r.get() - 1
	*r = Register16(value)
}
