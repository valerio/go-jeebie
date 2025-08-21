package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

func TestExtractVRAMData(t *testing.T) {
	mmu := memory.New()

	// Set up test tile pattern data at tile 0
	// Create a simple 2x2 checkerboard pattern in the first tile
	tileAddr := uint16(VRAMBaseAddr)
	mmu.Write(tileAddr, 0xF0)   // Row 0 low bits:  11110000
	mmu.Write(tileAddr+1, 0x0F) // Row 0 high bits: 00001111 -> colors: 1,1,1,1,2,2,2,2
	mmu.Write(tileAddr+2, 0x0F) // Row 1 low bits:  00001111
	mmu.Write(tileAddr+3, 0xF0) // Row 1 high bits: 11110000 -> colors: 2,2,2,2,1,1,1,1
	// Fill remaining rows with zeros for simplicity
	for i := 4; i < TileDataSize; i++ {
		mmu.Write(tileAddr+uint16(i), 0x00)
	}

	// Set LCDC register
	mmu.Write(0xFF40, 0x91) // LCD on, BG on, sprites on

	vramData := ExtractVRAMData(mmu)

	assert.NotNil(t, vramData)
	assert.Equal(t, TilePatternCount, len(vramData.TilePatterns))

	// Test tile 0 pattern
	tile0 := vramData.TilePatterns[0]
	assert.Equal(t, 0, tile0.Index)

	// Check first two rows of the checkerboard pattern
	expectedRow0 := []video.GBColor{
		video.GBColor(1), video.GBColor(1), video.GBColor(1), video.GBColor(1),
		video.GBColor(2), video.GBColor(2), video.GBColor(2), video.GBColor(2),
	}
	expectedRow1 := []video.GBColor{
		video.GBColor(2), video.GBColor(2), video.GBColor(2), video.GBColor(2),
		video.GBColor(1), video.GBColor(1), video.GBColor(1), video.GBColor(1),
	}

	pixels0 := tile0.Pixels()
	for x := 0; x < TilePixelWidth; x++ {
		assert.Equal(t, expectedRow0[x], pixels0[0][x], "Row 0, pixel %d", x)
		assert.Equal(t, expectedRow1[x], pixels0[1][x], "Row 1, pixel %d", x)
	}

	// Check remaining rows are zeros (color 0)
	for y := 2; y < TilePixelHeight; y++ {
		for x := 0; x < TilePixelWidth; x++ {
			assert.Equal(t, video.GBColor(0), pixels0[y][x], "Row %d, pixel %d should be 0", y, x)
		}
	}

	// Test tilemap info
	assert.True(t, vramData.TilemapInfo.BackgroundActive)
	assert.False(t, vramData.TilemapInfo.WindowActive) // Window not enabled in LCDC
	assert.Equal(t, uint8(0x91), vramData.TilemapInfo.LCDCValue)
}

func TestExtractTilePattern(t *testing.T) {
	mmu := memory.New()

	tests := []struct {
		name      string
		tileIndex int
		lowByte   uint8
		highByte  uint8
		expected  []video.GBColor
	}{
		{
			name:      "All zeros",
			tileIndex: 0,
			lowByte:   0x00,                                    // 00000000
			highByte:  0x00,                                    // 00000000
			expected:  []video.GBColor{0, 0, 0, 0, 0, 0, 0, 0}, // All color 0
		},
		{
			name:      "All low bits",
			tileIndex: 1,
			lowByte:   0xFF,                                    // 11111111
			highByte:  0x00,                                    // 00000000
			expected:  []video.GBColor{1, 1, 1, 1, 1, 1, 1, 1}, // All color 1
		},
		{
			name:      "All high bits",
			tileIndex: 2,
			lowByte:   0x00,                                    // 00000000
			highByte:  0xFF,                                    // 11111111
			expected:  []video.GBColor{2, 2, 2, 2, 2, 2, 2, 2}, // All color 2
		},
		{
			name:      "Both bits set",
			tileIndex: 3,
			lowByte:   0xFF,                                    // 11111111
			highByte:  0xFF,                                    // 11111111
			expected:  []video.GBColor{3, 3, 3, 3, 3, 3, 3, 3}, // All color 3
		},
		{
			name:      "Alternating pattern",
			tileIndex: 4,
			lowByte:   0xAA,                                    // 10101010
			highByte:  0x55,                                    // 01010101
			expected:  []video.GBColor{1, 2, 1, 2, 1, 2, 1, 2}, // Alternating 1,2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up tile data (only first row for simplicity)
			tileAddr := uint16(VRAMBaseAddr + tt.tileIndex*TileDataSize)
			mmu.Write(tileAddr, tt.lowByte)
			mmu.Write(tileAddr+1, tt.highByte)

			baseAddr := uint16(VRAMBaseAddr + tt.tileIndex*TileDataSize)
			tile := video.FetchTileWithIndex(mmu, baseAddr, tt.tileIndex)
			pixels := tile.Pixels()

			assert.Equal(t, tt.tileIndex, tile.Index)

			// Check first row pixels
			for x := 0; x < TilePixelWidth; x++ {
				assert.Equal(t, tt.expected[x], pixels[0][x],
					"Pixel %d should be color %d", x, tt.expected[x])
			}
		})
	}
}

func TestExtractTilemapInfo(t *testing.T) {
	mmu := memory.New()

	tests := []struct {
		name              string
		lcdcValue         uint8
		expectedBG        bool
		expectedWindow    bool
		expectedLCDCValue uint8
	}{
		{
			name:              "LCD off, all disabled",
			lcdcValue:         0x00,
			expectedBG:        false,
			expectedWindow:    false,
			expectedLCDCValue: 0x00,
		},
		{
			name:              "LCD on, BG enabled only",
			lcdcValue:         0x81, // Bit 7 (LCD) + Bit 0 (BG)
			expectedBG:        true,
			expectedWindow:    false,
			expectedLCDCValue: 0x81,
		},
		{
			name:              "LCD on, Window enabled only",
			lcdcValue:         0xA0, // Bit 7 (LCD) + Bit 5 (Window)
			expectedBG:        false,
			expectedWindow:    true,
			expectedLCDCValue: 0xA0,
		},
		{
			name:              "LCD on, BG and Window enabled",
			lcdcValue:         0xA1, // Bit 7 (LCD) + Bit 5 (Window) + Bit 0 (BG)
			expectedBG:        true,
			expectedWindow:    true,
			expectedLCDCValue: 0xA1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu.Write(0xFF40, tt.lcdcValue)

			tilemapInfo := extractTilemapInfoFromReader(mmu)

			assert.Equal(t, tt.expectedBG, tilemapInfo.BackgroundActive)
			assert.Equal(t, tt.expectedWindow, tilemapInfo.WindowActive)
			assert.Equal(t, tt.expectedLCDCValue, tilemapInfo.LCDCValue)
		})
	}
}

func TestGetTileGrid(t *testing.T) {
	mmu := memory.New()
	vramData := ExtractVRAMData(mmu)

	grid := vramData.GetTileGrid()

	assert.Equal(t, TileRows, len(grid))
	for row := 0; row < TileRows; row++ {
		assert.Equal(t, TilesPerRow, len(grid[row]))
		for col := 0; col < TilesPerRow; col++ {
			expectedIndex := row*TilesPerRow + col
			if expectedIndex < TilePatternCount {
				assert.Equal(t, expectedIndex, grid[row][col].Index)
			}
		}
	}
}

func TestFormatTilemapSummary(t *testing.T) {
	tests := []struct {
		name     string
		info     TilemapInfo
		expected string
	}{
		{
			name: "Both inactive",
			info: TilemapInfo{
				BackgroundActive: false,
				WindowActive:     false,
				LCDCValue:        0x80,
			},
			expected: "Background Map: 0x9800 [INACTIVE] | Window Map: 0x9C00 [INACTIVE] | LCDC: 0x80",
		},
		{
			name: "Background active only",
			info: TilemapInfo{
				BackgroundActive: true,
				WindowActive:     false,
				LCDCValue:        0x81,
			},
			expected: "Background Map: 0x9800 [ACTIVE] | Window Map: 0x9C00 [INACTIVE] | LCDC: 0x81",
		},
		{
			name: "Both active",
			info: TilemapInfo{
				BackgroundActive: true,
				WindowActive:     true,
				LCDCValue:        0xA1,
			},
			expected: "Background Map: 0x9800 [ACTIVE] | Window Map: 0x9C00 [ACTIVE] | LCDC: 0xA1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.info.FormatSummary()
			assert.Equal(t, tt.expected, summary)
		})
	}
}

func TestTilePatternExtraction(t *testing.T) {
	mmu := memory.New()

	// Test a simple cross pattern in tile 5
	tileIndex := 5
	tileAddr := uint16(VRAMBaseAddr + tileIndex*TileDataSize)

	// Create a cross pattern:
	// Row 0: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0
	// Row 1: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0
	// Row 2: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0
	// Row 3: 11111111 (low) + 00000000 (high) = 11111111 -> colors 1,1,1,1,1,1,1,1
	// Row 4: 11111111 (low) + 00000000 (high) = 11111111 -> colors 1,1,1,1,1,1,1,1
	// Row 5: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0
	// Row 6: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0
	// Row 7: 00011000 (low) + 00000000 (high) = 00011000 -> colors 0,0,0,1,1,0,0,0

	crossPattern := []uint8{
		0x18, 0x00, // Row 0: 00011000, 00000000
		0x18, 0x00, // Row 1: 00011000, 00000000
		0x18, 0x00, // Row 2: 00011000, 00000000
		0xFF, 0x00, // Row 3: 11111111, 00000000
		0xFF, 0x00, // Row 4: 11111111, 00000000
		0x18, 0x00, // Row 5: 00011000, 00000000
		0x18, 0x00, // Row 6: 00011000, 00000000
		0x18, 0x00, // Row 7: 00011000, 00000000
	}

	for i, data := range crossPattern {
		mmu.Write(tileAddr+uint16(i), data)
	}

	tile := video.FetchTileWithIndex(mmu, tileAddr, tileIndex)
	pixels := tile.Pixels()

	// Expected cross pattern
	expectedRows := [][]video.GBColor{
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 0
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 1
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 2
		{1, 1, 1, 1, 1, 1, 1, 1}, // Row 3
		{1, 1, 1, 1, 1, 1, 1, 1}, // Row 4
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 5
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 6
		{0, 0, 0, 1, 1, 0, 0, 0}, // Row 7
	}

	for y := 0; y < TilePixelHeight; y++ {
		for x := 0; x < TilePixelWidth; x++ {
			expected := video.GBColor(expectedRows[y][x])
			actual := pixels[y][x]
			assert.Equal(t, expected, actual,
				"Cross pattern mismatch at row %d, col %d", y, x)
		}
	}
}
