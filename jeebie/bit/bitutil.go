package bit

// Combine combines two 8 bit values into a single 16 bit value.
// The high byte will be the most significant one.
func Combine(high, low uint8) uint16 {
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

// IsSet will check if the bit at the specified index is Set to 1 or not.
func IsSet(index, byte uint8) bool {
	return ((byte >> index) & 1) == 1
}

func IsSet16(index, value uint16) bool {
	return ((value >> index) & 1) == 1
}

// Clear will return the passed byte with the bit at the specified index Set to 0.
func Clear(index, byte uint8) uint8 {
	return byte & ^(1 << index)
}

// Set will return the passed byte with the bit at the specified index Set to 1.
func Set(index, byte uint8) uint8 {
	return byte | (1 << index)
}

// Reset will return the passed byte with the bit at the specified index Set to 0.
func Reset(index, byte uint8) uint8 {
	return byte & ((1 << index) ^ 0xFF)
}

// GetBitValue returns a byte set to the value of the bit at the specified index.
func GetBitValue(index, byte uint8) uint8 {
	if IsSet(index, byte) {
		return 1
	}

	return 0
}

// Low returns the low (LSB) part of a 16 bit number.
func Low(value uint16) uint8 {
	return uint8(value)
}

// Low returns the higg (MSB) part of a 16 bit number.
func High(value uint16) uint8 {
	return uint8(value >> 8)
}

// ExtractBits extracts bits from highBit to lowBit (inclusive)
// Example: ExtractBits(0b11010110, 6, 4) -> 0b101 (extracts bits 6, 5, 4)
func ExtractBits(value uint8, highBit, lowBit uint8) uint8 {
	shift := lowBit
	width := highBit - lowBit + 1
	mask := uint8((1 << width) - 1)
	return (value >> shift) & mask
}
