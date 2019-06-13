package util

// Register8 represents an 8-bit register in the CPU
type Register8 uint8

// Get retrieves the the register as a byte
func (r Register8) Get() uint8 {
	return uint8(r)
}

// Set will replace the register value with the given byte
func (r *Register8) Set(value uint8) {
	*r = Register8(value)
}

// Incr adds 1 to the value of the register.
// The value will wrap if it overflows, meaning it will go from 255 to 0.
func (r *Register8) Incr() {
	value := r.Get() + 1
	*r = Register8(value)
}

// Decr decrements the register by 1.
// The value will wrap if it borrows, meaning it will go from 0 to 255.
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

// GetLow returns the lower byte of the register (LSB).
func (r Register16) GetLow() uint8 {
	return uint8(r.Low)
}

// GetHigh returns the high part of the the register (MSB).
func (r Register16) GetHigh() uint8 {
	return uint8(r.High)
}

// Set replaces the value of the register with a 16 bit integer.
func (r *Register16) Set(value uint16) {
	r.Low = Register8(value)
	r.High = Register8(value >> 8)
}

// SetHigh sets the most significant byte (MSB) to the given byte value.
func (r *Register16) SetHigh(high uint8) {
	r.High = Register8(high)
}

// SetLow sets the least significant byte (LSB) to the given byte value.
func (r *Register16) SetLow(low uint8) {
	r.Low = Register8(low)
}

// Get returns the value of the register as a 16 bit unsigned integer.
func (r Register16) Get() uint16 {
	return (uint16(r.High) << 8) | uint16(r.Low)
}

// Incr increments the value of the register by 1, wrapping to 0 if it overflows.
func (r *Register16) Incr() {
	value := r.Get() + 1
	r.Set(value)
}

// Decr decrements the value of the register, wrapping to the max value if it borrows.
func (r *Register16) Decr() {
	value := r.Get() - 1
	r.Set(value)
}
