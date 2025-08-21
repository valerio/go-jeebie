package video

import "github.com/valerio/go-jeebie/jeebie/bit"

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
