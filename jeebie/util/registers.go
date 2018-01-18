package util

// Register8 represents an 8-bit register in the CPU
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

// Register16 represents a 16-bit register in the CPU
type Register16 struct {
	High Register8
	Low  Register8
}

func newRegister16(value uint16) Register16 {
	r := Register16{}
	r.Set(value)
	return r
}

func (r Register16) GetLow() uint8 {
	return uint8(r.Low)
}

func (r Register16) GetHigh() uint8 {
	return uint8(r.High)
}

func (r *Register16) Set(value uint16) {
	r.Low = Register8(value)
	r.High = Register8(value >> 8)
}

func (r *Register16) SetHigh(high uint8) {
	r.High = Register8(high)
}

func (r *Register16) SetLow(low uint8) {
	r.Low = Register8(low)
}

func (r Register16) Get() uint16 {
	return (uint16(r.High) << 8) | uint16(r.Low)
}

func (r *Register16) Incr() {
	value := r.Get() + 1
	r.Set(value)
}

func (r *Register16) Decr() {
	value := r.Get() - 1
	r.Set(value)
}
