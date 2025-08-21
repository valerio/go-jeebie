package video

import (
	"testing"
)

func TestPaletteMapping(t *testing.T) {
	tests := []struct {
		name     string
		palette  byte
		colorVal int
		expected GBColor
	}{
		{"Default palette 0xE4, color 0", 0xE4, 0, WhiteColor},     // bits 1,0 = 00 → palette 0 → white
		{"Default palette 0xE4, color 1", 0xE4, 1, LightGreyColor}, // bits 3,2 = 01 → palette 1 → light grey
		{"Default palette 0xE4, color 2", 0xE4, 2, DarkGreyColor},  // bits 5,4 = 10 → palette 2 → dark grey
		{"Default palette 0xE4, color 3", 0xE4, 3, BlackColor},     // bits 7,6 = 11 → palette 3 → black
		{"Custom palette 0x1B, color 0", 0x1B, 0, BlackColor},      // bits 1,0 = 11 → palette 3 → black
		{"Custom palette 0x1B, color 1", 0x1B, 1, DarkGreyColor},   // bits 3,2 = 10 → palette 2 → dark grey
		{"Custom palette 0x1B, color 2", 0x1B, 2, LightGreyColor},  // bits 5,4 = 01 → palette 1 → light grey
		{"Custom palette 0x1B, color 3", 0x1B, 3, WhiteColor},      // bits 7,6 = 00 → palette 0 → white
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This simulates the palette mapping logic in gpu.go:251-252
			color := (tt.palette >> (tt.colorVal * 2)) & 0x03
			result := ByteToColor(color)

			if result != tt.expected {
				t.Errorf("Palette %02X, color %d: expected %08X, got %08X (mapped to %d)",
					tt.palette, tt.colorVal, tt.expected, result, color)
			}
		})
	}
}

func TestTilePixelDecoding(t *testing.T) {
	tests := []struct {
		name     string
		low      byte
		high     byte
		bitIndex uint8
		expected int
	}{
		{"Both bits set", 0xFF, 0xFF, 7, 3},     // bit 7 in both = color 3
		{"Low bit only", 0xFF, 0x00, 7, 1},      // bit 7 in low only = color 1
		{"High bit only", 0x00, 0xFF, 7, 2},     // bit 7 in high only = color 2
		{"No bits set", 0x00, 0x00, 7, 0},       // no bits = color 0
		{"Checkered - pos 0", 0xAA, 0x00, 7, 1}, // 0xAA = 10101010, bit 7 = 1, bit 0 = 0 = color 1
		{"Checkered - pos 1", 0xAA, 0x00, 6, 0}, // 0xAA = 10101010, bit 6 = 0, bit 0 = 0 = color 0
		{"Checkered - pos 2", 0xAA, 0x00, 5, 1}, // 0xAA = 10101010, bit 5 = 1, bit 0 = 0 = color 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This simulates the pixel decoding logic in gpu.go:241-247
			pixel := 0
			if (tt.low>>tt.bitIndex)&1 == 1 {
				pixel |= 1
			}
			if (tt.high>>tt.bitIndex)&1 == 1 {
				pixel |= 2
			}

			if pixel != tt.expected {
				t.Errorf("Low=%02X, High=%02X, bit %d: expected color %d, got %d",
					tt.low, tt.high, tt.bitIndex, tt.expected, pixel)
			}
		})
	}
}
