package debug

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/video"
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
	Index     int
	Sprite    video.Sprite // embed the shared sprite structure
	IsVisible bool
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

// DecodeAttributes returns the parsed sprite attributes
// Now delegates to the embedded Sprite structure
func (s *SpriteInfo) DecodeAttributes() struct {
	BackgroundPriority bool
	FlipY              bool
	FlipX              bool
	PaletteNumber      int
} {
	return struct {
		BackgroundPriority bool
		FlipY              bool
		FlipX              bool
		PaletteNumber      int
	}{
		BackgroundPriority: s.Sprite.BehindBG,
		FlipY:              s.Sprite.FlipY,
		FlipX:              s.Sprite.FlipX,
		PaletteNumber:      map[bool]int{false: 0, true: 1}[s.Sprite.PaletteOBP1],
	}
}

func (s *SpriteInfo) String() string {
	status := "OFF"
	if s.IsVisible {
		status = "ACTIVE"
	}
	return fmt.Sprintf("Sprite %2d: Y=%3d X=%3d  Tile=0x%02X Flags=0x%02X [%s]",
		s.Index, s.Sprite.Y, s.Sprite.X, s.Sprite.TileIndex, s.Sprite.Flags, status)
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
