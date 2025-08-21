package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

func TestExtractOAMData(t *testing.T) {
	mmu := memory.New()

	// Set up test OAM data for sprite 0
	mmu.Write(OAMBaseAddr, 16+50)  // Y position (raw Y = 66, adjusted = 50)
	mmu.Write(OAMBaseAddr+1, 8+30) // X position (raw X = 38, adjusted = 30)
	mmu.Write(OAMBaseAddr+2, 0x42) // Tile index
	mmu.Write(OAMBaseAddr+3, 0x80) // Attributes (background priority set)

	// Set up test OAM data for sprite 1
	mmu.Write(OAMBaseAddr+4, 16+60) // Y position (raw Y = 76, adjusted = 60)
	mmu.Write(OAMBaseAddr+5, 8+40)  // X position (raw X = 48, adjusted = 40)
	mmu.Write(OAMBaseAddr+6, 0x24)  // Tile index
	mmu.Write(OAMBaseAddr+7, 0x00)  // Attributes (no flags)

	currentLine := 55
	spriteHeight := 8

	oamData := ExtractOAMData(mmu, currentLine, spriteHeight)

	assert.NotNil(t, oamData)
	assert.Equal(t, 40, len(oamData.Sprites))
	assert.Equal(t, currentLine, oamData.CurrentLine)
	assert.Equal(t, spriteHeight, oamData.SpriteHeight)

	// Test sprite 0 (should be visible on line 55)
	sprite0 := oamData.Sprites[0]
	assert.Equal(t, 0, sprite0.Index)
	assert.Equal(t, uint8(50), sprite0.Sprite.Y)
	assert.Equal(t, uint8(30), sprite0.Sprite.X)
	assert.Equal(t, uint8(0x42), sprite0.Sprite.TileIndex)
	assert.Equal(t, uint8(0x80), sprite0.Sprite.Flags)
	assert.True(t, sprite0.IsVisible) // Y=50, line=55, height=8 -> visible (50 <= 55 < 58)

	// Test sprite 1 (should NOT be visible on line 55)
	sprite1 := oamData.Sprites[1]
	assert.Equal(t, 1, sprite1.Index)
	assert.Equal(t, uint8(60), sprite1.Sprite.Y)
	assert.Equal(t, uint8(40), sprite1.Sprite.X)
	assert.Equal(t, uint8(0x24), sprite1.Sprite.TileIndex)
	assert.Equal(t, uint8(0x00), sprite1.Sprite.Flags)
	assert.False(t, sprite1.IsVisible) // Y=60, line=55, height=8 -> not visible (60 > 55)

	// Test active sprite count
	assert.Equal(t, 1, oamData.ActiveSprites)
}

func TestSpriteVisibility(t *testing.T) {
	mmu := memory.New()

	tests := []struct {
		name         string
		spriteY      int // Raw Y value (will be adjusted by -16)
		currentLine  int
		spriteHeight int
		expected     bool
	}{
		{"Sprite above line", 16 + 10, 20, 8, false},      // Y=10, line=20 -> not visible
		{"Sprite on line", 16 + 20, 20, 8, true},          // Y=20, line=20 -> visible
		{"Sprite below line start", 16 + 15, 20, 8, true}, // Y=15, line=20 -> visible (15 <= 20 < 23)
		{"Sprite below line end", 16 + 25, 20, 8, false},  // Y=25, line=20 -> not visible (25 > 20)
		{"16px sprite", 16 + 10, 20, 16, true},            // Y=10, line=20, height=16 -> visible (10 <= 20 < 26)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up sprite data
			mmu.Write(OAMBaseAddr, uint8(tt.spriteY))
			mmu.Write(OAMBaseAddr+1, 8+10) // X position doesn't matter for visibility test
			mmu.Write(OAMBaseAddr+2, 0x00) // Tile index doesn't matter
			mmu.Write(OAMBaseAddr+3, 0x00) // Attributes don't matter

			oamData := ExtractOAMData(mmu, tt.currentLine, tt.spriteHeight)

			assert.Equal(t, tt.expected, oamData.Sprites[0].IsVisible,
				"Sprite Y=%d, line=%d, height=%d should be visible=%v",
				tt.spriteY-16, tt.currentLine, tt.spriteHeight, tt.expected)
		})
	}
}

func TestDecodeAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes uint8
		expected   struct {
			BackgroundPriority bool
			FlipY              bool
			FlipX              bool
			PaletteNumber      int
		}
	}{
		{
			name:       "No flags set",
			attributes: 0x00,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{false, false, false, 0},
		},
		{
			name:       "Background priority",
			attributes: 0x80,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{true, false, false, 0},
		},
		{
			name:       "Flip Y",
			attributes: 0x40,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{false, true, false, 0},
		},
		{
			name:       "Flip X",
			attributes: 0x20,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{false, false, true, 0},
		},
		{
			name:       "Palette 1",
			attributes: 0x10,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{false, false, false, 1},
		},
		{
			name:       "All flags",
			attributes: 0xF0,
			expected: struct {
				BackgroundPriority bool
				FlipY              bool
				FlipX              bool
				PaletteNumber      int
			}{true, true, true, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sprite := SpriteInfo{
				Sprite: video.Sprite{
					Flags: tt.attributes,
				},
			}
			// manually parse flags for the test
			sprite.Sprite.BehindBG = (tt.attributes & 0x80) != 0
			sprite.Sprite.FlipY = (tt.attributes & 0x40) != 0
			sprite.Sprite.FlipX = (tt.attributes & 0x20) != 0
			sprite.Sprite.PaletteOBP1 = (tt.attributes & 0x10) != 0

			decoded := sprite.DecodeAttributes()

			assert.Equal(t, tt.expected.BackgroundPriority, decoded.BackgroundPriority)
			assert.Equal(t, tt.expected.FlipY, decoded.FlipY)
			assert.Equal(t, tt.expected.FlipX, decoded.FlipX)
			assert.Equal(t, tt.expected.PaletteNumber, decoded.PaletteNumber)
		})
	}
}

func TestGetVisibleSprites(t *testing.T) {
	mmu := memory.New()

	// Set up 3 sprites: 2 visible, 1 not visible
	// Sprite 0 - visible
	mmu.Write(OAMBaseAddr, 16+20)  // Y=20
	mmu.Write(OAMBaseAddr+1, 8+10) // X=10
	mmu.Write(OAMBaseAddr+2, 0x01) // Tile
	mmu.Write(OAMBaseAddr+3, 0x00) // Attributes

	// Sprite 1 - not visible
	mmu.Write(OAMBaseAddr+4, 16+100) // Y=100
	mmu.Write(OAMBaseAddr+5, 8+20)   // X=20
	mmu.Write(OAMBaseAddr+6, 0x02)   // Tile
	mmu.Write(OAMBaseAddr+7, 0x00)   // Attributes

	// Sprite 2 - visible
	mmu.Write(OAMBaseAddr+8, 16+18) // Y=18 (will be visible on line 22)
	mmu.Write(OAMBaseAddr+9, 8+30)  // X=30
	mmu.Write(OAMBaseAddr+10, 0x03) // Tile
	mmu.Write(OAMBaseAddr+11, 0x00) // Attributes

	oamData := ExtractOAMData(mmu, 22, 8) // Line 22
	visibleSprites := oamData.GetVisibleSprites()

	assert.Equal(t, 2, len(visibleSprites))
	assert.Equal(t, 0, visibleSprites[0].Index) // Sprite 0
	assert.Equal(t, 2, visibleSprites[1].Index) // Sprite 2
}

func TestFormatSummary(t *testing.T) {
	oamData := &OAMData{
		CurrentLine:   144,
		ActiveSprites: 3,
		SpriteHeight:  8,
	}

	summary := oamData.FormatSummary()
	expected := "Current Line: 144 | Active Sprites: 3/10 | Height: 8px"

	assert.Equal(t, expected, summary)
}
