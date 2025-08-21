package video

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

// GpuMode represents the PPU's current rendering stage.
// These values match the STAT register bits 1-0.
type GpuMode int

const (
	// hblankMode (Mode 0): Horizontal blank period, CPU can access VRAM/OAM
	hblankMode GpuMode = 0
	// vblankMode (Mode 1): Vertical blank period, CPU can access VRAM/OAM
	vblankMode GpuMode = 1
	// oamReadMode (Mode 2): PPU is reading OAM, CPU cannot access OAM
	oamReadMode GpuMode = 2
	// vramReadMode (Mode 3): PPU is reading VRAM, CPU cannot access VRAM/OAM
	vramReadMode GpuMode = 3
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
	bgPixelBuffer []byte // stores background/window pixel colors for sprite priority
	oam           *OAM   // OAM scanner for sprite management

	// PPU state - these map to Game Boy hardware registers/behavior
	mode                 GpuMode // current PPU mode (matches STAT bits 1-0)
	line                 int     // current scanline (LY register, 0-153)
	cycles               int     // cycle counter for current mode
	modeCounterAux       int     // auxiliary counter for VBlank timing
	vBlankLine           int     // which VBlank line we're on (0-9)
	pixelCounter         int     // pixel counter within scanline
	tileCycleCounter     int     // cycle counter for tile fetching
	isScanLineTransfered bool    // whether current scanline has been rendered
	windowLine           int     // internal window line counter (0-143)
}

func NewGpu(memory *memory.MMU) *GPU {
	fb := NewFrameBuffer()
	gpu := &GPU{
		framebuffer:   fb,
		memory:        memory,
		mode:          vblankMode,
		bgPixelBuffer: make([]byte, FramebufferSize),
		oam:           NewOAM(memory),

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
	if !g.isBackgroundEnabled() {
		g.clearBackground()
		return
	}

	g.renderBackgroundTiles()
}

func (g *GPU) isBackgroundEnabled() bool {
	return g.readLCDCVariable(bgDisplay) == 1
}

// call when background is disabled, fill with color 0 from BGP
func (g *GPU) clearBackground() {
	lineWidth := g.line * FramebufferWidth
	palette := g.memory.Read(addr.BGP)
	color0 := palette & 0x03
	displayColor := uint32(ByteToColor(color0))

	for i := range FramebufferWidth {
		g.framebuffer.buffer[lineWidth+i] = displayColor
		g.bgPixelBuffer[lineWidth+i] = 0
	}
}

func (g *GPU) renderBackgroundTiles() {
	lineWidth := g.line * FramebufferWidth
	tileDataAddr := g.getTileDataAddress()
	tileMapAddr := g.getBackgroundTileMapAddress()
	useSigned := g.readLCDCVariable(bgWindowTileDataSelect) == 0
	scrollX, scrollY := g.getBackgroundScroll()
	scrolledY := (g.line + scrollY) & 0xFF

	// render each pixel of the scanline
	for screenX := range FramebufferWidth {
		// calculate position in background map (with wrapping)
		bgX := (screenX + scrollX) & 0xFF
		bgY := scrolledY

		pixelColor := g.fetchBackgroundPixel(bgX, bgY, tileDataAddr, tileMapAddr, useSigned)

		position := lineWidth + screenX
		g.drawBackgroundPixel(position, pixelColor)
	}
}

func (g *GPU) getTileDataAddress() uint16 {
	if g.readLCDCVariable(bgWindowTileDataSelect) == 0 {
		return addr.TileData2 // signed addressing mode (0x8800-0x97FF)
	}
	return addr.TileData0 // unsigned addressing mode (0x8000-0x8FFF)
}

func (g *GPU) getBackgroundTileMapAddress() uint16 {
	if g.readLCDCVariable(bgTileMapDisplaySelect) == 0 {
		return addr.TileMap0
	}
	return addr.TileMap1
}

func (g *GPU) getBackgroundScroll() (x, y int) {
	return int(g.memory.Read(addr.SCX)), int(g.memory.Read(addr.SCY))
}

func (g *GPU) fetchBackgroundPixel(bgX, bgY int, tileDataAddr, tileMapAddr uint16, useSigned bool) int {
	// calculate tile coordinates
	tileX := bgX / 8
	tileY := bgY / 8
	pixelXInTile := bgX % 8
	pixelYInTile := bgY % 8

	// fetch tile index from tilemap
	tileMapOffset := tileY*32 + tileX
	tileIndex := g.memory.Read(tileMapAddr + uint16(tileMapOffset))

	tileRow := g.fetchTileRow(tileIndex, pixelYInTile, tileDataAddr, useSigned)
	return tileRow.GetPixel(pixelXInTile)
}

func (g *GPU) fetchTileRow(tileIndex byte, row int, baseAddr uint16, signed bool) TileRow {
	var tileAddr uint16

	if signed {
		// signed addressing: interpret as -128 to 127
		signedIndex := int8(tileIndex)
		tileAddr = uint16(int(baseAddr) + int(signedIndex)*16 + row*2)
	} else {
		// unsigned addressing: 0 to 255
		tileAddr = baseAddr + uint16(tileIndex)*16 + uint16(row*2)
	}

	return TileRow{
		Low:  g.memory.Read(tileAddr),
		High: g.memory.Read(tileAddr + 1),
	}
}

func (g *GPU) drawBackgroundPixel(position int, pixelColor int) {
	palette := g.memory.Read(addr.BGP)
	paletteColor := (palette >> (pixelColor * 2)) & 0x03

	g.framebuffer.buffer[position] = uint32(ByteToColor(paletteColor))
	g.bgPixelBuffer[position] = paletteColor
}

func (g *GPU) drawWindow() {
	if !g.shouldRenderWindow() {
		return
	}

	g.renderWindowTiles()
	g.windowLine++
}

func (g *GPU) shouldRenderWindow() bool {
	// check if window should be rendered on this scanline
	if g.windowLine > 143 {
		return false
	}

	if g.readLCDCVariable(windowDisplayEnable) != 1 {
		return false
	}

	wy := g.memory.Read(addr.WY)
	if wy > 143 || int(wy) > g.line {
		return false
	}

	wx := g.memory.Read(addr.WX)
	if wx > 166 { // WX - 7 > 159
		return false
	}

	return true
}

func (g *GPU) renderWindowTiles() {
	lineWidth := g.line * FramebufferWidth

	// get window position
	wx := int(g.memory.Read(addr.WX)) - 7

	// determine tile data and tilemap sources
	tileDataAddr := g.getTileDataAddress()
	tileMapAddr := g.getWindowTileMapAddress()
	useSigned := g.readLCDCVariable(bgWindowTileDataSelect) == 0

	// calculate visible tile range
	startX := wx
	if startX < 0 {
		startX = 0
	}

	// render each visible pixel of the window
	for screenX := startX; screenX < FramebufferWidth; screenX++ {
		// calculate window-relative coordinates
		windowX := screenX - wx
		windowY := g.windowLine

		// fetch and decode the pixel
		pixelColor := g.fetchWindowPixel(windowX, windowY, tileDataAddr, tileMapAddr, useSigned)

		// apply palette and write to buffers
		position := lineWidth + screenX
		g.drawWindowPixel(position, pixelColor)
	}
}

func (g *GPU) getWindowTileMapAddress() uint16 {
	if g.readLCDCVariable(windowTileMapSelect) == 0 {
		return addr.TileMap0 // 0x9800-0x9BFF
	}
	return addr.TileMap1 // 0x9C00-0x9FFF
}

func (g *GPU) fetchWindowPixel(windowX, windowY int, tileDataAddr, tileMapAddr uint16, useSigned bool) int {
	// calculate tile coordinates within window
	tileX := windowX / 8
	tileY := windowY / 8
	pixelXInTile := windowX % 8
	pixelYInTile := windowY % 8

	// fetch tile index from tilemap
	tileMapOffset := tileY*32 + tileX
	tileIndex := g.memory.Read(tileMapAddr + uint16(tileMapOffset))

	// fetch tile data (reuse background tile fetching)
	tileRow := g.fetchTileRow(tileIndex, pixelYInTile, tileDataAddr, useSigned)

	// extract pixel from tile row
	return tileRow.GetPixel(pixelXInTile)
}

func (g *GPU) drawWindowPixel(position int, pixelColor int) {
	// window uses same palette as background
	palette := g.memory.Read(addr.BGP)
	paletteColor := (palette >> (pixelColor * 2)) & 0x03

	g.framebuffer.buffer[position] = uint32(ByteToColor(paletteColor))
	g.bgPixelBuffer[position] = paletteColor
}

func (g *GPU) drawSprites() {
	if g.readLCDCVariable(spriteDisplayEnable) != 1 {
		return
	}

	sprites := g.oam.GetSpritesForScanline(g.line)
	for _, sprite := range sprites {
		if !sprite.HasPriorityForAnyPixel() {
			continue // no priority = no rendery
		}
		g.renderSprite(sprite)
	}
}

func (g *GPU) renderSprite(sprite Sprite) {
	lineWidth := g.line * FramebufferWidth
	tileRow := g.fetchSpriteTileRow(sprite)
	paletteAddr := g.getSpritePaletteAddress(sprite)

	// render each pixel of the sprite
	for pixelX := range 8 {
		if !sprite.HasPriorityForPixel(pixelX) {
			continue
		}

		bufferX := int(sprite.X) + pixelX
		if bufferX < 0 || bufferX >= FramebufferWidth {
			continue // out of screen bounds
		}

		var pixelColor int
		if sprite.FlipX {
			pixelColor = tileRow.GetPixelFlipped(pixelX)
		} else {
			pixelColor = tileRow.GetPixel(pixelX)
		}

		if pixelColor == 0 {
			continue // transparent
		}

		// check background priority
		position := lineWidth + bufferX
		if sprite.BehindBG && g.bgPixelBuffer[position] != 0 {
			continue
		}

		g.drawSpritePixel(position, pixelColor, paletteAddr)
	}
}

func (g *GPU) fetchSpriteTileRow(sprite Sprite) TileRow {
	// calculate which row of the tile we're rendering
	pixelY := g.line - int(sprite.Y)
	if sprite.FlipY {
		pixelY = sprite.Height - 1 - pixelY
	}

	// calculate tile address
	tileIndex := sprite.TileIndex
	if sprite.Height == 16 {
		tileIndex &= 0xFE // ignore bit 0 for 8x16 sprites
	}

	// calculate the offset within the tile
	var tileRowOffset int
	if sprite.Height == 16 && pixelY >= 8 {
		tileRowOffset = (pixelY-8)*2 + 16 // second tile
	} else {
		tileRowOffset = pixelY * 2
	}

	// sprites always use unsigned addressing from 0x8000
	tileAddr := addr.TileData0 + uint16(int(tileIndex)*16+tileRowOffset)

	return TileRow{
		Low:  g.memory.Read(tileAddr),
		High: g.memory.Read(tileAddr + 1),
	}
}

func (g *GPU) getSpritePaletteAddress(sprite Sprite) uint16 {
	if sprite.PaletteOBP1 {
		return addr.OBP1
	}
	return addr.OBP0
}

func (g *GPU) drawSpritePixel(position int, pixelColor int, paletteAddr uint16) {
	palette := g.memory.Read(paletteAddr)
	color := (palette >> (pixelColor * 2)) & 0x03
	g.framebuffer.buffer[position] = uint32(ByteToColor(color))
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

// setLY updates the current scanline (LY register).
// This also triggers interrupts if necessary (LY/LYC comparison)
func (g *GPU) setLY(line int) {
	g.line = line
	g.memory.Write(addr.LY, byte(g.line))
	g.compareLYToLYC()
}
