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
type Register16 struct {
	high Register8
	low  Register8
}

func newRegister16(value uint16) Register16 {
	r := Register16{}
	r.set(value)
	return r
}

func (r Register16) getLow() uint8 {
	return uint8(r.low)
}

func (r Register16) getHigh() uint8 {
	return uint8(r.high)
}

func (r *Register16) set(value uint16) {
	r.low = Register8(value)
	r.high = Register8(value >> 8)
}

func (r *Register16) setHigh(high uint8) {
	r.high = Register8(high)
}

func (r *Register16) setLow(low uint8) {
	r.low = Register8(low)
}

func (r Register16) get() uint16 {
	return (uint16(r.high) << 8) | uint16(r.low)
}

func (r *Register16) incr() {
	value := r.get() + 1
	r.set(value)
}

func (r *Register16) decr() {
	value := r.get() - 1
	r.set(value)
}
