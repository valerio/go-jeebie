package video

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

type GpuMode int

const (
	hblankMode GpuMode = iota
	vblankMode
	oamReadMode
	vramReadMode
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

	mode                 GpuMode
	line                 int
	cycles               int
	modeCounterAux       int
	vBlankLine           int
	pixelCounter         int
	tileCycleCounter     int
	isScanLineTransfered bool
	windowLine           int
	irqSignal            uint8
}

func NewGpu(screen *Screen, memory *memory.MMU) *GPU {
	fb := NewFrameBuffer()
	return &GPU{
		framebuffer: fb,
		screen:      screen,
		memory:      memory,
		mode:        vblankMode,

		line: 144,
	}
}

// Tick simulates gpu behaviour for a certain amount of clock cycles.
func (g *GPU) Tick(cycles int) {
	g.cycles += cycles

	switch g.mode {
	case hblankMode:
		if g.cycles < hblankCycles {
			break
		}

		g.cycles %= hblankCycles
		g.mode = oamReadMode

		g.line++
		g.memory.Write(addr.LY, byte(g.line))
		g.compareLYToLYC()

		if g.line == 144 {
			g.mode = vblankMode
			g.vBlankLine = 0
			g.modeCounterAux = g.cycles

			g.memory.RequestInterrupt(addr.VBlankInterrupt)

			// We're switching to VBlank Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			g.irqSignal &= 0x9
			if g.memory.ReadBit(statVblankIrq, addr.STAT) {
				if bit.IsSet(0, g.irqSignal) && !bit.IsSet(3, g.irqSignal) {
					g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
				}
				g.irqSignal = bit.Set(1, g.irqSignal)
			}
			g.irqSignal &= 0xE

			g.windowLine = 0
		} else {
			// We're switching to OAM Read Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			g.irqSignal &= 0x9
			if g.memory.ReadBit(statOamIrq, addr.STAT) {
				if bit.IsSet(0, g.irqSignal) && !bit.IsSet(3, g.irqSignal) {
					g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
				}
				g.irqSignal = bit.Set(2, g.irqSignal)
			}
			g.irqSignal &= 0xE

			g.updateMode()
		}

		break
	case vblankMode:
		g.modeCounterAux += cycles

		if g.cycles >= scanlineCycles {
			g.cycles %= scanlineCycles
			g.vBlankLine++

			if g.vBlankLine <= 9 {
				g.line++
				g.memory.Write(addr.LY, byte(g.line))
				g.compareLYToLYC()
			}
		}

		if g.cycles >= 4104 && g.modeCounterAux >= 4 && g.line == 153 {
			g.line = 0
			g.memory.Write(addr.LY, byte(g.line))
			g.compareLYToLYC()
		}

		if g.cycles >= 4560 {
			g.cycles %= 4560
			g.mode = oamReadMode
			g.updateMode()

			// We're switching to OAM Read Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			g.irqSignal &= 0x7
			g.irqSignal &= 0xA
			if g.memory.ReadBit(statOamIrq, addr.STAT) {
				if bit.IsSet(0, g.irqSignal) && !bit.IsSet(3, g.irqSignal) {
					g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
				}
				g.irqSignal = bit.Set(2, g.irqSignal)
			}
			g.irqSignal &= 0xD
		}

		break
	case oamReadMode:
		if g.cycles >= oamScanlineCycles {
			g.cycles %= oamScanlineCycles
			g.mode = vramReadMode
			g.updateMode()
			g.isScanLineTransfered = false
			g.irqSignal &= 0x8
		}
		break
	case vramReadMode:
		if g.pixelCounter < 160 {
			g.tileCycleCounter += cycles

			if g.readLCDCVariable(lcdDisplayEnable) == 1 {
				for g.tileCycleCounter >= 3 {
					g.drawBackground(g.line, g.pixelCounter, 4)

					g.pixelCounter += 4
					g.tileCycleCounter -= 3

					if g.pixelCounter >= 160 {
						break
					}
				}

			}
		}

		if g.cycles >= 160 && g.isScanLineTransfered {
			g.drawScanline(g.line)
			g.isScanLineTransfered = true
		}

		if g.cycles >= vramScanlineCycles {
			g.pixelCounter = 0
			g.cycles %= vramScanlineCycles
			g.mode = hblankMode
			g.tileCycleCounter = 0
			g.updateMode()

			// We're switching to HBlank Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			g.irqSignal &= 0x8
			if g.memory.ReadBit(statHblankIrq, addr.STAT) {
				if !bit.IsSet(3, g.irqSignal) {
					g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
				}
				g.irqSignal = bit.Set(0, g.irqSignal)
			}
		}
		break
	}
}

func (g *GPU) drawScanline(line int) {
	lcdEnabled := g.readLCDCVariable(lcdDisplayEnable) == 1
	if lcdEnabled {
		g.drawWindow(line)
		g.drawSprites(line)
		return
	}

	g.framebuffer.Clear()
}

func (g *GPU) drawBackground(line, pixel, count int) {
	// g.line, g.pixelCounter, 4
	startXOffset := pixel % 8
	endXOffset := startXOffset + count
	screenTile := pixel / 8
	lineWidth := line * width

	backgroundEnabled := g.readLCDCVariable(bgDisplay) == 1
	if !backgroundEnabled {
		// TODO: clear current line
		return
	}

	useTileSetZero := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	useTileMapZero := g.readLCDCVariable(bgTileMapDisplaySelect) == 0

	// select the correct starting address based on which tileMap/tileSet is set
	tilesAddr := uint16(0x8000)
	if useTileSetZero {
		tilesAddr = 0x8800
	}

	tileMapAddr := 0x9C00
	if useTileMapZero {
		tileMapAddr = 0x9800
	}

	scrollX := g.memory.Read(addr.SCX)
	scrollY := g.memory.Read(addr.SCY)
	lineScrolled := line + int(scrollY)
	lineScrolled32 := (lineScrolled / 8) * 32
	tilePixelY := lineScrolled % 8
	tilePixelY2 := tilePixelY * 2

	for xOffset := startXOffset; xOffset < endXOffset; xOffset++ {
		screenPixelX := (screenTile * 8) + xOffset
		mapPixelX := screenPixelX + int(scrollX)
		mapTileX := mapPixelX / 8
		mapTileXOffset := mapPixelX % 8
		mapTileAddr := uint16(tileMapAddr + lineScrolled32 + mapTileX)

		mapTile := 0

		if useTileSetZero {
			offset := int8(g.memory.Read(mapTileAddr))
			mapTile = int(offset) + 128
		} else {
			mapTile = int(g.memory.Read(mapTileAddr))
		}

		mapTile16 := mapTile * 16
		tileAddr := tilesAddr + uint16(mapTile16) + uint16(tilePixelY2)

		low := g.memory.Read(tileAddr)
		high := g.memory.Read(tileAddr + 1)

		pixelIndex := uint8(7 - mapTileXOffset)
		// the pixel is the bitwise OR of the low/high bit at
		// the current X index (from 7 to 0)
		pixel := 0
		if bit.IsSet(pixelIndex, low) {
			pixel |= 1
		}
		if bit.IsSet(pixelIndex, high) {
			pixel |= 2
		}

		pixelPosition := lineWidth + screenPixelX

		palette := g.memory.Read(addr.BGP)
		color := (palette >> (pixel * 2)) & 0x03
		g.framebuffer.buffer[pixelPosition] = uint32(ByteToColor(color))
	}
}

func (g *GPU) drawWindow(line int) {
	if g.windowLine > 143 {
		return
	}

	windowEnabled := g.readLCDCVariable(windowDisplayEnable) == 1
	if !windowEnabled {
		return
	}

	wx := g.memory.Read(addr.WX) - 7
	wy := g.memory.Read(addr.WY)

	if wx > 159 {
		return
	}

	if wy > 143 || int(wy) > line {
		return
	}

	useTileSetZero := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	useTileMapZero := g.readLCDCVariable(windowTileMapSelect) == 0

	// select the correct starting address based on which tileMap/tileSet is set
	tilesAddr := uint16(0x8000)
	if useTileSetZero {
		tilesAddr = 0x8800
	}

	tileMapAddr := 0x9C00
	if useTileMapZero {
		tileMapAddr = 0x9800
	}

	lineAdj := g.windowLine

	y32 := (lineAdj / 8) * 32
	pixelY := lineAdj & 8
	pixelY2 := pixelY * 2
	lineWidth := line * width

	for x := 0; x < 32; x++ {
		tileIndexAddr := uint16(tileMapAddr + y32 + x)
		tile := 0

		if useTileSetZero {
			offset := int8(g.memory.Read(tileIndexAddr))
			tile = int(offset) + 128
		} else {
			tile = int(g.memory.Read(tileIndexAddr))
		}

		xOffset := x * 8
		tile16 := tile * 16

		tileAddr := tilesAddr + uint16(tile16) + uint16(pixelY2)

		low := g.memory.Read(tileAddr)
		high := g.memory.Read(tileAddr + 1)

		for pixelX := 0; pixelX < 8; pixelX++ {
			bufferX := xOffset + pixelX + int(wx)

			if bufferX < 0 || bufferX > width {
				continue
			}

			// the pixel is the bitwise OR of the low/high bit at
			// the current X index (from 7 to 0)
			pixel := 0
			if bit.IsSet(7-uint8(pixelX), low) {
				pixel |= 1
			}
			if bit.IsSet(7-uint8(pixelX), high) {
				pixel |= 2
			}

			position := lineWidth + bufferX

			palette := g.memory.Read(addr.BGP)
			color := (palette >> (pixel * 2)) & 0x03
			g.framebuffer.buffer[position] = uint32(ByteToColor(color))
		}
	}
	g.windowLine++
}

func (g *GPU) drawSprites(line int) {
	// TODO: implement this
}

// LCD Stat (Status) Register bit values
// Bit 7 - unused
// Bit 6 - Interrupt based on LYC to LY comparison (based on bit 2)
// Bit 5 - Interrupt when Mode 10 (oamReadMode)
// Bit 4 - Interrupt when Mode 01 (vblankMode)
// Bit 3 - Interrupt when Mode 00 (hblankMode)
// Bit 2 - condition for triggering LYC/LY (0=LYC != LY, 1=LYC == LY)
// Bit 1,0 - represents the current GPU mode
//         - 00 -> hblankMode
//         - 01 -> vblankMode
//         - 10 -> oamReadMode
//         - 11 -> vramReadMode

type statFlag uint8

const (
	statLycIrq       statFlag = 6
	statOamIrq                = 5
	statVblankIrq             = 4
	statHblankIrq             = 3
	statLycCondition          = 2
	statModeHigh              = 1
	statModeLow               = 0
)

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

func (g *GPU) compareLYToLYC() {
	ly := g.memory.Read(addr.LY)
	lyc := g.memory.Read(addr.LYC)
	stat := g.memory.Read(addr.STAT)

	if ly == lyc {
		stat = bit.Set(statLycCondition, stat)
		if bit.IsSet(uint8(statLycIrq), stat) {
			if g.irqSignal == 0 {
				g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
			}
			g.irqSignal = bit.Set(3, g.irqSignal)
		}
	} else {
		stat = bit.Reset(statLycCondition, stat)
		g.irqSignal = bit.Reset(3, g.irqSignal)
	}

	g.memory.Write(addr.STAT, stat)
}

// updateMode sets the two bits (1,0) in the STAT register
// according to the current GPU mode.
func (g *GPU) updateMode() {
	stat := g.memory.Read(addr.STAT)
	stat = stat&0xFC | byte(g.mode)
	g.memory.Write(addr.STAT, stat)
}
