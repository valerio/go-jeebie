package bit

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

func IsBitSet(index, byte uint8) bool {
	return ((byte >> index) & 1) == 1
}
