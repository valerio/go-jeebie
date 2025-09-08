package debug

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/video"
)

const (
	TilemapWidth  = 32
	TilemapHeight = 32
	ScreenWidth   = 20
	ScreenHeight  = 18
)

type SpriteVisualizer struct {
	Sprites      []Sprite
	TileData     []video.Tile
	CurrentLine  uint8
	SpriteHeight int
	PaletteOBP0  uint8
	PaletteOBP1  uint8
}

type Sprite struct {
	Info     SpriteInfo
	TileData video.Tile
	OnScreen bool
	X        int
	Y        int
}

type BackgroundVisualizer struct {
	Tilemap           [TilemapHeight][TilemapWidth]uint8
	WindowTilemap     [TilemapHeight][TilemapWidth]uint8
	TileData          []video.Tile
	ScrollX           uint8
	ScrollY           uint8
	WindowX           uint8
	WindowY           uint8
	WindowEnabled     bool
	BGEnabled         bool
	TilemapBase       uint16
	WindowTilemapBase uint16
	TileDataBase      uint16
	PaletteBGP        uint8
}

type PaletteVisualizer struct {
	BGP  PaletteInfo
	OBP0 PaletteInfo
	OBP1 PaletteInfo
}

type PaletteInfo struct {
	Raw    uint8
	Colors [4]video.GBColor
}

func extractSpriteDataInto(reader MemoryReader, currentLine uint8, vis *SpriteVisualizer) {
	vis.CurrentLine = currentLine

	lcdc := reader.Read(addr.LCDC)
	vis.SpriteHeight = 8
	if (lcdc & 0x04) != 0 {
		vis.SpriteHeight = 16
	}

	vis.PaletteOBP0 = reader.Read(addr.OBP0)
	vis.PaletteOBP1 = reader.Read(addr.OBP1)

	if vis.TileData == nil {
		vis.TileData = make([]video.Tile, 256)
	}
	for i := 0; i < 256; i++ {
		baseAddr := uint16(0x8000 + i*16)
		vis.TileData[i] = video.FetchTileWithIndex(reader, baseAddr, i)
	}

	oamData := ExtractOAMDataFromReader(reader, int(currentLine), vis.SpriteHeight)

	if cap(vis.Sprites) < len(oamData.Sprites) {
		vis.Sprites = make([]Sprite, len(oamData.Sprites))
	} else {
		vis.Sprites = vis.Sprites[:len(oamData.Sprites)]
	}

	for i, sprite := range oamData.Sprites {
		s := Sprite{
			Info: sprite,
			X:    int(sprite.Sprite.X),
			Y:    int(sprite.Sprite.Y),
		}

		tileIndex := sprite.Sprite.TileIndex
		if vis.SpriteHeight == 16 {
			tileIndex &= 0xFE
		}
		s.TileData = vis.TileData[tileIndex]

		x := int(s.X)
		y := int(s.Y)
		s.OnScreen = (x >= 0 && x < 160) && (y >= 0 && y < 144)

		vis.Sprites[i] = s
	}
}

func ExtractSpriteData(reader MemoryReader, currentLine uint8) *SpriteVisualizer {
	vis := &SpriteVisualizer{}
	extractSpriteDataInto(reader, currentLine, vis)
	return vis
}

func extractBackgroundDataInto(reader MemoryReader, vis *BackgroundVisualizer) {
	lcdc := reader.Read(addr.LCDC)
	vis.BGEnabled = (lcdc & 0x01) != 0
	vis.WindowEnabled = (lcdc & 0x20) != 0

	if (lcdc & 0x08) != 0 {
		vis.TilemapBase = 0x9C00
	} else {
		vis.TilemapBase = 0x9800
	}

	// Window tilemap base (LCDC bit 6)
	if (lcdc & 0x40) != 0 {
		vis.WindowTilemapBase = 0x9C00
	} else {
		vis.WindowTilemapBase = 0x9800
	}

	if (lcdc & 0x10) != 0 {
		vis.TileDataBase = 0x8000
	} else {
		vis.TileDataBase = 0x8800
	}

	vis.ScrollX = reader.Read(addr.SCX)
	vis.ScrollY = reader.Read(addr.SCY)
	vis.WindowX = reader.Read(addr.WX)
	vis.WindowY = reader.Read(addr.WY)
	vis.PaletteBGP = reader.Read(addr.BGP)

	// Read background tilemap
	for row := 0; row < TilemapHeight; row++ {
		for col := 0; col < TilemapWidth; col++ {
			addr := vis.TilemapBase + uint16(row*TilemapWidth+col)
			vis.Tilemap[row][col] = reader.Read(addr)
		}
	}

	// Read window tilemap
	for row := 0; row < TilemapHeight; row++ {
		for col := 0; col < TilemapWidth; col++ {
			addr := vis.WindowTilemapBase + uint16(row*TilemapWidth+col)
			vis.WindowTilemap[row][col] = reader.Read(addr)
		}
	}

	if vis.TileData == nil {
		vis.TileData = make([]video.Tile, 384)
	}

	// First 256 tiles from 0x8000-0x8FFF
	for i := 0; i < 256; i++ {
		baseAddr := uint16(0x8000 + i*16)
		vis.TileData[i] = video.FetchTileWithIndex(reader, baseAddr, i)
	}

	// Next 128 tiles from 0x9000-0x97FF (for signed addressing mode)
	for i := 256; i < 384; i++ {
		baseAddr := uint16(0x9000 + (i-256)*16)
		vis.TileData[i] = video.FetchTileWithIndex(reader, baseAddr, i-256)
	}
}

func ExtractBackgroundData(reader MemoryReader) *BackgroundVisualizer {
	vis := &BackgroundVisualizer{}
	extractBackgroundDataInto(reader, vis)
	return vis
}

func ExtractPaletteData(reader MemoryReader) *PaletteVisualizer {
	vis := &PaletteVisualizer{}

	vis.BGP = extractPalette(reader.Read(addr.BGP))
	vis.OBP0 = extractPalette(reader.Read(addr.OBP0))
	vis.OBP1 = extractPalette(reader.Read(addr.OBP1))

	return vis
}

func extractPalette(paletteReg uint8) PaletteInfo {
	info := PaletteInfo{
		Raw: paletteReg,
	}

	for i := 0; i < 4; i++ {
		colorIndex := (paletteReg >> (i * 2)) & 0x03
		info.Colors[i] = video.GBColor(colorIndex)
	}

	return info
}

func (sv *SpriteVisualizer) GetVisibleSprites() []Sprite {
	var visible []Sprite
	for _, sprite := range sv.Sprites {
		if sprite.Info.IsVisible && sprite.OnScreen {
			visible = append(visible, sprite)
		}
	}
	return visible
}

func (sv *SpriteVisualizer) GetSpritesOnLine(line uint8) []Sprite {
	var sprites []Sprite
	for _, sprite := range sv.Sprites {
		spriteY := sprite.Y
		if line >= uint8(spriteY) && line < uint8(spriteY+sv.SpriteHeight) {
			sprites = append(sprites, sprite)
		}
	}
	return sprites
}

func (bv *BackgroundVisualizer) GetViewportTiles() [ScreenHeight][ScreenWidth]uint8 {
	var viewport [ScreenHeight][ScreenWidth]uint8

	startTileY := int(bv.ScrollY) / 8
	startTileX := int(bv.ScrollX) / 8

	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			tileY := (startTileY + y) % TilemapHeight
			tileX := (startTileX + x) % TilemapWidth
			viewport[y][x] = bv.Tilemap[tileY][tileX]
		}
	}

	return viewport
}

func (bv *BackgroundVisualizer) GetWindowViewport() (active bool, startX, startY int) {
	if !bv.WindowEnabled {
		return false, 0, 0
	}

	if bv.WindowX < 7 || bv.WindowX >= 167 {
		return false, 0, 0
	}

	return true, int(bv.WindowX) - 7, int(bv.WindowY)
}

func ApplyPalette(color video.GBColor, palette PaletteInfo) video.GBColor {
	return palette.Colors[color&0x03]
}
