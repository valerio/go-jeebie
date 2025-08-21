package debug

import (
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// MemoryReader provides read-only access to emulator memory for debug tools
// This interface decouples debug tools from the specific MMU implementation
type MemoryReader interface {
	// Read reads a single byte from the specified address
	Read(addr uint16) uint8

	// ReadBit reads a specific bit from a memory address
	ReadBit(bit uint8, addr uint16) bool
}

// ExtractOAMDataFromReader extracts OAM data using the generic memory reader interface
func ExtractOAMDataFromReader(reader MemoryReader, currentLine int, spriteHeight int) *OAMData {
	data := &OAMData{
		Sprites:      make([]SpriteInfo, OAMSpriteCount),
		CurrentLine:  currentLine,
		SpriteHeight: spriteHeight,
	}

	activeCount := 0

	for i := 0; i < OAMSpriteCount; i++ {
		baseAddr := uint16(OAMBaseAddr + i*OAMBytesPerSprite)

		rawY := reader.Read(baseAddr)
		rawX := reader.Read(baseAddr + 1)
		tileIndex := reader.Read(baseAddr + 2)
		attributes := reader.Read(baseAddr + 3)

		adjustedY := int(rawY) - SpriteYOffset
		adjustedX := int(rawX) - SpriteXOffset

		isVisible := (adjustedY <= currentLine) &&
			(adjustedY+spriteHeight > currentLine)

		if isVisible {
			activeCount++
		}

		// create a Sprite structure matching the video package format
		sprite := video.Sprite{
			Y:         uint8(adjustedY),
			X:         uint8(adjustedX),
			TileIndex: tileIndex,
			Flags:     attributes,
		}
		// parse the flags
		sprite.PaletteOBP1 = bit.IsSet(4, attributes)
		sprite.FlipX = bit.IsSet(5, attributes)
		sprite.FlipY = bit.IsSet(6, attributes)
		sprite.BehindBG = bit.IsSet(7, attributes)

		data.Sprites[i] = SpriteInfo{
			Index:     i,
			Sprite:    sprite,
			IsVisible: isVisible,
		}
	}

	data.ActiveSprites = activeCount
	return data
}

// ExtractVRAMDataFromReader extracts VRAM data using the generic memory reader interface
func ExtractVRAMDataFromReader(reader MemoryReader) *VRAMData {
	data := &VRAMData{
		TilePatterns: make([]video.Tile, TilePatternCount),
	}

	// Extract all 384 tile patterns from VRAM
	for i := range TilePatternCount {
		baseAddr := uint16(VRAMBaseAddr + i*TileDataSize)
		data.TilePatterns[i] = video.FetchTileWithIndex(reader, baseAddr, i)
	}

	// Extract tilemap information
	data.TilemapInfo = extractTilemapInfoFromReader(reader)

	return data
}

func extractTilemapInfoFromReader(reader MemoryReader) TilemapInfo {
	lcdc := reader.Read(0xFF40)

	// Check LCDC flags
	backgroundEnabled := (lcdc & 0x01) != 0
	windowEnabled := (lcdc & 0x20) != 0

	return TilemapInfo{
		BackgroundActive: backgroundEnabled,
		WindowActive:     windowEnabled,
		LCDCValue:        lcdc,
	}
}
