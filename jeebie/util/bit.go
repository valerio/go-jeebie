package util

// CombineBytes combines two 8 bit values into a single 16 bit value.
// The high byte will be the most significant one.
func CombineBytes(low, high uint8) uint16 {
	return (uint16(high) << 8) | uint16(low)
}

// CheckedAdd adds two 8 bit unsigned values and detects if an overflow happened.
func CheckedAdd(a, b uint8) (result uint8, overflow bool) {
	overflow = false
	highBits := (uint16(a) + uint16(b)) & 0xFF00

	if highBits > 0 {
		overflow = true
	}

	result = a + b
	return
}

// CheckedSub subtracts two 8 bit unsigned values and detects if a borrow happened.
func CheckedSub(a, b uint8) (result uint8, borrow bool) {
	borrow = false
	highBits := (uint16(a) - uint16(b)) & 0xFF00

	if highBits > 0 {
		borrow = true
	}

	result = a - b
	return
}

// IsBitSet will check if the bit at the specified index is Set to 1 or not.
func IsBitSet(index, byte uint8) bool {
	return ((byte >> index) & 1) == 1
}

// ClearBit will return the passed byte with the bit at the specified index Set to 0.
func ClearBit(index, byte uint8) uint8 {
	return byte & ^(1 << index)
}

// SetBit will return the passed byte with the bit at the specified index Set to 1.
func SetBit(index, byte uint8) uint8 {
	return byte | (1 << index)
}

// GetBitValue returns a byte set to the value of the bit at the specified index.
func GetBitValue(index, byte uint8) uint8 {
	if IsBitSet(index, byte) {
		return 1
	}

	return 0
}
