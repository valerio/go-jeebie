package video

import (
	"math/rand"

	"github.com/valep27/go-jeebie/jeebie/memory"
	"github.com/valep27/go-jeebie/jeebie/util"
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
const lcdcAddress = 0xFF40

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
				interruptFlags := g.memory.ReadByte(0xFFFF)
				g.memory.WriteByte(0xFFFF, util.SetBit(0, interruptFlags))
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
				g.drawScanLine()
				g.screen.Draw(g.framebuffer.ToSlice())
				g.line = 0
				g.mode = oamRead
			}
		}
		break
	}
}

func (g *GPU) drawScanLine() {
	// placeholder: draws random pixels
	for i := 0; i < len(g.framebuffer.buffer); i++ {

		var color GBColor
		switch rand.Uint32() % 4 {
		case 0:
			color = WhiteColor
			break
		case 1:
			color = BlackColor
			break
		case 2:
			color = LightGreyColor
			break
		case 3:
			color = DarkGreyColor
			break
		default:
			color = BlackColor
		}

		g.framebuffer.buffer[i] = uint32(color)
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

type lcdcBit uint8

const (
	lcdDisplayEnable lcdcBit = iota
	windowTileMapSelect
	windowDisplayEnable
	bgWindowTileDataSelect
	bgTileMapDisplaySelect
	spriteSize
	spriteDisplayEnable
	bgDisplay
)

func (g *GPU) readLCDCVariable(bit lcdcBit) byte {
	if util.IsBitSet(uint8(bit), g.memory.ReadByte(lcdcAddress)) {
		return 1
	}

	return 0
}

func (g *GPU) setLCDCVariable(bit lcdcBit, shouldSet bool) {
	lcdcRegister := g.memory.ReadByte(lcdcAddress)

	if shouldSet {
		lcdcRegister = util.SetBit(uint8(bit), lcdcRegister)
	} else {
		lcdcRegister = util.ClearBit(uint8(bit), lcdcRegister)
	}
}
