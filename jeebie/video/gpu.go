package video

import (
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

type GPU struct {
	memory *memory.MMU

	line   uint8
	mode   GpuMode
	cycles uint
}

func NewGpu() *GPU {
	return &GPU{}
}

// Tick simulates gpu behaviour for a certain amount of clock cycles.
func (g *GPU) Tick(cycles uint) {
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
				g.line = 0
				g.mode = oamRead
			}
		}
		break
	}
}

func (g *GPU) drawScanLine() {

}
