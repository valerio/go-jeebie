package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestGPUSignedTileDataFetch(t *testing.T) {
	tests := []struct {
		name         string
		tileNumber   byte
		pixelRow     int
		expectedAddr uint16
	}{
		{"Tile 0 (0x00)", 0x00, 0, 0x9000},
		{"Tile 1 (0x01)", 0x01, 0, 0x9010},
		{"Tile 127 (0x7F)", 0x7F, 0, 0x97F0},

		{"Tile -128 (0x80)", 0x80, 0, 0x8800},
		{"Tile -127 (0x81)", 0x81, 0, 0x8810},
		{"Tile -1 (0xFF)", 0xFF, 0, 0x8FF0},

		{"Tile -64 (0xC0), row 3", 0xC0, 3, 0x8C06},
		{"Tile 64 (0x40), row 4", 0x40, 4, 0x9408},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)
			mmu.Write(addr.LCDC, 0x81) // LCD on, BG on, signed tiles
			mmu.Write(addr.BGP, defaultPalette)
			mmu.Write(addr.TileMap0, tt.tileNumber)

			testPattern := []byte{
				0xAA, 0x55, 0x33, 0xCC, 0x0F, 0xF0, 0x81, 0x7E,
				0xFF, 0x00, 0x00, 0xFF, 0x55, 0xAA, 0xCC, 0x33,
			}

			if tt.pixelRow == 0 {
				for i := range 16 {
					mmu.Write(tt.expectedAddr+uint16(i), testPattern[i])
				}
			} else {
				mmu.Write(tt.expectedAddr, testPattern[tt.pixelRow*2])
				mmu.Write(tt.expectedAddr+1, testPattern[tt.pixelRow*2+1])
			}

			mmu.Write(addr.SCX, 0)
			mmu.Write(addr.SCY, 0)

			gpu.line = tt.pixelRow
			gpu.mode = vramReadMode
			gpu.drawScanline()

			expectedLow := testPattern[tt.pixelRow*2]
			expectedHigh := testPattern[tt.pixelRow*2+1]

			pixel0 := byte(0)
			if (expectedLow>>7)&1 == 1 {
				pixel0 |= 1
			}
			if (expectedHigh>>7)&1 == 1 {
				pixel0 |= 2
			}

			pixel1 := byte(0)
			if (expectedLow>>6)&1 == 1 {
				pixel1 |= 1
			}
			if (expectedHigh>>6)&1 == 1 {
				pixel1 |= 2
			}

			palette := defaultPalette
			color0 := byte((palette >> (pixel0 * 2)) & 0x03)
			color1 := byte((palette >> (pixel1 * 2)) & 0x03)

			actualColor0 := gpu.framebuffer.GetPixel(0, uint(tt.pixelRow))
			actualColor1 := gpu.framebuffer.GetPixel(1, uint(tt.pixelRow))

			expectedColor0 := uint32(ByteToColor(color0))
			expectedColor1 := uint32(ByteToColor(color1))

			assert.Equal(t, expectedColor0, actualColor0,
				"Tile %02X (signed %d) row %d: wrong color at pixel 0",
				tt.tileNumber, int8(tt.tileNumber), tt.pixelRow)
			assert.Equal(t, expectedColor1, actualColor1,
				"Tile %02X (signed %d) row %d: wrong color at pixel 1",
				tt.tileNumber, int8(tt.tileNumber), tt.pixelRow)
		})
	}
}

func TestGPUUnsignedTileDataFetch(t *testing.T) {
	tests := []struct {
		name         string
		tileNumber   byte
		pixelRow     int
		expectedAddr uint16
	}{
		{"Tile 0, row 0", 0, 0, 0x8000},
		{"Tile 1, row 0", 1, 0, 0x8010},
		{"Tile 127, row 0", 127, 0, 0x87F0},
		{"Tile 128, row 0", 128, 0, 0x8800},
		{"Tile 255, row 0", 255, 0, 0x8FF0},
		{"Tile 255, row 7", 255, 7, 0x8FFE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			mmu.Write(addr.LCDC, 0x91|0x10) // LCD on, BG on, unsigned tiles
			mmu.Write(addr.BGP, defaultPalette)
			mmu.Write(addr.TileMap0, tt.tileNumber)

			testPattern := []byte{0x81, 0x42}
			mmu.Write(tt.expectedAddr, testPattern[0])
			mmu.Write(tt.expectedAddr+1, testPattern[1])

			mmu.Write(addr.SCX, 0)
			mmu.Write(addr.SCY, 0)

			gpu.line = tt.pixelRow
			gpu.mode = vramReadMode
			gpu.drawScanline()

			expectedPixel := byte(0)
			if (testPattern[0]>>7)&1 == 1 {
				expectedPixel |= 1
			}
			if (testPattern[1]>>7)&1 == 1 {
				expectedPixel |= 2
			}

			palette := defaultPalette
			expectedColor := byte((palette >> (expectedPixel * 2)) & 0x03)
			actualColor := gpu.framebuffer.GetPixel(0, uint(tt.pixelRow))

			assert.Equal(t, uint32(ByteToColor(expectedColor)), actualColor,
				"Tile %d row %d: wrong color", tt.tileNumber, tt.pixelRow)
		})
	}
}

func TestGPUTileMapAddressing(t *testing.T) {
	tests := []struct {
		name         string
		tileMapBase  uint16
		tileX        int
		tileY        int
		expectedAddr uint16
	}{
		{"Map 0, tile (0,0)", 0x9800, 0, 0, 0x9800},
		{"Map 0, tile (1,0)", 0x9800, 1, 0, 0x9801},
		{"Map 0, tile (31,0)", 0x9800, 31, 0, 0x981F},
		{"Map 0, tile (0,1)", 0x9800, 0, 1, 0x9820},
		{"Map 0, tile (31,31)", 0x9800, 31, 31, 0x9BFF},

		{"Map 1, tile (0,0)", 0x9C00, 0, 0, 0x9C00},
		{"Map 1, tile (31,31)", 0x9C00, 31, 31, 0x9FFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			lcdcFlags := byte(0x91) // LCD on, BG on, unsigned tiles
			if tt.tileMapBase == addr.TileMap1 {
				lcdcFlags |= 0x08
			}
			mmu.Write(addr.LCDC, lcdcFlags|0x10)
			mmu.Write(addr.BGP, defaultPalette)

			// write a unique tile index at the calculated position
			uniqueTileIndex := byte(tt.tileX + tt.tileY*32)
			tileMapAddr := tt.expectedAddr
			mmu.Write(tileMapAddr, uniqueTileIndex)

			tileDataAddr := addr.TileData0 + uint16(uniqueTileIndex)*16
			for row := 0; row < 8; row++ {
				mmu.Write(tileDataAddr+uint16(row*2), uniqueTileIndex)
				mmu.Write(tileDataAddr+uint16(row*2)+1, ^uniqueTileIndex)
			}

			scrollX := byte((tt.tileX * 8) & 0xFF)
			scrollY := byte((tt.tileY * 8) & 0xFF)
			mmu.Write(addr.SCX, scrollX)
			mmu.Write(addr.SCY, scrollY)

			gpu.line = 0
			gpu.mode = vramReadMode
			gpu.drawScanline()

			expectedPixel := byte(0)
			if (uniqueTileIndex>>7)&1 == 1 {
				expectedPixel |= 1
			}
			if (^uniqueTileIndex>>7)&1 == 1 {
				expectedPixel |= 2
			}

			palette := defaultPalette
			expectedColor := byte((palette >> (expectedPixel * 2)) & 0x03)
			actualColor := gpu.framebuffer.GetPixel(0, 0)

			assert.Equal(t, uint32(ByteToColor(expectedColor)), actualColor,
				"Tile (%d,%d) in map %04X not drawn correctly",
				tt.tileX, tt.tileY, tt.tileMapBase)
		})
	}
}

// TestGPUScrollWrapping tests that GPU correctly handles scroll wrapping
func TestGPUScrollWrapping(t *testing.T) {
	tests := []struct {
		name          string
		scrollX       byte
		scrollY       byte
		screenX       int
		screenY       int
		expectedTileX int // which tile should be visible
		expectedTileY int
	}{
		{"No scroll, top-left", 0, 0, 0, 0, 0, 0},
		{"No scroll, tile (1,1)", 0, 0, 8, 8, 1, 1},

		{"Scroll X=8", 8, 0, 0, 0, 1, 0},
		{"Scroll Y=8", 0, 8, 0, 0, 0, 1},

		{"Wrap X", 200, 0, 159, 0, 12, 0},
		{"Wrap Y", 0, 200, 0, 143, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			mmu.Write(addr.LCDC, 0x91|0x10) // LCD on, BG on, unsigned tiles
			mmu.Write(addr.BGP, defaultPalette)

			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					tileIndex := byte((y*32 + x) & 0xFF)
					mmu.Write(addr.TileMap0+uint16(y*32+x), tileIndex)

					tileAddrCalc := addr.TileData0 + uint16(tileIndex)*16
					for row := 0; row < 8; row++ {
						mmu.Write(tileAddrCalc+uint16(row*2), tileIndex)
						mmu.Write(tileAddrCalc+uint16(row*2)+1, byte(x+y))
					}
				}
			}

			mmu.Write(addr.SCX, tt.scrollX)
			mmu.Write(addr.SCY, tt.scrollY)

			gpu.line = tt.screenY
			gpu.mode = vramReadMode
			gpu.drawScanline()

			expectedTileIndex := byte((tt.expectedTileY*32 + tt.expectedTileX) & 0xFF)
			expectedPixel := byte(0)
			if (expectedTileIndex>>7)&1 == 1 {
				expectedPixel |= 1
			}
			if (byte(tt.expectedTileX+tt.expectedTileY)>>7)&1 == 1 {
				expectedPixel |= 2
			}

			palette := defaultPalette
			expectedColor := byte((palette >> (expectedPixel * 2)) & 0x03)
			actualColor := gpu.framebuffer.GetPixel(uint(tt.screenX), uint(tt.screenY))

			assert.Equal(t, uint32(ByteToColor(expectedColor)), actualColor,
				"Wrong tile at screen (%d,%d) with scroll (%d,%d)",
				tt.screenX, tt.screenY, tt.scrollX, tt.scrollY)
		})
	}
}

func TestGPUTilePixelExtraction(t *testing.T) {
	tests := []struct {
		name           string
		lowByte        byte
		highByte       byte
		expectedColors []byte
	}{
		{
			name:           "All white",
			lowByte:        0x00,
			highByte:       0x00,
			expectedColors: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:           "All black",
			lowByte:        0xFF,
			highByte:       0xFF,
			expectedColors: []byte{3, 3, 3, 3, 3, 3, 3, 3},
		},
		{
			name:           "Alternating pattern",
			lowByte:        0xAA,
			highByte:       0x00,
			expectedColors: []byte{1, 0, 1, 0, 1, 0, 1, 0},
		},
		{
			name:           "Mixed colors",
			lowByte:        0x0F,
			highByte:       0xF0,
			expectedColors: []byte{2, 2, 2, 2, 1, 1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			// enable LCD and background
			mmu.Write(addr.LCDC, 0x91|0x10) // unsigned tiles

			mmu.Write(addr.BGP, defaultPalette)
			mmu.Write(addr.TileMap0, 0x00)

			mmu.Write(addr.TileData0, tt.lowByte)
			mmu.Write(addr.TileData0+1, tt.highByte)

			for i := uint16(2); i < 16; i++ {
				mmu.Write(addr.TileData0+i, 0x00)
			}

			mmu.Write(addr.SCX, 0)
			mmu.Write(addr.SCY, 0)

			gpu.line = 0
			gpu.mode = vramReadMode
			gpu.drawScanline()

			palette := defaultPalette
			for pixelX := 0; pixelX < 8; pixelX++ {
				expectedColorIndex := tt.expectedColors[pixelX]
				expectedColor := byte((palette >> (expectedColorIndex * 2)) & 0x03)
				actualColor := gpu.framebuffer.GetPixel(uint(pixelX), 0)

				assert.Equal(t, uint32(ByteToColor(expectedColor)), actualColor,
					"Pixel %d: expected color %d, got different color",
					pixelX, expectedColorIndex)
			}
		})
	}
}
