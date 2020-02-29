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
		g.cycles -= hblankCycles
		g.setMode(oamReadMode)
		g.setLY(g.line + 1)

		if g.line == 144 {
			g.setMode(vblankMode)
			g.vBlankLine = 0
			g.modeCounterAux = g.cycles
			g.windowLine = 0

			// Always trigger the VBlank interrupt when switching
			g.memory.RequestInterrupt(addr.VBlankInterrupt)

			// We're switching to VBlank Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			if g.memory.ReadBit(statVblankIrq, addr.STAT) {
				g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
			}
		} else if g.memory.ReadBit(statOamIrq, addr.STAT) {
			// We're switching to OAM Read Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
		}

		break
	case vblankMode:
		g.modeCounterAux += cycles

		if g.modeCounterAux >= scanlineCycles {
			g.modeCounterAux -= scanlineCycles
			g.vBlankLine++

			if g.vBlankLine <= 9 {
				g.setLY(g.line + 1)
			}
		}

		if g.cycles >= 4104 && g.modeCounterAux >= 4 && g.line == 153 {
			g.setLY(0)
		}

		if g.cycles >= 4560 {
			g.cycles -= 4560
			g.setMode(oamReadMode)
			// We're switching to OAM Read Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			if g.memory.ReadBit(statOamIrq, addr.STAT) {
				g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
			}
		}

		break
	case oamReadMode:
		if g.cycles >= oamScanlineCycles {
			g.cycles -= oamScanlineCycles
			g.setMode(vramReadMode)
			g.isScanLineTransfered = false
		}
		break
	case vramReadMode:
		if g.pixelCounter < 160 {
			g.tileCycleCounter += cycles

			if g.readLCDCVariable(lcdDisplayEnable) == 1 {
				for g.tileCycleCounter >= 3 {
					g.drawBackground()
					g.pixelCounter += 4
					g.tileCycleCounter -= 3

					if g.pixelCounter >= 160 {
						break
					}
				}
			}
		}

		if g.cycles >= 160 && !g.isScanLineTransfered {
			g.drawScanline()
			g.isScanLineTransfered = true
		}

		if g.cycles >= vramScanlineCycles {
			g.pixelCounter = 0
			g.cycles -= vramScanlineCycles
			g.tileCycleCounter = 0
			g.setMode(hblankMode)

			// We're switching to HBlank Mode
			// if enabled on STAT, trigger the LCDStat interrupt
			if g.memory.ReadBit(statHblankIrq, addr.STAT) {
				g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
			}
		}
		break
	}

	if g.cycles >= 70224 {
		g.cycles -= 70224
	}
}

func (g *GPU) drawScanline() {
	lcdEnabled := g.readLCDCVariable(lcdDisplayEnable) == 1
	if lcdEnabled {
		g.drawWindow()
		g.drawSprites()
		return
	}

	g.framebuffer.Clear()
}

func (g *GPU) drawBackground() {
	startXOffset := g.pixelCounter % 8
	endXOffset := startXOffset + 4
	screenTile := g.pixelCounter / 8
	lineWidth := g.line * width

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
	lineScrolled := g.line + int(scrollY)
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

func (g *GPU) drawWindow() {
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

	if wy > 143 || int(wy) > g.line {
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
	lineWidth := g.line * width

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
			if bit.IsSet(uint8(7-pixelX), low) {
				pixel |= 1
			}
			if bit.IsSet(uint8(7-pixelX), high) {
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

func (g *GPU) drawSprites() {
	if g.readLCDCVariable(spriteDisplayEnable) != 1 {
		return
	}

	spriteHeight := 8
	if g.readLCDCVariable(spriteSize) == 1 {
		spriteHeight = 16
	}

	lineWidth := g.line * width

	for sprite := 39; sprite >= 0; sprite-- {
		sprite4 := sprite * 4
		spriteY := int(g.memory.Read(0xFE00+uint16(sprite4))) - 16
		spriteX := int(g.memory.Read(0xFE00+uint16(sprite4)+1)) - 8

		if int(spriteY) > g.line || (int(spriteY)+spriteHeight) <= g.line {
			continue
		}

		if spriteX < -7 || spriteX >= width {
			continue
		}

		spriteTile := g.memory.Read(0xFE00 + uint16(sprite4) + 2)

		spriteMask := 0xFF
		if spriteHeight == 16 {
			spriteMask = 0xFE
		}

		spriteTile16 := (int(spriteTile) & spriteMask) * 16
		spriteFlags := g.memory.Read(0xFE00 + uint16(sprite4) + 3)
		objPaletteAddr := addr.OBP0
		if bit.IsSet(4, spriteFlags) {
			objPaletteAddr = addr.OBP1
		}

		flipX := bit.IsSet(5, spriteFlags)
		flipY := bit.IsSet(6, spriteFlags)
		aboveBG := !bit.IsSet(7, spriteFlags)

		tileAddr := 0x8000

		pixelY := g.line - spriteY
		if flipY {
			pixelY = spriteHeight - 1 - pixelY
		}

		pixelY2 := 0
		offset := 0

		if spriteHeight == 16 && pixelY >= 8 {
			pixelY2 = (pixelY - 8) * 2
			offset = 16
		} else {
			pixelY2 = pixelY * 2
		}

		tileAddr += spriteTile16 + pixelY2 + offset

		low := g.memory.Read(uint16(tileAddr))
		high := g.memory.Read(uint16(tileAddr) + 1)

		for pixelX := 0; pixelX < 8; pixelX++ {
			// if the flip flag is set, we render a mirrored sprite
			// i.e. we start from the other end (0 instead of 7)
			pixelIdx := 7 - pixelX
			if flipX {
				pixelIdx = pixelX
			}

			pixel := 0
			if bit.IsSet(uint8(pixelIdx), low) {
				pixel |= 1
			}
			if bit.IsSet(uint8(pixelIdx), high) {
				pixel |= 2
			}

			// transparent pixel
			if pixel == 0 {
				continue
			}

			bufferX := spriteX + pixelX
			if bufferX < 0 || bufferX >= width {
				continue
			}

			position := lineWidth + bufferX

			if !aboveBG {
				continue
			}

			palette := g.memory.Read(objPaletteAddr)
			color := (palette >> (pixel * 2)) & 0x03
			g.framebuffer.buffer[position] = uint32(ByteToColor(color))
		}
	}
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
			g.memory.RequestInterrupt(addr.LCDSTATInterrupt)
		}
	} else {
		stat = bit.Reset(statLycCondition, stat)
	}

	g.memory.Write(addr.STAT, stat)
}

// setMode sets the two bits (1,0) in the STAT register
// according to the selected GPU mode.
func (g *GPU) setMode(mode GpuMode) {
	g.mode = mode
	stat := g.memory.Read(addr.STAT)
	stat = stat&0xFC | byte(g.mode)
	g.memory.Write(addr.STAT, stat)
}

func (g *GPU) setLY(line int) {
	g.line = line
	g.memory.Write(addr.LY, byte(g.line))
	g.compareLYToLYC()
}
