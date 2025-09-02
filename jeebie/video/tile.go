package video

import "github.com/valerio/go-jeebie/jeebie/bit"

const (
	TileWidth        = 8   // Tile width in pixels
	TileHeight       = 8   // Tile height in pixels
	TileBytes        = 16  // Bytes per tile (8 rows × 2 bytes/row)
	TilemapWidth     = 32  // Tilemap width in tiles
	TilemapHeight    = 32  // Tilemap height in tiles
	TilemapPixelSize = 256 // Tilemap size in pixels (32 tiles × 8 pixels)
)

// TileRow represents one row of a tile pattern (8 pixels).
//
// Game Boy tiles are 8x8 pixels, with 2 bits per pixel allowing 4 colors.
// Each tile row uses 2 bytes in a bit-plane format:
//
//	Byte 1 (Low):  Bit plane 0 - provides bit 0 of each pixel's color
//	Byte 2 (High): Bit plane 1 - provides bit 1 of each pixel's color
//
// Bit 7 represents the leftmost pixel, bit 0 the rightmost:
//
//	Bit:     7 6 5 4 3 2 1 0
//	Pixel:   0 1 2 3 4 5 6 7
//
// Example: Bytes $3C and $7E represent a row:
//
//	Low  (0x3C): 0 0 1 1 1 1 0 0
//	High (0x7E): 0 1 1 1 1 1 1 0
//	            -----------------
//	Colors:      0 2 3 3 3 3 2 0
//
// Each pixel's 2-bit color index (0-3) is formed by combining the
// corresponding bits from both bytes. The actual display color is
// determined by the palette registers (BGP for background, OBP0/OBP1
// for sprites). For sprites, color 0 is always transparent.
//
// A complete 8x8 tile occupies 16 bytes (8 rows × 2 bytes/row) in VRAM.
//
// Reference: https://gbdev.io/pandocs/Tile_Data.html
type TileRow struct {
	Low  byte
	High byte
}

// GetPixel extracts a pixel color (0-3) from the tile row.
// pixelX should be 0-7, where 0 is the leftmost pixel.
func (t TileRow) GetPixel(pixelX int) int {
	// bit 7 is leftmost pixel, bit 0 is rightmost
	bitIndex := uint8(7 - pixelX)

	pixel := 0
	if bit.IsSet(bitIndex, t.Low) {
		pixel |= 1
	}
	if bit.IsSet(bitIndex, t.High) {
		pixel |= 2
	}

	return pixel
}

// GetPixelFlipped extracts a pixel color with horizontal flip.
// Used for sprite rendering with the flip X attribute.
func (t TileRow) GetPixelFlipped(pixelX int) int {
	// when flipped, bit 0 is leftmost pixel, bit 7 is rightmost
	bitIndex := uint8(pixelX)

	pixel := 0
	if bit.IsSet(bitIndex, t.Low) {
		pixel |= 1
	}
	if bit.IsSet(bitIndex, t.High) {
		pixel |= 2
	}

	return pixel
}

// Tile represents a complete 8x8 tile pattern.
// Each tile consists of 8 rows, totaling 16 bytes in VRAM.
type Tile struct {
	Index int // optional tile index (0-383 for VRAM tiles)
	Rows  [8]TileRow
}

// GetPixel returns the color index (0-3) for a pixel at (x, y).
// x and y should be 0-7, where (0,0) is the top-left pixel.
func (t *Tile) GetPixel(x, y int) int {
	if y < 0 || y >= 8 || x < 0 || x >= 8 {
		return 0
	}
	return t.Rows[y].GetPixel(x)
}

// Pixels returns the tile as an 8x8 array of GBColor values.
// This provides compatibility with the debug package.
func (t *Tile) Pixels() [8][8]GBColor {
	var pixels [8][8]GBColor
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			pixels[y][x] = GBColor(t.Rows[y].GetPixel(x))
		}
	}
	return pixels
}

// FetchTile reads a complete tile from memory at the given address.
// Each tile is 16 bytes (8 rows × 2 bytes per row).
// The index field is not set - use FetchTileWithIndex if you need it.
func FetchTile(memory MemoryReader, baseAddr uint16) Tile {
	var tile Tile
	for row := 0; row < 8; row++ {
		addr := baseAddr + uint16(row*2)
		tile.Rows[row] = TileRow{
			Low:  memory.Read(addr),
			High: memory.Read(addr + 1),
		}
	}
	return tile
}

// FetchTileWithIndex reads a tile and sets its index.
func FetchTileWithIndex(memory MemoryReader, baseAddr uint16, index int) Tile {
	tile := FetchTile(memory, baseAddr)
	tile.Index = index
	return tile
}

// MemoryReader interface for reading from memory.
// TODO: unify these into 2 shared interfaces?
type MemoryReader interface {
	Read(addr uint16) byte
}

// RenderTileToBuffer renders a single tile to a buffer at the specified position.
// The buffer is assumed to be RGBA format (0xAABBGGRR).
// x, y are the top-left position in the buffer, stride is the buffer width.
func RenderTileToBuffer(tile *Tile, buffer []uint32, x, y, stride int, palette []uint32) {
	for ty := 0; ty < 8; ty++ {
		if y+ty < 0 || y+ty >= stride {
			continue
		}
		for tx := 0; tx < 8; tx++ {
			if x+tx < 0 || x+tx >= stride {
				continue
			}
			colorIndex := tile.Rows[ty].GetPixel(tx)
			offset := (y+ty)*stride + (x + tx)
			if offset >= 0 && offset < len(buffer) {
				buffer[offset] = palette[colorIndex]
			}
		}
	}
}

// ApplyPalette converts a GB palette byte to RGBA colors.
// The palette byte maps 2-bit color indices to 2-bit shade values:
// Bits 0-1: Color 0, Bits 2-3: Color 1, Bits 4-5: Color 2, Bits 6-7: Color 3
func ApplyPalette(paletteByte byte) []uint32 {
	palette := make([]uint32, 4)
	for i := range 4 {
		shade := (paletteByte >> (i * 2)) & 0x3
		palette[i] = gbShadeToRGBA(shade)
	}
	return palette
}

// gbShadeToRGBA converts a 2-bit GB shade to RGBA color.
func gbShadeToRGBA(shade byte) uint32 {
	// Swap byte order from RGBA to ABGR for internal representation
	switch shade {
	case 0:
		return uint32(WhiteColor)
	case 1:
		return uint32(LightGreyColor)
	case 2:
		return uint32(DarkGreyColor)
	case 3:
		return uint32(BlackColor)
	default:
		return uint32(WhiteColor)
	}
}

// RenderTilemapToBuffer renders a complete 32x32 tilemap to a buffer.
// The tilemap is 256x256 pixels (32x32 tiles of 8x8 pixels each).
func RenderTilemapToBuffer(memory MemoryReader, tilemapAddr uint16, tileDataAddr uint16,
	buffer []uint32, palette []uint32, signed bool) {

	for ty := 0; ty < TilemapHeight; ty++ {
		for tx := 0; tx < TilemapWidth; tx++ {
			// Get tile index from tilemap
			tileIndex := memory.Read(tilemapAddr + uint16(ty*TilemapWidth+tx))

			// Calculate tile data address
			var tileAddr uint16
			if signed && tileIndex < 128 {
				// Unsigned addressing (0-127)
				tileAddr = tileDataAddr + uint16(tileIndex)*TileBytes
			} else if signed {
				// Signed addressing (128-255 maps to -128 to -1)
				offset := int8(tileIndex)
				tileAddr = uint16(int32(tileDataAddr) + int32(offset)*TileBytes)
			} else {
				// Unsigned addressing
				tileAddr = tileDataAddr + uint16(tileIndex)*TileBytes
			}

			// Fetch and render tile
			tile := FetchTile(memory, tileAddr)
			RenderTileToBuffer(&tile, buffer, tx*TileWidth, ty*TileHeight, TilemapPixelSize, palette)
		}
	}
}

// RenderSpritesToBuffer renders all sprites to a buffer.
// Handles sprite priority, flipping, and palettes.
func RenderSpritesToBuffer(sprites []Sprite, memory MemoryReader, buffer []uint32,
	palette0 []uint32, palette1 []uint32, width, height int) {

	// Clear buffer with transparent
	for i := range buffer {
		buffer[i] = 0x00000000
	}

	// Render sprites in reverse order (lower priority first)
	for i := len(sprites) - 1; i >= 0; i-- {
		sprite := sprites[i]

		// Choose palette
		palette := palette0
		if sprite.PaletteOBP1 {
			palette = palette1
		}

		// Fetch tile(s) - handle 8x16 sprites
		spriteHeight := sprite.Height
		if spriteHeight == 0 {
			spriteHeight = 8
		}

		for tileOffset := 0; tileOffset < spriteHeight; tileOffset += 8 {
			tileIndex := sprite.TileIndex
			if spriteHeight == 16 && tileOffset == 8 {
				tileIndex++ // Second tile for 8x16 sprites
			}

			tileAddr := uint16(0x8000) + uint16(tileIndex)*TileBytes
			tile := FetchTile(memory, tileAddr)

			// Handle flipping
			if sprite.FlipY {
				// Flip tile rows
				for j := 0; j < 4; j++ {
					tile.Rows[j], tile.Rows[7-j] = tile.Rows[7-j], tile.Rows[j]
				}
			}

			// Render tile row by row with flip handling
			for row := 0; row < 8; row++ {
				y := int(sprite.Y) + row + tileOffset
				if y < 0 || y >= height {
					continue
				}

				for col := 0; col < 8; col++ {
					x := int(sprite.X) + col
					if x < 0 || x >= width {
						continue
					}

					var colorIndex int
					if sprite.FlipX {
						colorIndex = tile.Rows[row].GetPixelFlipped(col)
					} else {
						colorIndex = tile.Rows[row].GetPixel(col)
					}

					// Color 0 is transparent for sprites
					if colorIndex == 0 {
						continue
					}

					offset := y*width + x
					if offset >= 0 && offset < len(buffer) {
						buffer[offset] = palette[colorIndex]
					}
				}
			}
		}
	}
}
