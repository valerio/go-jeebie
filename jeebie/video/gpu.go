package video

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

type GpuMode int

const (
	oamRead GpuMode = iota
	vramRead
	hblank
	vblank
)

const (
	hblankCycles       = 204
	oamScanlineCycles  = 80
	vramScanlineCycles = 172
	scanlineCycles     = oamScanlineCycles + vramScanlineCycles + hblankCycles
)

type GPU struct {
	memory      *memory.MMU
	screen      *Screen
	framebuffer *FrameBuffer

	line       uint8
	vblankLine int
	mode       GpuMode
	cycles     int
}

func NewGpu(screen *Screen, memory *memory.MMU) *GPU {
	fb := NewFrameBuffer()
	return &GPU{
		framebuffer: fb,
		screen:      screen,
		memory:      memory,
		mode:        oamRead,
		line:        0,
		cycles:      0,
	}
}

// Tick simulates gpu behaviour for a certain amount of clock cycles.
func (g *GPU) Tick(cycles int) {
	g.cycles += cycles

	switch g.mode {
	case oamRead:
		if g.cycles >= oamScanlineCycles {
			g.cycles %= oamScanlineCycles
			g.mode = vramRead
		}
		break
	case vramRead:
		if g.cycles >= vramScanlineCycles {
			g.cycles %= vramScanlineCycles
			g.mode = hblank
		}
		break
	case hblank:
		if g.cycles >= hblankCycles {
			g.line++
			g.memory.Write(addr.LY, g.line)
			// TODO: g.compareLYtoLYC()

			g.cycles %= hblankCycles
			g.mode = oamRead

			if g.line == 144 {
				g.mode = vblank
				g.vblankLine = 0

				// set vblank interrupt (bit 0)
				g.memory.RequestInterrupt(0)

				// g.drawFrame()
			}
		}
		break
	case vblank:
		if g.cycles >= scanlineCycles {
			g.line++
			g.cycles %= scanlineCycles

			if g.line == 154 {
				g.framebuffer.DrawNoise()
				// g.drawScanline()
				g.screen.Draw(g.framebuffer.ToSlice())
				g.line = 0
				g.mode = oamRead
			}
		}
		break
	}
}

func (g *GPU) drawScanline() {
	if g.readLCDCVariable(lcdDisplayEnable) == 0 {
		// display is disabled
		return
	}

	if g.readLCDCVariable(bgDisplay) == 0 {
		// drawing the background is disabled
		return
	}

	g.drawTiles()
}

func (g *GPU) drawTiles() {
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			g.drawTile(x, y)
		}
	}
}

func (g *GPU) drawTile(tileX, tileY int) {
	useTileSetZero := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	useTileMapZero := g.readLCDCVariable(bgTileMapDisplaySelect) == 0

	// select the correct starting address based on which tileMap/tileSet is set
	tileSetAddress := uint16(0x8000)
	if useTileSetZero {
		tileSetAddress = 0x8800
	}

	tileMapAddress := 0x9C00
	if useTileMapZero {
		tileMapAddress = 0x9800
	}

	// 32 tiles per scanline
	tileIndex := tileY*32 + tileX
	tileNumberAddress := tileMapAddress + tileIndex
	// grab the tile number
	tileNumber := g.memory.Read(uint16(tileNumberAddress))
	// offset is tile number times tile size (16 bytes per tile)
	tileOffset := uint16(tileNumber) * 16

	tileAddress := tileSetAddress + tileOffset
	tile := newTile(uint16(tileAddress), g.memory)

	// x and y inside a tile are offset by 8 in terms of address
	fbX := uint(8 * tileX)
	fbY := uint(8 * tileY)

	for y := uint(0); y < 8; y++ {
		for x := uint(0); x < 8; x++ {
			color := tile.getPixel(x, y)
			g.framebuffer.SetPixel(fbX+x, fbY+y, color)
		}
	}
}

// LCDC (LCD Control) Register bit values
// Bit 7 - LCD Display Enable (0=Off, 1=On)
// Bit 6 - Window Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
// Bit 5 - Window Display Enable (0=Off, 1=On)
// Bit 4 - BG & Window Tile Data Select (0=8800-97FF, 1=8000-8FFF)
// Bit 3 - BG Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
// Bit 2 - OBJ (Sprite) Size (0=8x8, 1=8x16)
// Bit 1 - OBJ (Sprite) Display Enable (0=Off, 1=On)
// Bit 0 - BG Display (0=Off, 1=On)

type lcdcFlag uint8

const (
	lcdDisplayEnable       lcdcFlag = 7
	windowTileMapSelect             = 6
	windowDisplayEnable             = 5
	bgWindowTileDataSelect          = 4
	bgTileMapDisplaySelect          = 3
	spriteSize                      = 2
	spriteDisplayEnable             = 1
	bgDisplay                       = 0
)

func (g *GPU) readLCDCVariable(flag lcdcFlag) byte {
	if bit.IsSet(uint8(flag), g.memory.Read(addr.LCDC)) {
		return 1
	}

	return 0
}

func (g *GPU) setLCDCVariable(flag lcdcFlag, shouldSet bool) {
	lcdcRegister := g.memory.Read(addr.LCDC)

	if shouldSet {
		lcdcRegister = bit.Set(uint8(flag), lcdcRegister)
	} else {
		lcdcRegister = bit.Clear(uint8(flag), lcdcRegister)
	}
}
