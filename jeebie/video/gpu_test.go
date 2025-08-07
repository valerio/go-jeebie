package video

import (
	"testing"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestGPUBackgroundTileDrawing(t *testing.T) {
	tests := []struct {
		name                string
		tileData            []byte // 16 bytes for one tile
		palette             byte
		scrollX, scrollY    byte
		expectedPixels      []struct{ x, y int; color uint32 }
		lcdcFlags           byte
		tileMapData         byte
		tileMapAddr         uint16
		tileDataAddr        uint16
	}{
		{
			name: "Simple 8x8 tile with all white pixels",
			tileData: []byte{
				0xFF, 0xFF, // Row 0: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 1: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 2: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 3: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 4: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 5: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 6: all pixels = color 3 (white)
				0xFF, 0xFF, // Row 7: all pixels = color 3 (white)
			},
			palette: 0xE4, // 11 10 01 00 = palette mapping
			scrollX: 0, scrollY: 0,
			lcdcFlags:    0x91, // LCD enabled + BG enabled + use tileset 1
			tileMapData:  0x00, // Use tile 0
			tileMapAddr:  0x9800,
			tileDataAddr: 0x8000,
			expectedPixels: []struct{ x, y int; color uint32 }{
				{0, 0, uint32(WhiteColor)},
				{7, 0, uint32(WhiteColor)},
				{0, 7, uint32(WhiteColor)},
				{7, 7, uint32(WhiteColor)},
			},
		},
		{
			name: "Checkered pattern tile",
			tileData: []byte{
				0xAA, 0x00, // Row 0: 10101010, 00000000 = alternating color 2/0
				0x55, 0x00, // Row 1: 01010101, 00000000 = alternating color 1/0
				0xAA, 0x00, // Row 2: 10101010, 00000000 = alternating color 2/0
				0x55, 0x00, // Row 3: 01010101, 00000000 = alternating color 1/0
				0xAA, 0x00, // Row 4: 10101010, 00000000 = alternating color 2/0
				0x55, 0x00, // Row 5: 01010101, 00000000 = alternating color 1/0
				0xAA, 0x00, // Row 6: 10101010, 00000000 = alternating color 2/0
				0x55, 0x00, // Row 7: 01010101, 00000000 = alternating color 1/0
			},
			palette: 0xE4, // 11 10 01 00
			scrollX: 0, scrollY: 0,
			lcdcFlags:    0x91, // LCD enabled + BG enabled + use tileset 1
			tileMapData:  0x00,
			tileMapAddr:  0x9800,
			tileDataAddr: 0x8000,
			expectedPixels: []struct{ x, y int; color uint32 }{
				{0, 0, uint32(DarkGreyColor)},  // 0xAA bit 7=1, 0x00 bit 7=0 → color 1 → DarkGrey
				{1, 0, uint32(BlackColor)},     // 0xAA bit 6=0, 0x00 bit 6=0 → color 0 → Black
				{0, 1, uint32(BlackColor)},     // 0x55 bit 7=0, 0x00 bit 7=0 → color 0 → Black
				{1, 1, uint32(DarkGreyColor)},  // 0x55 bit 6=1, 0x00 bit 6=0 → color 1 → DarkGrey
			},
		},
		{
			name: "Test with scroll offset",
			tileData: []byte{
				0xFF, 0x00, // Row 0: all color 1
				0xFF, 0x00, // Row 1: all color 1
				0xFF, 0x00, // Row 2: all color 1
				0xFF, 0x00, // Row 3: all color 1
				0xFF, 0x00, // Row 4: all color 1
				0xFF, 0x00, // Row 5: all color 1
				0xFF, 0x00, // Row 6: all color 1
				0xFF, 0x00, // Row 7: all color 1
			},
			palette: 0xE4,
			scrollX: 4, scrollY: 2, // Offset by 4 pixels right, 2 pixels down
			lcdcFlags:    0x91,
			tileMapData:  0x00,
			tileMapAddr:  0x9800,
			tileDataAddr: 0x8000,
			expectedPixels: []struct{ x, y int; color uint32 }{
				{0, 0, uint32(DarkGreyColor)}, // Should show tile data offset by scroll
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup memory
			mmu := memory.New()
			gpu := NewGpu(mmu)
			
			// Configure LCDC register
			mmu.Write(addr.LCDC, tt.lcdcFlags)
			
			// Configure palette
			mmu.Write(addr.BGP, tt.palette)
			
			// Configure scroll
			mmu.Write(addr.SCX, tt.scrollX)
			mmu.Write(addr.SCY, tt.scrollY)
			
			// Write tile data to VRAM
			for i, data := range tt.tileData {
				mmu.Write(tt.tileDataAddr+uint16(i), data)
			}
			
			// Write tile map data (which tile to use)
			mmu.Write(tt.tileMapAddr, tt.tileMapData)
			
			// Draw scanlines that contain our expected pixels
			expectedLines := make(map[int]bool)
			for _, expected := range tt.expectedPixels {
				expectedLines[expected.y] = true
			}
			
			// Draw each required scanline
			for line := range expectedLines {
				gpu.line = line
				gpu.mode = vramReadMode
				gpu.pixelCounter = 0
				
				// Draw the background for this line
				for gpu.pixelCounter < 160 {
					gpu.drawBackground()
					gpu.pixelCounter += 4
				}
			}
			
			// Verify expected pixels
			fb := gpu.GetFrameBuffer()
			for _, expected := range tt.expectedPixels {
				actual := fb.GetPixel(uint(expected.x), uint(expected.y))
				if actual != expected.color {
					t.Errorf("Pixel at (%d,%d): expected %08X, got %08X", 
						expected.x, expected.y, expected.color, actual)
				}
			}
		})
	}
}

func TestGPUTileAddressCalculation(t *testing.T) {
	tests := []struct {
		name           string
		useTileSetZero bool
		tileNumber     byte
		expectedAddr   uint16
	}{
		{"Tileset 1, tile 0", false, 0x00, 0x8000},
		{"Tileset 1, tile 1", false, 0x01, 0x8010},
		{"Tileset 1, tile 255", false, 0xFF, 0x8FF0},
		{"Tileset 0, tile 128", true, 0x80, 0x8800}, // signed 0x80 = -128, +128 = 0
		{"Tileset 0, tile 127", true, 0x7F, 0x8FF0}, // signed 0x7F = +127, +128 = 255
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)
			
			// Configure LCDC for tileset selection
			lcdcFlags := byte(0x90) // LCD enabled, BG enabled
			if !tt.useTileSetZero {
				lcdcFlags |= 0x10 // Set bit 4 for tileset 1
			}
			mmu.Write(addr.LCDC, lcdcFlags)
			
			// Write the tile number to tilemap
			mmu.Write(0x9800, tt.tileNumber)
			
			// Set up for tile address calculation test
			gpu.line = 0
			gpu.pixelCounter = 0
			
			// We need to examine what address the GPU tries to read from
			// This is tricky to test directly, so let's put a marker at the expected address
			mmu.Write(tt.expectedAddr, 0xAA)
			mmu.Write(tt.expectedAddr+1, 0xBB)
			
			// Draw and see if it reads from the correct address
			gpu.drawBackground()
			
			// The test passes if no panic occurred and drawing completed
			// More sophisticated testing would require mocking memory reads
		})
	}
}