package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

// ============================================================================
// Sprite Priority Tests
// These tests verify Game Boy hardware sprite priority rules:
// 1. Sprites with lower X coordinate have priority
// 2. If X coordinates are equal, lower OAM index has priority
// 3. Sprite priority flag affects sprite vs background priority
// ============================================================================

// TestSpritePriorityXCoordinate tests sprite-to-sprite priority based on X coordinate and OAM index
func TestSpritePriorityXCoordinate(t *testing.T) {
	tests := []struct {
		name    string
		sprites []struct {
			oamIndex int
			x, y     int
			tileData [16]byte // tile pattern for 8x8 sprite
		}
		expectedPixelOwner []int // which sprite index owns each pixel at y=50
	}{
		{
			name: "Lower X coordinate has priority",
			sprites: []struct {
				oamIndex int
				x, y     int
				tileData [16]byte
			}{
				{
					oamIndex: 0,
					x:        20, // higher X
					y:        50,
					tileData: [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
						0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // all black
				},
				{
					oamIndex: 1,
					x:        10, // lower X - should have priority
					y:        50,
					tileData: [16]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF,
						0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}, // all dark grey
				},
			},
			// sprite 1 (x=10) should win over sprite 0 (x=20) in overlap
			expectedPixelOwner: []int{
				-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 0-9: no sprite
				1, 1, 1, 1, 1, 1, 1, 1, // 10-17: sprite 1
				-1, -1, // 18-19: no sprite
				0, 0, 0, 0, 0, 0, 0, 0, // 20-27: sprite 0
				// ... rest is -1
			},
		},
		{
			name: "Same X coordinate - lower OAM index has priority",
			sprites: []struct {
				oamIndex int
				x, y     int
				tileData [16]byte
			}{
				{
					oamIndex: 0,
					x:        20,
					y:        50,
					tileData: [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
						0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // all black
				},
				{
					oamIndex: 1,
					x:        20, // same X as sprite 0
					y:        50,
					tileData: [16]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF,
						0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}, // all dark grey
				},
			},
			// sprite 0 should win (lower OAM index)
			expectedPixelOwner: []int{
				-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 0-9: no sprite
				-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 10-19: no sprite
				0, 0, 0, 0, 0, 0, 0, 0, // 20-27: sprite 0 wins
				// ... rest is -1
			},
		},
		{
			name: "Complex overlap - X coord then OAM index",
			sprites: []struct {
				oamIndex int
				x, y     int
				tileData [16]byte
			}{
				{
					oamIndex: 0,
					x:        15, // middle X
					y:        50,
					tileData: [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
						0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // all black
				},
				{
					oamIndex: 1,
					x:        10, // lowest X - highest priority
					y:        50,
					tileData: [16]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF,
						0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}, // all dark grey
				},
				{
					oamIndex: 2,
					x:        15, // same as sprite 0, but higher OAM index
					y:        50,
					tileData: [16]byte{0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00,
						0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00}, // all light grey
				},
			},
			expectedPixelOwner: []int{
				-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, // 0-9: no sprite
				1, 1, 1, 1, 1, // 10-14: sprite 1 (lowest X)
				1, 1, 1, // 15-17: sprite 1 still wins (lower X than sprite 0/2)
				0, 0, 0, 0, 0, // 18-22: sprite 0 wins over sprite 2 (same X, lower OAM)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			// enable LCD, sprites, and background
			mmu.Write(addr.LCDC, 0x83) // LCD on, sprites on, BG on
			mmu.Write(addr.BGP, 0xE4)  // background palette (same as sprite)
			mmu.Write(addr.OBP0, 0xE4) // sprite palette

			// set up sprites in OAM
			for _, sprite := range tt.sprites {
				oamAddr := uint16(0xFE00 + sprite.oamIndex*4)
				mmu.Write(oamAddr, byte(sprite.y+16))         // Y position (OAM offset)
				mmu.Write(oamAddr+1, byte(sprite.x+8))        // X position (OAM offset)
				mmu.Write(oamAddr+2, byte(sprite.oamIndex+1)) // tile number
				mmu.Write(oamAddr+3, 0x00)                    // attributes

				// write tile data
				tileAddr := uint16(0x8000 + (sprite.oamIndex+1)*16)
				for i := 0; i < 16; i++ {
					mmu.Write(tileAddr+uint16(i), sprite.tileData[i])
				}
			}

			// draw scanline
			gpu.line = 50
			gpu.drawScanline()

			// check which sprite "owns" each pixel
			fb := gpu.GetFrameBuffer()
			for x := 0; x < len(tt.expectedPixelOwner); x++ {
				pixel := fb.GetPixel(uint(x), 50)
				expectedOwner := tt.expectedPixelOwner[x]

				if expectedOwner == -1 {
					// should show background (white with default palette)
					if pixel != uint32(WhiteColor) {
						t.Logf("Pixel %d: expected white (0x%08X), got 0x%08X", x, uint32(WhiteColor), pixel)
					}
					assert.Equal(t, uint32(WhiteColor), pixel,
						"Pixel %d should be background", x)
				} else {
					// should show the expected sprite's color
					sprite := tt.sprites[expectedOwner]
					// determine what color this sprite shows
					// for our test patterns:
					// 0xFF, 0xFF = black
					// 0x00, 0xFF = dark grey
					// 0xFF, 0x00 = light grey
					var expectedColor uint32
					if sprite.tileData[0] == 0xFF && sprite.tileData[1] == 0xFF {
						expectedColor = uint32(BlackColor)
					} else if sprite.tileData[0] == 0x00 && sprite.tileData[1] == 0xFF {
						expectedColor = uint32(DarkGreyColor)
					} else if sprite.tileData[0] == 0xFF && sprite.tileData[1] == 0x00 {
						expectedColor = uint32(LightGreyColor)
					}

					assert.Equal(t, expectedColor, pixel,
						"Pixel %d should show sprite %d", x, expectedOwner)
				}
			}
		})
	}
}

// ============================================================================
// Sprite Hardware Limit Tests
// These tests verify Game Boy hardware sprite limits:
// 1. Maximum 10 sprites per scanline
// 2. Off-screen sprites count toward the limit
// ============================================================================

// TestTenSpriteLimitPerScanline verifies that only 10 sprites can be rendered per scanline
func TestTenSpriteLimitPerScanline(t *testing.T) {
	mmu := memory.New()
	gpu := NewGpu(mmu)

	// enable LCD and sprites
	mmu.Write(addr.LCDC, 0x93) // LCD on, BG on, sprites on, tile set 1
	mmu.Write(addr.BGP, 0xE4)  // BG palette
	mmu.Write(addr.OBP0, 0xE4) // Sprite palette

	// add 12 sprites on the same scanline (Y=50)
	for i := 0; i < 12; i++ {
		spriteY := 50 + 16     // OAM Y offset
		spriteX := 8 + i*8 + 8 // spread out, OAM X offset

		// set up sprite in OAM
		oamAddr := uint16(0xFE00 + i*4)
		mmu.Write(oamAddr, byte(spriteY))   // Y position
		mmu.Write(oamAddr+1, byte(spriteX)) // X position
		mmu.Write(oamAddr+2, byte(i+1))     // Tile number
		mmu.Write(oamAddr+3, 0)             // No flags

		// write tile data (all black for visibility)
		tileAddr := uint16(0x8000 + (i+1)*16)
		for j := 0; j < 16; j += 2 {
			mmu.Write(tileAddr+uint16(j), 0xFF)
			mmu.Write(tileAddr+uint16(j)+1, 0xFF)
		}
	}

	// draw scanline 50
	gpu.line = 50
	gpu.drawScanline()

	// verify that the first 10 sprites are visible
	fb := gpu.GetFrameBuffer()
	bgColor := fb.GetPixel(0, 50) // get background color

	for i := 0; i < 10; i++ {
		x := uint(8 + i*8)
		pixel := fb.GetPixel(x, 50)
		assert.NotEqual(t, bgColor, pixel,
			"Sprite %d should be visible", i)
	}

	// sprites 10 and 11 should NOT be visible
	for i := 10; i < 12; i++ {
		x := uint(8 + i*8)
		pixel := fb.GetPixel(x, 50)
		assert.Equal(t, bgColor, pixel,
			"Sprite %d should NOT be visible (exceeds 10-sprite limit)", i)
	}
}

// TestOffScreenSpritesCountTowardLimit verifies off-screen sprites count toward 10-sprite limit
func TestOffScreenSpritesCountTowardLimit(t *testing.T) {
	mmu := memory.New()
	gpu := NewGpu(mmu)

	// enable LCD and sprites
	mmu.Write(addr.LCDC, 0x82) // LCD on, sprites on
	mmu.Write(addr.OBP0, 0xE4) // sprite palette

	// set up 12 sprites: 8 off-screen (X=0), 4 on-screen
	// the first 10 should be selected (including off-screen ones)
	for i := 0; i < 12; i++ {
		oamAddr := uint16(0xFE00 + i*4)
		mmu.Write(oamAddr, 66) // Y = 50 + 16

		if i < 8 {
			mmu.Write(oamAddr+1, 0) // X = 0 (off-screen, X-8 = -8)
		} else {
			mmu.Write(oamAddr+1, byte(20+i*10)) // X = visible
		}

		mmu.Write(oamAddr+2, byte(i+1)) // tile number
		mmu.Write(oamAddr+3, 0x00)      // attributes

		// write tile data (all black for visibility)
		tileAddr := uint16(0x8000 + (i+1)*16)
		for j := 0; j < 16; j += 2 {
			mmu.Write(tileAddr+uint16(j), 0xFF)
			mmu.Write(tileAddr+uint16(j)+1, 0xFF)
		}
	}

	// draw scanline
	gpu.line = 50
	gpu.drawScanline()

	// only sprites 8 and 9 should be visible (indices 8-9 are within first 10)
	// sprites 10 and 11 should NOT be drawn (exceed 10-sprite limit)
	fb := gpu.GetFrameBuffer()

	// check that sprite 8 is visible at X=92 (OAM X=100, screen X=100-8=92)
	pixel8 := fb.GetPixel(uint(92), 50)
	assert.Equal(t, uint32(BlackColor), pixel8, "Sprite 8 should be visible")

	// check that sprite 9 is visible at X=102 (OAM X=110, screen X=110-8=102)
	pixel9 := fb.GetPixel(uint(102), 50)
	assert.Equal(t, uint32(BlackColor), pixel9, "Sprite 9 should be visible")

	// check that sprite 10 is NOT visible at X=112 (OAM X=120, exceeds 10-sprite limit)
	pixel10 := fb.GetPixel(uint(112), 50)
	assert.Equal(t, uint32(WhiteColor), pixel10, "Sprite 10 should NOT be visible (exceeds limit)")

	// check that sprite 11 is NOT visible at X=122 (OAM X=130, exceeds 10-sprite limit)
	pixel11 := fb.GetPixel(uint(122), 50)
	assert.Equal(t, uint32(WhiteColor), pixel11, "Sprite 11 should NOT be visible (exceeds limit)")
}

// ============================================================================
// Sprite vs Background Priority Tests
// ============================================================================

// TestSpritePriorityOverBackground tests sprite priority flag behavior with background
func TestSpritePriorityOverBackground(t *testing.T) {
	tests := []struct {
		name           string
		bgPixel        byte
		spritePriority bool
		spritePixel    byte
		expectedDrawn  bool
	}{
		// sprite with priority=0 (above BG)
		{"Sprite above BG color 0", 0, false, 1, true},
		{"Sprite above BG color 1", 1, false, 1, true},
		{"Sprite above BG color 2", 2, false, 1, true},
		{"Sprite above BG color 3", 3, false, 1, true},

		// sprite with priority=1 (behind BG)
		{"Sprite behind BG, over color 0", 0, true, 1, true},  // visible over BG color 0
		{"Sprite behind BG, under color 1", 1, true, 1, false}, // hidden by BG colors 1-3
		{"Sprite behind BG, under color 2", 2, true, 1, false},
		{"Sprite behind BG, under color 3", 3, true, 1, false},

		// transparent sprite (color 0)
		{"Transparent sprite", 0, false, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmu := memory.New()
			gpu := NewGpu(mmu)

			// enable LCD and sprites
			mmu.Write(addr.LCDC, 0x93) // LCD on, BG on, sprites on, tile set 1
			mmu.Write(addr.BGP, 0xE4)  // BG palette
			mmu.Write(addr.OBP0, 0xE4) // Sprite palette

			// set up background tile at position (50, 50)
			tileX := 50 / 8
			tileY := 50 / 8
			tileMapAddr := uint16(0x9800) + uint16(tileY)*32 + uint16(tileX)
			mmu.Write(tileMapAddr, 0) // use tile 0

			// write background tile data
			for row := 0; row < 8; row++ {
				byte1 := byte(0)
				byte2 := byte(0)
				for bit := 0; bit < 8; bit++ {
					if tt.bgPixel&1 != 0 {
						byte1 |= (1 << bit)
					}
					if tt.bgPixel&2 != 0 {
						byte2 |= (1 << bit)
					}
				}
				mmu.Write(0x8000+uint16(row*2), byte1)
				mmu.Write(0x8000+uint16(row*2)+1, byte2)
			}

			// set up sprite at position (50, 50)
			spriteY := 50 + 16 // OAM Y offset
			spriteX := 50 + 8  // OAM X offset

			oamAddr := uint16(0xFE00)
			mmu.Write(oamAddr, byte(spriteY))   // Y position
			mmu.Write(oamAddr+1, byte(spriteX)) // X position
			mmu.Write(oamAddr+2, 1)             // Tile number 1
			attrs := byte(0)
			if tt.spritePriority {
				attrs |= 0x80 // Set priority bit
			}
			mmu.Write(oamAddr+3, attrs) // Attributes

			// write sprite tile data
			for i := 0; i < 16; i += 2 {
				if tt.spritePixel&1 != 0 {
					mmu.Write(0x8010+uint16(i), 0xFF)
				} else {
					mmu.Write(0x8010+uint16(i), 0x00)
				}
				if tt.spritePixel&2 != 0 {
					mmu.Write(0x8010+uint16(i+1), 0xFF)
				} else {
					mmu.Write(0x8010+uint16(i+1), 0x00)
				}
			}

			// draw the scanline
			gpu.line = 50
			gpu.drawScanline()

			// check the result
			fb := gpu.GetFrameBuffer()
			pixel := fb.GetPixel(50, 50)

			// determine expected color
			palette := []GBColor{WhiteColor, LightGreyColor, DarkGreyColor, BlackColor}
			bgExpectedColor := palette[tt.bgPixel]
			spriteExpectedColor := palette[tt.spritePixel]

			if tt.expectedDrawn {
				assert.Equal(t, uint32(spriteExpectedColor), pixel,
					"Sprite should be drawn")
			} else {
				assert.Equal(t, uint32(bgExpectedColor), pixel,
					"Background should show through")
			}
		})
	}
}
