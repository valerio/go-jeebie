package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func createColorTile(colorValue int) [16]byte {
	var tile [16]byte
	for row := 0; row < 8; row++ {
		lowByte := byte(0)
		highByte := byte(0)

		for bit := 0; bit < 8; bit++ {
			if colorValue&1 != 0 {
				lowByte |= (1 << bit)
			}
			if colorValue&2 != 0 {
				highByte |= (1 << bit)
			}
		}

		tile[row*2] = lowByte
		tile[row*2+1] = highByte
	}
	return tile
}

func TestGPUPaletteApplication(t *testing.T) {
	tests := []struct {
		name          string
		bgp           byte
		colorValue    byte // tile color value (0-3)
		expectedColor GBColor
	}{
		{"Default palette, color 0", 0xE4, 0, WhiteColor},
		{"Default palette, color 1", 0xE4, 1, LightGreyColor},
		{"Default palette, color 2", 0xE4, 2, DarkGreyColor},
		{"Default palette, color 3", 0xE4, 3, BlackColor},

		{"Inverted palette, color 0", 0x1B, 0, BlackColor},
		{"Inverted palette, color 1", 0x1B, 1, DarkGreyColor},
		{"Inverted palette, color 2", 0x1B, 2, LightGreyColor},
		{"Inverted palette, color 3", 0x1B, 3, WhiteColor},

		{"All black, color 0", 0xFF, 0, BlackColor},
		{"All black, color 1", 0xFF, 1, BlackColor},
		{"All black, color 2", 0xFF, 2, BlackColor},
		{"All black, color 3", 0xFF, 3, BlackColor},

		{"All white, color 0", 0x00, 0, WhiteColor},
		{"All white, color 1", 0x00, 1, WhiteColor},
		{"All white, color 2", 0x00, 2, WhiteColor},
		{"All white, color 3", 0x00, 3, WhiteColor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			// enable LCD and background
			mmu.Write(addr.LCDC, 0x91|0x10)

			// set the test palette
			mmu.Write(addr.BGP, tt.bgp)

			// create and write a tile with the test color
			tileData := createColorTile(int(tt.colorValue))
			for i := 0; i < 16; i++ {
				mmu.Write(0x8000+uint16(i), tileData[i])
			}

			// place tile at (0,0)
			mmu.Write(0x9800, 0x00)

			// no scrolling
			mmu.Write(addr.SCX, 0)
			mmu.Write(addr.SCY, 0)

			// draw first scanline
			gpu.line = 0
			gpu.mode = vramReadMode
			gpu.drawScanline()

			// check first pixel
			actualColor := gpu.framebuffer.GetPixel(0, 0)
			expectedColor := uint32(tt.expectedColor)

			assert.Equal(t, expectedColor, actualColor,
				"Palette %02X, color %d: expected %08X",
				tt.bgp, tt.colorValue, expectedColor)
		})
	}
}

func TestGPUWindowPalette(t *testing.T) {
	mmu := memory.New()
	gpu := NewGpu(mmu)

	// enable LCD, background, and window
	// bit 7: LCD on
	// bit 6: window tilemap (0=9800, 1=9C00)
	// bit 5: window on
	// bit 4: BG/window tile data (0=signed, 1=unsigned)
	// bit 0: BG on
	mmu.Write(addr.LCDC, 0xF1) // 11110001: LCD on, window map 1 (9C00), window on, unsigned tiles, BG on

	// set inverted palette to make it obvious
	mmu.Write(addr.BGP, 0x1B) // inverted palette

	// create different tiles for BG and window
	bgTile := createColorTile(0)     // color 0
	windowTile := createColorTile(3) // color 3

	// write tiles
	for i := 0; i < 16; i++ {
		mmu.Write(0x8000+uint16(i), bgTile[i])     // tile 0 for BG
		mmu.Write(0x8010+uint16(i), windowTile[i]) // tile 1 for window
	}

	// fill background tilemap with tile 0
	for i := uint16(0); i < 32*32; i++ {
		mmu.Write(0x9800+i, 0x00)
	}

	// fill window tilemap with tile 1
	for i := uint16(0); i < 32*32; i++ {
		mmu.Write(0x9C00+i, 0x01)
	}

	// position window at (40, 40)
	mmu.Write(addr.WX, 47) // WX = 47 (40 + 7)
	mmu.Write(addr.WY, 40) // WY = 40

	// no scrolling
	mmu.Write(addr.SCX, 0)
	mmu.Write(addr.SCY, 0)

	// draw scanline 40 (where window starts)
	gpu.line = 40
	gpu.mode = vramReadMode
	gpu.drawScanline()

	// check background pixel (before window)
	bgPixel := gpu.framebuffer.GetPixel(30, 40)
	// with inverted palette 0x1B, color 0 -> black
	expectedBgColor := uint32(BlackColor)
	assert.Equal(t, expectedBgColor, bgPixel, "Background should use inverted palette")

	// check window pixel
	windowPixel := gpu.framebuffer.GetPixel(50, 40)
	// with inverted palette 0x1B, color 3 -> white
	expectedWindowColor := uint32(WhiteColor)
	assert.Equal(t, expectedWindowColor, windowPixel, "Window should use same inverted palette as BG")
}

func TestGPUPaletteChange(t *testing.T) {
	mmu := memory.New()
	gpu := NewGpu(mmu)

	// enable LCD and background
	mmu.Write(addr.LCDC, 0x91|0x10)

	// create a tile with color 2
	tileData := createColorTile(2)
	for i := 0; i < 16; i++ {
		mmu.Write(0x8000+uint16(i), tileData[i])
	}

	// fill entire tilemap with tile 0
	for i := uint16(0); i < 32*32; i++ {
		mmu.Write(0x9800+i, 0x00)
	}

	// no scrolling
	mmu.Write(addr.SCX, 0)
	mmu.Write(addr.SCY, 0)

	// set initial palette (default)
	mmu.Write(addr.BGP, 0xE4)

	// draw first scanline with default palette
	gpu.line = 0
	gpu.mode = vramReadMode
	gpu.drawScanline()

	// verify first scanline uses default palette
	// color 2 with palette 0xE4 -> dark grey
	pixel0 := gpu.framebuffer.GetPixel(0, 0)
	assert.Equal(t, uint32(DarkGreyColor), pixel0, "Line 0 should use default palette")

	// change palette to inverted
	mmu.Write(addr.BGP, 0x1B)

	// draw second scanline with new palette
	gpu.line = 1
	gpu.mode = vramReadMode
	gpu.drawScanline()

	// verify second scanline uses new palette
	// color 2 with palette 0x1B -> light grey
	pixel1 := gpu.framebuffer.GetPixel(0, 1)
	assert.Equal(t, uint32(LightGreyColor), pixel1, "Line 1 should use new palette")

	// verify first scanline still has old colors (not retroactively changed)
	pixel0Again := gpu.framebuffer.GetPixel(0, 0)
	assert.Equal(t, uint32(DarkGreyColor), pixel0Again, "Line 0 should still have old colors")
}
