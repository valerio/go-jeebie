package util

// Register8 represents an 8-util register in the CPU
type Register8 uint8

func (r Register8) Get() uint8 {
	return uint8(r)
}

func (r *Register8) Set(value uint8) {
	*r = Register8(value)
}

func (r *Register8) Incr() {
	value := r.Get() + 1
	*r = Register8(value)
}

func (r *Register8) Decr() {
	value := r.Get() - 1
	*r = Register8(value)
}

// Register16 represents a 16-util register in the CPU
type Register16 struct {
	high Register8
	low  Register8
}

func newRegister16(value uint16) Register16 {
	r := Register16{}
	r.Set(value)
	return r
}

func (r Register16) GetLow() uint8 {
	return uint8(r.low)
}

func (r Register16) GetHigh() uint8 {
	return uint8(r.high)
}

func (r *Register16) Set(value uint16) {
	r.low = Register8(value)
	r.high = Register8(value >> 8)
}

func (r *Register16) SetHigh(high uint8) {
	r.high = Register8(high)
}

func (r *Register16) SetLow(low uint8) {
	r.low = Register8(low)
}

func (r Register16) Get() uint16 {
	return (uint16(r.high) << 8) | uint16(r.low)
}

func (r *Register16) Incr() {
	value := r.Get() + 1
	r.Set(value)
}

func (r *Register16) Decr() {
	value := r.Get() - 1
	r.Set(value)
}
