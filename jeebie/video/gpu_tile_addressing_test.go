package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

const defaultPalette = 0xE4

func TestGPUSignedTileAddressing(t *testing.T) {
	tests := []struct {
		name             string
		tileNumber       byte
		expectedTileAddr uint16
	}{
		{"Tile -128 (0x80)", 0x80, 0x8800},
		{"Tile -127 (0x81)", 0x81, 0x8810},
		{"Tile -1 (0xFF)", 0xFF, 0x8FF0},
		{"Tile 0 (0x00)", 0x00, 0x9000},
		{"Tile 1 (0x01)", 0x01, 0x9010},
		{"Tile 127 (0x7F)", 0x7F, 0x97F0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			mmu.Write(addr.LCDC, 0x81) // LCD on, BG on, signed tiles
			mmu.Write(addr.BGP, defaultPalette)
			mmu.Write(addr.TileMap0, tt.tileNumber)

			mmu.Write(tt.expectedTileAddr, 0xAA)
			mmu.Write(tt.expectedTileAddr+1, 0xBB)

			gpu.line = 0
			gpu.drawScanline()

			fb := gpu.GetFrameBuffer()
			expectedColors := []uint32{
				uint32(BlackColor),
				uint32(WhiteColor),
				uint32(BlackColor),
				uint32(DarkGreyColor),
				uint32(BlackColor),
				uint32(WhiteColor),
				uint32(BlackColor),
				uint32(DarkGreyColor),
			}

			for i := 0; i < 8; i++ {
				pixel := fb.GetPixel(uint(i), 0)
				assert.Equal(t, expectedColors[i], pixel,
					"Pixel %d for tile %02X at address %04X", i, tt.tileNumber, tt.expectedTileAddr)
			}
		})
	}
}

func TestGPUUnsignedTileAddressing(t *testing.T) {
	tests := []struct {
		name             string
		tileNumber       byte
		expectedTileAddr uint16
	}{
		{"Tile 0 (0x00)", 0x00, 0x8000},
		{"Tile 1 (0x01)", 0x01, 0x8010},
		{"Tile 127 (0x7F)", 0x7F, 0x87F0},
		{"Tile 128 (0x80)", 0x80, 0x8800},
		{"Tile 255 (0xFF)", 0xFF, 0x8FF0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			mmu.Write(addr.LCDC, 0x91) // LCD on, BG on, unsigned tiles
			mmu.Write(addr.BGP, defaultPalette)
			mmu.Write(addr.TileMap0, tt.tileNumber)

			mmu.Write(tt.expectedTileAddr, 0xFF)
			mmu.Write(tt.expectedTileAddr+1, 0x00)

			gpu.line = 0
			gpu.drawScanline()

			fb := gpu.GetFrameBuffer()
			for i := 0; i < 8; i++ {
				pixel := fb.GetPixel(uint(i), 0)
				assert.Equal(t, uint32(LightGreyColor), pixel,
					"Pixel %d for tile %02X at address %04X", i, tt.tileNumber, tt.expectedTileAddr)
			}
		})
	}
}
