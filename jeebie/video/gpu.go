package video

import (
	"fmt"

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

// addresses for gpu registers
const (
	// LCD Control
	lcdcAddr uint16 = 0xFF40
	// LCDC Status
	statAddr uint16 = 0xFF41
	// Scroll Y (SCY)
	scyAddr uint16 = 0xFF42
	// Scroll X (SCX)
	scxAddr uint16 = 0xFF43
	// LCDC Y-Coordinate (readonly)
	lyAddr uint16 = 0xFF44
	// LY Compare
	lycAddr uint16 = 0xFF45
	// DMA Transfer and Start
	dmaAddr uint16 = 0xFF46
	// BG Palette
	bgpAddr uint16 = 0xFF47
	// Object Palette 0
	obp0Addr uint16 = 0xFF48
	// Object Palette 1
	obp1Addr uint16 = 0xFF49
	// Window Y Position
	wyAddr uint16 = 0xFF4A
	// Window X Position
	wxAddr uint16 = 0xFF4B
)

type GPU struct {
	memory      *memory.MMU
	screen      *Screen
	framebuffer *FrameBuffer

	line   uint8
	mode   GpuMode
	cycles int
}

func NewGpu(screen *Screen, memory *memory.MMU) *GPU {
	fb := NewFrameBuffer(160, 144)
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
	fmt.Printf("GPU TICK %v %v \n\n", cycles, g.cycles)
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
			g.cycles %= hblankCycles

			if g.line == 144 {
				g.mode = vblank
				// set vblank interrupt (bit 0)
				interruptFlags := g.memory.Read(0xFFFF)
				g.memory.Write(0xFFFF, bit.Set(0, interruptFlags))
				// g.drawFrame()
			} else {
				g.mode = oamRead
			}
		}
		break
	case vblank:
		if g.cycles >= scanlineCycles {
			g.line++
			g.cycles %= scanlineCycles

			if g.line == 154 {
				g.drawNoise()
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
	if bit.IsSet(uint8(flag), g.memory.Read(lcdcAddr)) {
		return 1
	}

	return 0
}

func (g *GPU) setLCDCVariable(flag lcdcFlag, shouldSet bool) {
	lcdcRegister := g.memory.Read(lcdcAddr)

	if shouldSet {
		lcdcRegister = bit.Set(uint8(flag), lcdcRegister)
	} else {
		lcdcRegister = bit.Clear(uint8(flag), lcdcRegister)
	}
}
