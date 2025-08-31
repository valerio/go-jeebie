package debug

import (
	"github.com/valerio/go-jeebie/jeebie/video"
)

// FetchTileForIndex fetches a tile using the same logic as the GPU
// This ensures debug visualization matches actual rendering
func FetchTileForIndex(reader MemoryReader, tileIndex byte, baseAddr uint16, signed bool) video.Tile {
	var tileAddr uint16

	if signed {
		// Signed addressing: interpret as -128 to 127
		// In signed mode, baseAddr is 0x8800
		// Index 0 should map to 0x9000 (offset +0x800 from 0x8800)
		// Index 128 (0x80, -128 as signed) should map to 0x8800
		signedIndex := int8(tileIndex)
		tileAddr = uint16(int(baseAddr) + int(signedIndex)*16)
	} else {
		// Unsigned addressing: 0 to 255
		// In unsigned mode, baseAddr is 0x8000
		tileAddr = baseAddr + uint16(tileIndex)*16
	}

	// Fetch all 8 rows of the tile (16 bytes total)
	var tile video.Tile
	tile.Index = int(tileIndex)

	for row := 0; row < 8; row++ {
		rowAddr := tileAddr + uint16(row*2)
		tile.Rows[row] = video.TileRow{
			Low:  reader.Read(rowAddr),
			High: reader.Read(rowAddr + 1),
		}
	}

	return tile
}

// GetTileForBackgroundIndex gets the correct tile for a background/window tile index
// taking into account the current addressing mode
func GetTileForBackgroundIndex(tiles []video.Tile, tileIndex byte, useSigned bool) video.Tile {
	if !useSigned {
		// Unsigned mode: direct mapping
		return tiles[tileIndex]
	}

	// Signed mode: remap indices
	// Indices 0-127 map to tiles 256-383 (in 0x9000-0x97FF range)
	// Indices 128-255 map to tiles 0-127 (in 0x8800-0x8FFF range)
	if tileIndex < 128 {
		// Check if we have enough tiles loaded
		arrayIndex := int(tileIndex) + 256
		if arrayIndex < len(tiles) {
			return tiles[arrayIndex]
		}
		// Fallback if we only loaded 256 tiles
		return tiles[0]
	}

	// Index 128-255 maps to tiles 0-127
	return tiles[int(tileIndex)-128]
}
