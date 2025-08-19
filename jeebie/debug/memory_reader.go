package debug

import (
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

		data.Sprites[i] = SpriteInfo{
			Index:      i,
			Y:          adjustedY,
			X:          adjustedX,
			TileIndex:  tileIndex,
			Attributes: attributes,
			IsVisible:  isVisible,
		}
	}

	data.ActiveSprites = activeCount
	return data
}

// ExtractVRAMDataFromReader extracts VRAM data using the generic memory reader interface
func ExtractVRAMDataFromReader(reader MemoryReader) *VRAMData {
	data := &VRAMData{
		TilePatterns: make([]TilePattern, TilePatternCount),
	}

	// Extract all 384 tile patterns from VRAM
	for i := 0; i < TilePatternCount; i++ {
		data.TilePatterns[i] = extractTilePatternFromReader(reader, i)
	}

	// Extract tilemap information
	data.TilemapInfo = extractTilemapInfoFromReader(reader)

	return data
}

func extractTilePatternFromReader(reader MemoryReader, tileIndex int) TilePattern {
	pattern := TilePattern{Index: tileIndex}

	baseAddr := uint16(VRAMBaseAddr + tileIndex*TileDataSize)

	for y := 0; y < TilePixelHeight; y++ {
		// Each row is 2 bytes (low bit plane + high bit plane)
		lowByte := reader.Read(baseAddr + uint16(y*2))
		highByte := reader.Read(baseAddr + uint16(y*2) + 1)

		for x := 0; x < TilePixelWidth; x++ {
			// Extract bit from each plane
			bitPos := uint(7 - x)
			lowBit := (lowByte >> bitPos) & 1
			highBit := (highByte >> bitPos) & 1

			// Combine to get 2-bit color value
			colorValue := (highBit << 1) | lowBit
			pattern.Pixels[y][x] = video.GBColor(colorValue)
		}
	}

	return pattern
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
