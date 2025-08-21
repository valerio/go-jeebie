package video

import (
	"fmt"
	"log/slog"

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
	memory        *memory.MMU
	framebuffer   *FrameBuffer
	bgPixelBuffer []byte

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

func NewGpu(memory *memory.MMU) *GPU {
	fb := NewFrameBuffer()
	gpu := &GPU{
		framebuffer:   fb,
		memory:        memory,
		mode:          vblankMode,
		bgPixelBuffer: make([]byte, FramebufferSize),

		line: 144,
	}

	// Log initial LCD state
	lcdc := memory.Read(0xFF40)
	bgp := memory.Read(0xFF47) // Background palette
	slog.Debug("GPU initialized", "LCDC", fmt.Sprintf("0x%02X", lcdc), "LCD_enabled", (lcdc&0x80) != 0, "BGP", fmt.Sprintf("0x%02X", bgp))

	return gpu
}

func (g *GPU) GetFrameBuffer() *FrameBuffer {
	return g.framebuffer
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
	case oamReadMode:
		if g.cycles >= oamScanlineCycles {
			g.cycles -= oamScanlineCycles
			g.setMode(vramReadMode)
			g.isScanLineTransfered = false
		}
	case vramReadMode:
		// Render the entire scanline once when entering VRAM mode
		if !g.isScanLineTransfered {
			if g.readLCDCVariable(lcdDisplayEnable) == 1 {
				g.drawScanline()
			}
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
	}

	if g.cycles >= 70224 {
		g.cycles -= 70224
	}
}

func (g *GPU) drawScanline() {
	lcdEnabled := g.readLCDCVariable(lcdDisplayEnable) == 1

	if !lcdEnabled {
		// Clear the current line when LCD is disabled
		lineWidth := g.line * FramebufferWidth
		for i := 0; i < FramebufferWidth; i++ {
			g.framebuffer.buffer[lineWidth+i] = 0xFFFFFFFF // White
		}
		return
	}

	// Draw all layers in correct order: Background -> Window -> Sprites
	g.drawBackground()
	g.drawWindow()
	g.drawSprites()
}

func (g *GPU) drawBackground() {
	lineWidth := g.line * FramebufferWidth
	backgroundEnabled := g.readLCDCVariable(bgDisplay) == 1

	if !backgroundEnabled {
		for i := range FramebufferWidth {
			g.framebuffer.buffer[lineWidth+i] = 0
			g.bgPixelBuffer[lineWidth+i] = 0
		}
		return
	}

	useTileSetZero := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	useTileMapZero := g.readLCDCVariable(bgTileMapDisplaySelect) == 0

	// select the correct starting address based on which tileMap/tileSet is set
	tilesAddr := uint16(0x8000)
	if useTileSetZero {
		tilesAddr = 0x9000
	}

	tileMapAddr := 0x9C00
	if useTileMapZero {
		tileMapAddr = 0x9800
	}

	scrollX := g.memory.Read(addr.SCX)
	scrollY := g.memory.Read(addr.SCY)
	lineScrolled := (g.line + int(scrollY)) & 0xFF // Y coordinate wraps at 256
	lineScrolled32 := (lineScrolled / 8) * 32
	tilePixelY := lineScrolled % 8
	tilePixelY2 := tilePixelY * 2

	// Render the entire scanline (160 pixels)
	for screenPixelX := 0; screenPixelX < FramebufferWidth; screenPixelX++ {
		mapPixelX := (screenPixelX + int(scrollX)) & 0xFF
		mapTileX := mapPixelX / 8
		mapTileXOffset := mapPixelX % 8
		mapTileAddr := uint16(tileMapAddr + lineScrolled32 + mapTileX)

		mapTileValue := g.memory.Read(mapTileAddr)

		var tileAddr uint16
		if useTileSetZero {
			// signed addressing: base 0x9000, tile numbers -128 to 127
			signedTile := int8(mapTileValue)
			tileOffset := int(signedTile) * 16
			tileAddr = uint16(int(tilesAddr) + tileOffset + int(tilePixelY2))
		} else {
			// unsigned addressing: base 0x8000, tile numbers 0 to 255
			mapTile := int(mapTileValue)
			mapTile16 := mapTile * 16
			tileAddr = tilesAddr + uint16(mapTile16) + uint16(tilePixelY2)
		}

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
		finalColor := uint32(ByteToColor(color))

		g.framebuffer.buffer[pixelPosition] = finalColor
		g.bgPixelBuffer[pixelPosition] = color // just use the color value (0-3) for the buffer
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

	// Debug window rendering
	if g.line < 5 { // Only log first few lines to avoid spam
		slog.Debug("Window rendering", "line", g.line, "windowLine", g.windowLine, "wx", wx, "wy", wy)
	}

	useTileSetZero := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	useTileMapZero := g.readLCDCVariable(windowTileMapSelect) == 0

	// select the correct starting address based on which tileMap/tileSet is set
	tilesAddr := uint16(0x8000)
	if useTileSetZero {
		tilesAddr = 0x9000 // Signed mode uses base 0x9000
	}

	tileMapAddr := 0x9C00
	if useTileMapZero {
		tileMapAddr = 0x9800
	}

	lineAdj := g.windowLine

	y32 := (lineAdj / 8) * 32
	pixelY := lineAdj & 7
	pixelY2 := pixelY * 2
	lineWidth := g.line * FramebufferWidth

	// Only render tiles where the window is actually visible
	startTileX := 0
	if wx > 0 {
		startTileX = 0 // Window starts from tile 0 in window space
	}
	endTileX := (FramebufferWidth - int(wx) + 7) / 8 // Calculate how many tiles are visible
	if endTileX > 32 {
		endTileX = 32
	}

	for x := startTileX; x < endTileX; x++ {
		tileIndexAddr := uint16(tileMapAddr + y32 + x)
		tileValue := g.memory.Read(tileIndexAddr)
		xOffset := x * 8

		var tileAddr uint16
		if useTileSetZero {
			// signed addressing: base 0x9000, tile numbers -128 to 127
			signedTile := int8(tileValue)
			tileOffset := int(signedTile) * 16
			tileAddr = uint16(int(tilesAddr) + tileOffset + int(pixelY2))
		} else {
			// unsigned addressing: base 0x8000, tile numbers 0 to 255
			tile := int(tileValue)
			tile16 := tile * 16
			tileAddr = tilesAddr + uint16(tile16) + uint16(pixelY2)
		}

		low := g.memory.Read(tileAddr)
		high := g.memory.Read(tileAddr + 1)

		for pixelX := 0; pixelX < 8; pixelX++ {
			bufferX := xOffset + pixelX + int(wx)

			// Only draw pixels that are within the window area and on screen
			if bufferX < int(wx) || bufferX >= FramebufferWidth {
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

			// Safety check to prevent buffer overflow
			if position >= len(g.framebuffer.buffer) {
				continue
			}

			palette := g.memory.Read(addr.BGP)
			color := (palette >> (pixel * 2)) & 0x03
			g.framebuffer.buffer[position] = uint32(ByteToColor(color))
			g.bgPixelBuffer[position] = color
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

	lineWidth := g.line * FramebufferWidth
	var spritesToDraw []int

	// Handle "selection priority": there's a maximum of 10 sprites per scanline, but 2 priorities
	// we have to respect. First, we scan the OAM in ascending order to pick up the first 10 visible
	// objects/sprites, these will be the ones we'll draw.
	for sprite := 0; sprite < 40; sprite++ {
		sprite4 := sprite * 4
		spriteY := int(g.memory.Read(0xFE00+uint16(sprite4))) - 16
		spriteX := int(g.memory.Read(0xFE00+uint16(sprite4)+1)) - 8

		if spriteY > g.line || (spriteY+spriteHeight) <= g.line {
			continue
		}

		if spriteX < -7 || spriteX >= FramebufferWidth {
			continue
		}

		spritesToDraw = append(spritesToDraw, sprite)

		// Hit the limit, stop collecting and move on to drawing
		if len(spritesToDraw) >= 10 {
			break
		}
	}

	// Handle "drawing priority": we have a list of up to 10 objects/sprites here, but now their
	// priority is different, we go in *descending* order and draw sequentially.
	// in essence, a lower-indexed sprite gets drawn on top of others.
	for i := len(spritesToDraw) - 1; i >= 0; i-- {
		sprite := spritesToDraw[i]
		sprite4 := sprite * 4
		spriteY := int(g.memory.Read(0xFE00+uint16(sprite4))) - 16
		spriteX := int(g.memory.Read(0xFE00+uint16(sprite4)+1)) - 8

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
			if bufferX < 0 || bufferX >= FramebufferWidth {
				continue
			}

			position := lineWidth + bufferX

			// Safety check to prevent buffer overflow
			if position >= len(g.framebuffer.buffer) {
				continue
			}

			// Sprite priority: if !aboveBG, sprite is behind background, so we should stop
			// drawing it. If the background color is 0, we must draw it.
			if !aboveBG {
				// check the raw BG pixels which we stored so far.
				bgPixel := g.bgPixelBuffer[position]
				if bgPixel != 0 {
					// sprite pixel is 1-2-3, should not be drawn.
					continue
				}
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
//   - 00 -> hblankMode
//   - 01 -> vblankMode
//   - 10 -> oamReadMode
//   - 11 -> vramReadMode
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
