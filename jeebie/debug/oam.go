package debug

import (
	"fmt"
)

const (
	OAMBaseAddr       = 0xFE00
	OAMSpriteCount    = 40
	OAMBytesPerSprite = 4
	SpriteYOffset     = 16
	SpriteXOffset     = 8
	MaxSpritesPerLine = 10
)

// Sprite attribute bit positions
const (
	AttrBackgroundPriority = 7
	AttrFlipY              = 6
	AttrFlipX              = 5
	AttrPaletteNumber      = 4
)

type SpriteInfo struct {
	Index      int
	Y          int
	X          int
	TileIndex  uint8
	Attributes uint8
	IsVisible  bool
}

type SpriteAttributes struct {
	BackgroundPriority bool
	FlipY              bool
	FlipX              bool
	PaletteNumber      int
}

type OAMData struct {
	Sprites       []SpriteInfo
	CurrentLine   int
	ActiveSprites int
	SpriteHeight  int
}

func ExtractOAMData(reader MemoryReader, currentLine int, spriteHeight int) *OAMData {
	return ExtractOAMDataFromReader(reader, currentLine, spriteHeight)
}

func (s *SpriteInfo) DecodeAttributes() SpriteAttributes {
	return SpriteAttributes{
		BackgroundPriority: (s.Attributes & (1 << AttrBackgroundPriority)) != 0,
		FlipY:              (s.Attributes & (1 << AttrFlipY)) != 0,
		FlipX:              (s.Attributes & (1 << AttrFlipX)) != 0,
		PaletteNumber:      int((s.Attributes & (1 << AttrPaletteNumber)) >> AttrPaletteNumber),
	}
}

func (s *SpriteInfo) String() string {
	status := "OFF"
	if s.IsVisible {
		status = "ACTIVE"
	}
	return fmt.Sprintf("Sprite %2d: Y=%3d X=%3d  Tile=0x%02X Flags=0x%02X [%s]",
		s.Index, s.Y, s.X, s.TileIndex, s.Attributes, status)
}

func (data *OAMData) GetVisibleSprites() []SpriteInfo {
	visible := make([]SpriteInfo, 0, data.ActiveSprites)
	for _, sprite := range data.Sprites {
		if sprite.IsVisible {
			visible = append(visible, sprite)
		}
	}
	return visible
}

func (data *OAMData) FormatSummary() string {
	return fmt.Sprintf("Current Line: %d | Active Sprites: %d/%d | Height: %dpx",
		data.CurrentLine, data.ActiveSprites, MaxSpritesPerLine, data.SpriteHeight)
}
