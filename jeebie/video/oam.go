package video

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// Sprite represents a single sprite/object in OAM memory.
// The Game Boy has 40 sprites stored in OAM (Object Attribute Memory) from 0xFE00-0xFE9F.
type Sprite struct {
	Y         uint8 // Y position (actual screen position, without the +16 offset)
	X         uint8 // X position (actual screen position, without the +8 offset)
	TileIndex uint8 // Tile/pattern number (0-255)
	Flags     uint8 // Attribute flags byte
	OAMIndex  int   // OAM index (0-39)
	Height    int   // Sprite height (8 or 16 pixels, from LCDC bit 2)

	// parsed attribute flags for convenience
	PaletteOBP1 bool // false = OBP0, true = OBP1
	FlipX       bool // horizontally flip the sprite
	FlipY       bool // vertically flip the sprite
	BehindBG    bool // true = sprite is behind background (priority flag)

	// pixel priority mask - bit 7 is leftmost pixel, bit 0 is rightmost
	// a bit is set if this sprite has priority for that pixel after sprite-to-sprite priority resolution
	PixelMask uint8
}

func (s *Sprite) parseFlags() {
	s.PaletteOBP1 = bit.IsSet(4, s.Flags)
	s.FlipX = bit.IsSet(5, s.Flags)
	s.FlipY = bit.IsSet(6, s.Flags)
	s.BehindBG = bit.IsSet(7, s.Flags)
}

func (s *Sprite) HasPriorityForAnyPixel() bool {
	return s.PixelMask != 0
}

// HasPriorityForPixel returns true if this sprite has priority for the pixel at the given X position (0-7).
// Pixel 0 is the leftmost pixel, pixel 7 is the rightmost.
func (s *Sprite) HasPriorityForPixel(pixelX int) bool {
	if pixelX < 0 || pixelX > 7 {
		return false
	}
	pixelBit := uint8(1 << (7 - pixelX))
	return s.PixelMask&pixelBit != 0
}

// OAMBus is the interface OAM needs for memory access
type OAMBus interface {
	Read(address uint16) byte
}

// OAM manages Object Attribute Memory and sprite data.
type OAM struct {
	bus            OAMBus
	priorityBuffer SpritePriorityBuffer
	spriteBuffer   [10]Sprite // scanline sprites (hardware limit is 10)
}

func NewOAM(bus OAMBus) *OAM {
	return &OAM{
		bus: bus,
	}
}

// GetSpritesForScanline returns sprites that overlap the given scanline.
// Returns up to 10 sprites (hardware limit per scanline) with pre-resolved
// pixel priority.
//
// Priority is essentially sorting pixels by (X pos, OAM index), see priority
// buffer for a more thorough explanation.
func (o *OAM) GetSpritesForScanline(scanline int) []Sprite {
	sprites := o.spriteBuffer[:0]
	o.priorityBuffer.Clear()

	lcdc := o.bus.Read(addr.LCDC)
	spriteHeight := 8
	if bit.IsSet(2, lcdc) {
		spriteHeight = 16
	}

	// phase 1: scan through OAM and collect sprites
	for i := range 40 {
		baseAddr := addr.OAMStart + uint16(i*4)

		// read sprite Y position and check if it's on this scanline
		rawY := o.bus.Read(baseAddr)
		spriteY := int(rawY) - 16 // adjust for hardware offset

		// check if sprite overlaps this scanline
		// sprite is visible if: spriteY <= scanline < spriteY + height
		if spriteY <= scanline && scanline < spriteY+spriteHeight {
			rawX := o.bus.Read(baseAddr + 1)
			tileIndex := o.bus.Read(baseAddr + 2)
			flags := o.bus.Read(baseAddr + 3)

			sprite := Sprite{
				Y:         uint8(spriteY),
				X:         rawX - 8, // adjust for hardware offset
				TileIndex: tileIndex,
				Flags:     flags,
				OAMIndex:  i,
				Height:    spriteHeight,
				PixelMask: 0, // will be set after priority resolution
			}
			sprite.parseFlags()

			sprites = append(sprites, sprite)

			// resolve priority for this sprite's pixels
			for pixelX := range 8 {
				bufferX := int(sprite.X) + pixelX
				o.priorityBuffer.TryClaimPixel(bufferX, sprite.OAMIndex, int(sprite.X))
			}

			// hardware limit: maximum 10 sprites per scanline
			if len(sprites) >= 10 {
				break
			}
		}
	}

	// phase 2: set pixel priority masks based on priority resolution
	for i := range sprites {
		var mask uint8
		for pixelX := range 8 {
			bufferX := int(sprites[i].X) + pixelX
			if o.priorityBuffer.GetPriority(bufferX) == sprites[i].OAMIndex {
				mask |= (1 << (7 - pixelX)) // bit 7 is leftmost pixel
			}
		}
		sprites[i].PixelMask = mask
	}

	copy(o.spriteBuffer[:], sprites)
	return o.spriteBuffer[:len(sprites)]
}

func (o *OAM) readSprite(index int) Sprite {
	baseAddr := addr.OAMStart + uint16(index*4)

	rawY := o.bus.Read(baseAddr)
	rawX := o.bus.Read(baseAddr + 1)
	tileIndex := o.bus.Read(baseAddr + 2)
	flags := o.bus.Read(baseAddr + 3)

	lcdc := o.bus.Read(addr.LCDC)
	spriteHeight := 8
	if bit.IsSet(2, lcdc) {
		spriteHeight = 16
	}

	sprite := Sprite{
		Y:         rawY - 16,
		X:         rawX - 8,
		TileIndex: tileIndex,
		Flags:     flags,
		OAMIndex:  index,
		Height:    spriteHeight,
	}
	sprite.parseFlags()

	return sprite
}

// GetSprite returns a pointer to the sprite at the given index (0-39)
func (o *OAM) GetSprite(index int) *Sprite {
	if index < 0 || index >= 40 {
		return nil
	}
	sprite := o.readSprite(index)
	return &sprite
}

// GetAllSprites returns all 40 sprites. Useful for debug tools.
func (o *OAM) GetAllSprites() []Sprite {
	result := make([]Sprite, 40)
	for i := range 40 {
		result[i] = o.readSprite(i)
	}
	return result
}
