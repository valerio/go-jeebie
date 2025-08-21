package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

func TestOAMScan(t *testing.T) {
	// create a test MMU
	mmu := memory.New()
	oam := NewOAM(mmu)

	// write test sprite data to OAM memory
	// sprite 0: Y=50(+16), X=80(+8), tile=0x42, flags=0xE0
	mmu.Write(addr.OAMStart, 50+16)  // Y with offset
	mmu.Write(addr.OAMStart+1, 80+8) // X with offset
	mmu.Write(addr.OAMStart+2, 0x42) // tile index
	mmu.Write(addr.OAMStart+3, 0xE0) // flags: flip X, flip Y, behind BG

	// sprite 1: Y=100(+16), X=20(+8), tile=0x10, flags=0x10
	mmu.Write(addr.OAMStart+4, 100+16) // Y with offset
	mmu.Write(addr.OAMStart+5, 20+8)   // X with offset
	mmu.Write(addr.OAMStart+6, 0x10)   // tile index
	mmu.Write(addr.OAMStart+7, 0x10)   // flags: OBP1 palette

	// verify sprite 0
	sprite0 := oam.GetSprite(0)
	assert.NotNil(t, sprite0)
	assert.Equal(t, uint8(50), sprite0.Y, "Y position should be adjusted")
	assert.Equal(t, uint8(80), sprite0.X, "X position should be adjusted")
	assert.Equal(t, uint8(0x42), sprite0.TileIndex)
	assert.True(t, sprite0.FlipX, "FlipX should be set")
	assert.True(t, sprite0.FlipY, "FlipY should be set")
	assert.True(t, sprite0.BehindBG, "BehindBG should be set")
	assert.False(t, sprite0.PaletteOBP1, "Should use OBP0")

	// verify sprite 1
	sprite1 := oam.GetSprite(1)
	assert.NotNil(t, sprite1)
	assert.Equal(t, uint8(100), sprite1.Y)
	assert.Equal(t, uint8(20), sprite1.X)
	assert.Equal(t, uint8(0x10), sprite1.TileIndex)
	assert.False(t, sprite1.FlipX)
	assert.False(t, sprite1.FlipY)
	assert.False(t, sprite1.BehindBG)
	assert.True(t, sprite1.PaletteOBP1, "Should use OBP1")
}

func TestGetSpritesForScanline(t *testing.T) {
	mmu := memory.New()
	oam := NewOAM(mmu)

	// set up sprites at different Y positions
	// sprite 0: Y=10
	mmu.Write(addr.OAMStart, 10+16)
	mmu.Write(addr.OAMStart+1, 20+8)

	// sprite 1: Y=20
	mmu.Write(addr.OAMStart+4, 20+16)
	mmu.Write(addr.OAMStart+5, 30+8)

	// sprite 2: Y=20 (same scanline as sprite 1)
	mmu.Write(addr.OAMStart+8, 20+16)
	mmu.Write(addr.OAMStart+9, 40+8)

	// sprite 3: Y=50
	mmu.Write(addr.OAMStart+12, 50+16)
	mmu.Write(addr.OAMStart+13, 50+8)

	// test 8x8 sprites
	t.Run("8x8 sprites", func(t *testing.T) {
		// set 8x8 sprite mode (LCDC bit 2 = 0)
		mmu.Write(addr.LCDC, 0x00)

		// scanline 10: should find sprite 0
		sprites := oam.GetSpritesForScanline(10)
		assert.Len(t, sprites, 1)
		assert.Equal(t, 0, sprites[0].OAMIndex)

		// scanline 17: should find sprite 0 (still within 8 pixel height)
		sprites = oam.GetSpritesForScanline(17)
		assert.Len(t, sprites, 1)
		assert.Equal(t, 0, sprites[0].OAMIndex)

		// scanline 18: sprite 0 is now out of range
		sprites = oam.GetSpritesForScanline(18)
		assert.Empty(t, sprites)

		// scanline 20: should find sprites 1 and 2
		sprites = oam.GetSpritesForScanline(20)
		assert.Len(t, sprites, 2)
		assert.Equal(t, 1, sprites[0].OAMIndex)
		assert.Equal(t, 2, sprites[1].OAMIndex)

		// scanline 27: should find sprites 1 and 2 (last line)
		sprites = oam.GetSpritesForScanline(27)
		assert.Len(t, sprites, 2)
		assert.Equal(t, 1, sprites[0].OAMIndex)
		assert.Equal(t, 2, sprites[1].OAMIndex)

		// scanline 50: should find sprite 3
		sprites = oam.GetSpritesForScanline(50)
		assert.Len(t, sprites, 1)
		assert.Equal(t, 3, sprites[0].OAMIndex)
	})

	// test 8x16 sprites
	t.Run("8x16 sprites", func(t *testing.T) {
		// set 8x16 sprite mode (LCDC bit 2 = 1)
		mmu.Write(addr.LCDC, 0x04)

		// scanline 10: should find sprite 0
		sprites := oam.GetSpritesForScanline(10)
		assert.Len(t, sprites, 1)
		assert.Equal(t, 0, sprites[0].OAMIndex)

		// scanline 25: should find sprites 0, 1, and 2
		sprites = oam.GetSpritesForScanline(25)
		assert.Len(t, sprites, 3)
		assert.Equal(t, 0, sprites[0].OAMIndex)
		assert.Equal(t, 1, sprites[1].OAMIndex)
		assert.Equal(t, 2, sprites[2].OAMIndex)

		// scanline 35: should find sprites 1 and 2
		sprites = oam.GetSpritesForScanline(35)
		assert.Len(t, sprites, 2)
		assert.Equal(t, 1, sprites[0].OAMIndex)
		assert.Equal(t, 2, sprites[1].OAMIndex)
	})
}

func TestSpriteLimit(t *testing.T) {
	mmu := memory.New()
	oam := NewOAM(mmu)

	// create 15 sprites all on the same scanline (Y=50)
	for i := 0; i < 15; i++ {
		baseAddr := addr.OAMStart + uint16(i*4)
		mmu.Write(baseAddr, 50+16)        // Y
		mmu.Write(baseAddr+1, uint8(i)+8) // X (different for each)
		mmu.Write(baseAddr+2, uint8(i))   // tile
		mmu.Write(baseAddr+3, 0)          // flags
	}

	// set 8x8 sprite mode
	mmu.Write(addr.LCDC, 0x00)

	// get sprites for scanline 50
	sprites := oam.GetSpritesForScanline(50)

	// should return exactly 10 sprites (hardware limit)
	assert.Len(t, sprites, 10, "Should return maximum 10 sprites per scanline")

	// should return the first 10 sprites in OAM order
	for i := 0; i < 10; i++ {
		assert.Equal(t, i, sprites[i].OAMIndex, "Should return sprites in OAM order")
		assert.NotNil(t, sprites[i], "Should have sprite data")
	}
}

func TestGetAllSprites(t *testing.T) {
	mmu := memory.New()
	oam := NewOAM(mmu)

	// write some test data
	for i := 0; i < 40; i++ {
		baseAddr := addr.OAMStart + uint16(i*4)
		mmu.Write(baseAddr, uint8(i)+16)    // Y
		mmu.Write(baseAddr+1, uint8(i*2)+8) // X
		mmu.Write(baseAddr+2, uint8(i))     // tile
		mmu.Write(baseAddr+3, 0)            // flags
	}

	sprites := oam.GetAllSprites()
	assert.Len(t, sprites, 40, "Should return all 40 sprites")

	// verify a few sprites
	assert.Equal(t, uint8(0), sprites[0].Y)
	assert.Equal(t, uint8(0), sprites[0].X)
	assert.Equal(t, uint8(0), sprites[0].TileIndex)

	assert.Equal(t, uint8(10), sprites[10].Y)
	assert.Equal(t, uint8(20), sprites[10].X)
	assert.Equal(t, uint8(10), sprites[10].TileIndex)
}

func TestDirectMemoryRead(t *testing.T) {
	mmu := memory.New()
	oam := NewOAM(mmu)

	// write initial sprite data
	mmu.Write(addr.OAMStart, 50+16)
	sprite := oam.GetSprite(0)
	assert.Equal(t, uint8(50), sprite.Y)

	// modify OAM memory
	mmu.Write(addr.OAMStart, 60+16)

	// should immediately return new value (no caching)
	sprite = oam.GetSprite(0)
	assert.Equal(t, uint8(60), sprite.Y, "Should have new value immediately")
}

func TestEdgeCases(t *testing.T) {
	mmu := memory.New()
	oam := NewOAM(mmu)

	// test sprite at screen boundaries
	t.Run("boundary positions", func(t *testing.T) {
		// sprite at Y=0 (stored as 16 in OAM)
		mmu.Write(addr.OAMStart, 16)
		mmu.Write(addr.OAMStart+1, 8) // X=0 (stored as 8)

		sprite := oam.GetSprite(0)
		assert.Equal(t, uint8(0), sprite.Y)
		assert.Equal(t, uint8(0), sprite.X)

		// sprite off-screen (Y=255, X=255)
		mmu.Write(addr.OAMStart+4, 255) // Y=239 after adjustment
		mmu.Write(addr.OAMStart+5, 255) // X=247 after adjustment

		sprite = oam.GetSprite(1)
		assert.Equal(t, uint8(239), sprite.Y)
		assert.Equal(t, uint8(247), sprite.X)
	})

	// test invalid sprite index
	t.Run("invalid index", func(t *testing.T) {
		assert.Nil(t, oam.GetSprite(-1))
		assert.Nil(t, oam.GetSprite(40))
		assert.Nil(t, oam.GetSprite(100))
	})
}
