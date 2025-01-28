package bit

import (
	"testing"
)

func TestCombine(t *testing.T) {
	tests := []struct {
		high, low uint8
		expected  uint16
	}{
		{0xAB, 0xCD, 0xABCD},
		{0x00, 0x00, 0x0000},
		{0xFF, 0xFF, 0xFFFF},
		{0x12, 0x34, 0x1234},
	}

	for _, tt := range tests {
		result := Combine(tt.high, tt.low)
		if result != tt.expected {
			t.Errorf("Combine(%X, %X) = %X; want %X", tt.high, tt.low, result, tt.expected)
		}
	}
}

func TestCheckedAdd(t *testing.T) {
	tests := []struct {
		a, b             uint8
		expectedResult   uint8
		expectedOverflow bool
	}{
		{0b11111111, 0b00000001, 0, true},
		{0b11111111, 0b11111111, 254, true},
		{0b00000001, 0b00000001, 2, false},
		{0b10000000, 0b00000000, 128, false},
	}

	for _, tt := range tests {
		result, overflow := CheckedAdd(tt.a, tt.b)
		if result != tt.expectedResult || overflow != tt.expectedOverflow {
			t.Errorf("CheckedAdd(%d, %d) = (%d, %v); want (%d, %v)", tt.a, tt.b, result, overflow, tt.expectedResult, tt.expectedOverflow)
		}
	}
}

func TestCheckedSub(t *testing.T) {
	tests := []struct {
		a, b           uint8
		expectedResult uint8
		expectedBorrow bool
	}{
		{0b00000000, 0b00000001, 255, true},
		{0b00000001, 0b00000001, 0, false},
		{0b10000000, 0b00000000, 128, false},
		{0b11111111, 0b11111111, 0, false},
	}

	for _, tt := range tests {
		result, borrow := CheckedSub(tt.a, tt.b)
		if result != tt.expectedResult || borrow != tt.expectedBorrow {
			t.Errorf("CheckedSub(%d, %d) = (%d, %v); want (%d, %v)", tt.a, tt.b, result, borrow, tt.expectedResult, tt.expectedBorrow)
		}
	}
}

func TestIsSet(t *testing.T) {
	tests := []struct {
		byte     uint8
		index    uint8
		expected bool
	}{
		{0b10101010, 0, false},
		{0b10101010, 1, true},
		{0b10101010, 2, false},
		{0b10101010, 7, true},
		{0b10101010, 8, false},
		{0b10101010, 255, false},
	}

	for _, tt := range tests {
		result := IsSet(tt.index, tt.byte)
		if result != tt.expected {
			t.Errorf("IsSet(%d, %08b) = %v; want %v", tt.index, tt.byte, result, tt.expected)
		}
	}
}

func TestClear(t *testing.T) {
	tests := []struct {
		byte     uint8
		index    uint8
		expected uint8
	}{
		{0b10101010, 1, 0b10101000},
		{0b10101010, 7, 0b00101010},
		{0b10101010, 8, 0b10101010},
		{0b10101010, 255, 0b10101010},
	}

	for _, tt := range tests {
		result := Clear(tt.index, tt.byte)
		if result != tt.expected {
			t.Errorf("Clear(%d, %08b) = %08b; want %08b", tt.index, tt.byte, result, tt.expected)
		}
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		byte     uint8
		index    uint8
		expected uint8
	}{
		{0b10101010, 0, 0b10101011},
		{0b10101010, 2, 0b10101110},
		{0b10101010, 7, 0b10101010},
		{0b10101010, 8, 0b10101010},
		{0b10101010, 255, 0b10101010},
	}

	for _, tt := range tests {
		result := Set(tt.index, tt.byte)
		if result != tt.expected {
			t.Errorf("Set(%d, %08b) = %08b; want %08b", tt.index, tt.byte, result, tt.expected)
		}
	}
}

func TestReset(t *testing.T) {
	tests := []struct {
		byte     uint8
		index    uint8
		expected uint8
	}{
		{0b10101011, 0, 0b10101010},
		{0b10101011, 1, 0b10101001},
		{0b10101011, 7, 0b00101011},
		{0b10101011, 8, 0b10101011},
		{0b10101011, 255, 0b10101011},
	}

	for _, tt := range tests {
		result := Reset(tt.index, tt.byte)
		if result != tt.expected {
			t.Errorf("Reset(%d, %08b) = %08b; want %08b", tt.index, tt.byte, result, tt.expected)
		}
	}
}

func TestGetBitValue(t *testing.T) {
	tests := []struct {
		byte     uint8
		index    uint8
		expected uint8
	}{
		{0b10101010, 0, 0},
		{0b10101010, 1, 1},
		{0b10101010, 2, 0},
		{0b10101010, 7, 1},
		{0b10101010, 8, 0},
		{0b10101010, 255, 0},
	}

	for _, tt := range tests {
		result := GetBitValue(tt.index, tt.byte)
		if result != tt.expected {
			t.Errorf("GetBitValue(%d, %08b) = %d; want %d", tt.index, tt.byte, result, tt.expected)
		}
	}
}

func TestLow(t *testing.T) {
	tests := []struct {
		value    uint16
		expected uint8
	}{
		{0xABCD, 0xCD},
		{0x0000, 0x00},
		{0xFFFF, 0xFF},
		{0x1234, 0x34},
	}

	for _, tt := range tests {
		result := Low(tt.value)
		if result != tt.expected {
			t.Errorf("Low(%X) = %X; want %X", tt.value, result, tt.expected)
		}
	}
}

func TestHigh(t *testing.T) {
	tests := []struct {
		value    uint16
		expected uint8
	}{
		{0xABCD, 0xAB},
		{0x0000, 0x00},
		{0xFFFF, 0xFF},
		{0x1234, 0x12},
	}

	for _, tt := range tests {
		result := High(tt.value)
		if result != tt.expected {
			t.Errorf("High(%X) = %X; want %X", tt.value, result, tt.expected)
		}
	}
}
