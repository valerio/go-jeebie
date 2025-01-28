package memory

import (
	"testing"
)

func TestMBC1(t *testing.T) {
	t.Run("ROM Bank 0 (Fixed)", func(t *testing.T) {
		// Create a fake ROM with recognizable data
		rom := make([]uint8, 0x8000) // 32KB
		for i := range rom {
			rom[i] = uint8(i & 0xFF)
		}

		mbc := NewMBC1(rom, false, 0)

		// Test reading from bank 0 (non-switchable)
		for addr := uint16(0x0000); addr < 0x4000; addr++ {
			got := mbc.Read(addr)
			want := uint8(addr & 0xFF)
			if got != want {
				t.Errorf("Read(0x%04X) = 0x%02X; want 0x%02X", addr, got, want)
			}
		}
	})

	t.Run("ROM Bank Switching", func(t *testing.T) {
		// Create a fake ROM with 4 banks (64KB)
		rom := make([]uint8, 0x10000)
		for i := range rom {
			// Fill each bank with its bank number
			bankNum := uint8(i / 0x4000)
			rom[i] = bankNum
		}

		mbc := NewMBC1(rom, false, 0)

		tests := []struct {
			name     string
			bankNum  uint8
			wantByte uint8
		}{
			{"Default Bank (1)", 1, 1},
			{"Switch to Bank 2", 2, 2},
			{"Switch to Bank 3", 3, 3},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.bankNum > 1 {
					mbc.Write(0x2000, tt.bankNum)
				}
				got := mbc.Read(0x4000)
				if got != tt.wantByte {
					t.Errorf("Bank %d: Read(0x4000) = 0x%02X; want 0x%02X",
						tt.bankNum, got, tt.wantByte)
				}
			})
		}
	})

	t.Run("RAM Banking", func(t *testing.T) {
		mbc := NewMBC1(make([]uint8, 0x8000), false, 4) // 4 RAM banks

		t.Run("RAM Disabled by Default", func(t *testing.T) {
			got := mbc.Read(0xA000)
			if got != 0xFF {
				t.Errorf("Read from disabled RAM = 0x%02X; want 0xFF", got)
			}
		})

		t.Run("RAM Enable/Disable", func(t *testing.T) {
			// Enable RAM
			mbc.Write(0x0000, 0x0A)
			mbc.Write(0xA000, 0x42)
			got := mbc.Read(0xA000)
			if got != 0x42 {
				t.Errorf("Read after RAM enable = 0x%02X; want 0x42", got)
			}

			// Disable RAM
			mbc.Write(0x0000, 0x00)
			got = mbc.Read(0xA000)
			if got != 0xFF {
				t.Errorf("Read after RAM disable = 0x%02X; want 0xFF", got)
			}
		})

		t.Run("Multiple RAM Banks", func(t *testing.T) {
			// Enable RAM
			mbc.Write(0x0000, 0x0A)
			// Switch to RAM banking mode
			mbc.Write(0x6000, 1)

			// Write different values to different banks
			tests := []struct {
				bankNum uint8
				value   uint8
			}{
				{0, 0x42},
				{1, 0x43},
				{2, 0x44},
				{3, 0x45},
			}

			// Write to each bank
			for _, tt := range tests {
				mbc.Write(0x4000, tt.bankNum)
				mbc.Write(0xA000, tt.value)
			}

			// Verify each bank retained its value
			for _, tt := range tests {
				mbc.Write(0x4000, tt.bankNum)
				got := mbc.Read(0xA000)
				if got != tt.value {
					t.Errorf("Bank %d: got 0x%02X; want 0x%02X",
						tt.bankNum, got, tt.value)
				}
			}
		})
	})

	t.Run("Banking Modes", func(t *testing.T) {
		// Create a ROM with 8 banks (128KB)
		rom := make([]uint8, 8*0x4000) // 8 banks * 16KB per bank
		for i := range rom {
			// Fill each bank with its bank number
			bankNum := uint8(i / 0x4000)
			rom[i] = bankNum
		}

		mbc := NewMBC1(rom, false, 4)

		t.Run("ROM Banking Mode (0)", func(t *testing.T) {
			mbc.Write(0x6000, 0) // ROM banking mode
			mbc.Write(0x2000, 5) // Set lower 5 bits of ROM bank to 5
			mbc.Write(0x4000, 0) // Set upper 2 bits of ROM bank to 0

			got := mbc.Read(0x4000)
			want := uint8(5) // Bank 5 (00101b)
			if got != want {
				t.Errorf("Read in ROM mode = 0x%02X; want 0x%02X", got, want)
			}

			// Test bank wrapping (trying to access bank 37 with only 8 banks should wrap to bank 5)
			// 37 % 8 = 5
			mbc.Write(0x2000, 5) // Set lower 5 bits of ROM bank to 5
			mbc.Write(0x4000, 1) // Set upper 2 bits of ROM bank to 1 (would be bank 37)

			got = mbc.Read(0x4000)
			want = uint8(5) // Bank wraps from 37 to 5 (37 % 8 = 5)
			if got != want {
				t.Errorf("Read in ROM mode with bank wrapping = 0x%02X; want 0x%02X", got, want)
			}
		})

		t.Run("RAM Banking Mode (1)", func(t *testing.T) {
			mbc.Write(0x6000, 1) // RAM banking mode
			mbc.Write(0x2000, 5) // Set ROM bank to 5
			mbc.Write(0x4000, 2) // Set RAM bank to 2

			// In RAM mode, the upper bits should not affect ROM bank
			if mbc.romBank != 5 {
				t.Errorf("ROM bank in RAM mode = %d; want 5", mbc.romBank)
			}

			// But should affect RAM bank
			if mbc.ramBank != 2 {
				t.Errorf("RAM bank = %d; want 2", mbc.ramBank)
			}

			// Verify we can still read from the correct ROM bank
			got := mbc.Read(0x4000)
			want := uint8(5) // Should read from bank 5
			if got != want {
				t.Errorf("Read in RAM mode = 0x%02X; want 0x%02X", got, want)
			}
		})
	})

	t.Run("Invalid Bank Handling", func(t *testing.T) {
		mbc := NewMBC1(make([]uint8, 0x8000), false, 0)

		t.Run("Bank 0 Translation", func(t *testing.T) {
			mbc.Write(0x2000, 0)
			if mbc.romBank != 1 {
				t.Errorf("ROM bank 0 not translated to 1, got bank %d", mbc.romBank)
			}
		})

		t.Run("Out of Bounds Access", func(t *testing.T) {
			got := mbc.Read(0xC000) // Outside of ROM/RAM range
			if got != 0xFF {
				t.Errorf("Read from invalid address = 0x%02X; want 0xFF", got)
			}
		})
	})
}
