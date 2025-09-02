//go:build sdl2

package sdl2

import (
	"fmt"
	"unsafe"

	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	DebugWindowWidth  = 1280
	DebugWindowHeight = 800
	maxDisasmLines    = 20
	spriteScale       = 2
)

type DebugWindow struct {
	window   *sdl.Window
	renderer *sdl.Renderer
	visible  bool

	spriteTexture *sdl.Texture
	bgTexture     *sdl.Texture

	// Cached visualizers to avoid allocations
	cachedSpriteVis debug.SpriteVisualizer
	cachedBgVis     debug.BackgroundVisualizer

	// Pointers to current data
	debugData    *debug.Data // Full debug data for disassembly
	spriteVis    *debug.SpriteVisualizer
	bgVis        *debug.BackgroundVisualizer
	paletteVis   *debug.PaletteVisualizer
	audioData    *debug.AudioData
	layerBuffers *video.RenderLayers

	// Waveform visualization
	waveformSamples [5][128]float32 // Ch1-4 + Mix

	// Pre-allocated buffers to avoid allocations in hot loops
	tilemapPixelBuffer []byte              // 256*256*4 bytes for tilemap rendering
	spriteTileBuffer   []uint32            // 8*8 buffer for sprite tile rendering
	defaultPalette     []uint32            // Default grayscale palette
	disasmBuffer       *debug.DisasmBuffer // Pre-allocated disassembly buffer

	// Cached formatted strings to avoid sprintf on every frame
	cachedDisasmLines []string // Cached disassembly text
	cachedPC          uint16   // PC value when cache was created
	disasmCacheValid  bool     // Whether cached disasm is still valid

	needsUpdate bool
}

func NewDebugWindow() *DebugWindow {
	return &DebugWindow{
		visible:     false,
		needsUpdate: true,
	}
}

func (dw *DebugWindow) Init() error {
	window, err := sdl.CreateWindow(
		"Game Boy Debug",
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		DebugWindowWidth,
		DebugWindowHeight,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		return err
	}
	dw.window = window

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		window.Destroy()
		return err
	}
	dw.renderer = renderer

	dw.spriteTexture, err = renderer.CreateTexture(
		sdl.PIXELFORMAT_RGBA8888,
		sdl.TEXTUREACCESS_STREAMING,
		40*16, 16,
	)
	if err != nil {
		return err
	}

	dw.bgTexture, err = renderer.CreateTexture(
		sdl.PIXELFORMAT_RGBA8888,
		sdl.TEXTUREACCESS_STREAMING,
		256, 256,
	)
	if err != nil {
		return err
	}

	// Pre-allocate pixel buffers to avoid allocations in hot loops
	dw.tilemapPixelBuffer = make([]byte, 256*256*4)
	dw.spriteTileBuffer = make([]uint32, 8*8)
	dw.defaultPalette = []uint32{
		uint32(video.WhiteColor),
		uint32(video.LightGreyColor),
		uint32(video.DarkGreyColor),
		uint32(video.BlackColor),
	}
	dw.disasmBuffer = debug.NewDisasmBuffer(maxDisasmLines)

	dw.window.Hide()
	return nil
}

func (dw *DebugWindow) UpdateData(debugData *debug.Data) {
	if debugData == nil {
		return
	}

	// Invalidate disasm cache if PC changed
	if dw.debugData != nil && dw.debugData.CPU != nil &&
		debugData.CPU != nil && dw.debugData.CPU.PC != debugData.CPU.PC {
		dw.disasmCacheValid = false
	}

	dw.debugData = debugData
	dw.spriteVis = debugData.SpriteVis
	dw.bgVis = debugData.BackgroundVis
	dw.paletteVis = debugData.PaletteVis
	dw.audioData = debugData.Audio
	dw.layerBuffers = debugData.LayerBuffers
	if dw.audioData != nil {
		dw.updateWaveformSamples()
	}
	dw.needsUpdate = true
}

func (dw *DebugWindow) Render() error {
	if !dw.visible || !dw.needsUpdate {
		return nil
	}

	dw.renderer.SetDrawColor(30, 30, 30, 255)
	dw.renderer.Clear()

	dw.renderSpritePanel()
	dw.renderBackgroundPanel()
	dw.renderPalettePanel()
	dw.renderDisassemblyPanel()

	if dw.audioData != nil {
		dw.renderAudioPanel()
		dw.renderWaveforms()
	}

	dw.renderer.Present()
	dw.needsUpdate = false
	return nil
}

func (dw *DebugWindow) renderSpritePanel() {
	dw.renderPanelLabel(10, 10, "Sprites (OAM)")

	panelRect := &sdl.Rect{10, 35, 620, 300}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.spriteVis == nil {
		return
	}

	// Show all 40 sprites in a 3-column layout
	sprites := dw.spriteVis.Sprites
	const spritesPerColumn = 14
	const columnWidth = 200
	const rowHeight = 20

	for i := 0; i < len(sprites) && i < 40; i++ {
		sprite := sprites[i]

		// Calculate position (2 columns)
		column := i / spritesPerColumn
		row := i % spritesPerColumn
		x := int32(20 + column*columnWidth)
		y := int32(45 + row*rowHeight)

		// Render the small sprite tile
		dw.renderSmallSpriteTile(sprite.TileData, x, y)

		// Determine text color based on visibility
		textR, textG, textB := uint8(200), uint8(200), uint8(200)
		if !sprite.Info.IsVisible {
			textR, textG, textB = 100, 100, 100
		}

		// Compact info: index, tile, position
		info := fmt.Sprintf("%02d:%02X (%3d,%3d)",
			sprite.Info.Index,
			sprite.Info.Sprite.TileIndex,
			sprite.X,
			sprite.Y,
		)

		DrawText(dw.renderer, info, x+20, y+5, 1, textR, textG, textB)

		// Show flags as single letters
		flagX := x + 140
		if sprite.Info.Sprite.FlipX {
			DrawText(dw.renderer, "X", flagX, y+5, 1, 255, 150, 150)
			flagX += 8
		}
		if sprite.Info.Sprite.FlipY {
			DrawText(dw.renderer, "Y", flagX, y+5, 1, 150, 255, 150)
			flagX += 8
		}
		if sprite.Info.Sprite.BehindBG {
			DrawText(dw.renderer, "B", flagX, y+5, 1, 150, 150, 255)
			flagX += 8
		}
		if sprite.Info.Sprite.PaletteOBP1 {
			DrawText(dw.renderer, "1", flagX, y+5, 1, 255, 255, 150)
		} else {
			DrawText(dw.renderer, "0", flagX, y+5, 1, 200, 200, 200)
		}
	}

	// Legend at bottom
	legendY := int32(45 + spritesPerColumn*rowHeight + 5)
	DrawText(dw.renderer, "Format: ID:Tile (X,Y) | Flags: X=FlipX Y=FlipY B=BG 0/1=Palette",
		20, legendY, 1, 150, 150, 150)
}

func (dw *DebugWindow) renderBackgroundPanel() {
	dw.renderPanelLabel(650, 10, "Background Tilemap")

	// Changed to 256x256 for pixel-perfect display
	panelRect := &sdl.Rect{650, 35, 280, 280}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.bgVis == nil || !dw.bgVis.BGEnabled {
		DrawText(dw.renderer, "Background Disabled", 500, 180, 2, 100, 100, 100)
		return
	}

	dw.renderTilemap()

	infoY := int32(320) // Adjusted for smaller tilemap display
	winStatus := "OFF"
	if dw.bgVis.WindowEnabled {
		winStatus = fmt.Sprintf("ON (X:%d Y:%d)", dw.bgVis.WindowX, dw.bgVis.WindowY)
	}
	info := fmt.Sprintf("SCX:%d SCY:%d | Win: %s",
		dw.bgVis.ScrollX, dw.bgVis.ScrollY, winStatus,
	)
	DrawText(dw.renderer, info, 660, infoY, 1, 200, 200, 200)

	// Show tilemap addresses
	bgMapAddr := "9800"
	if dw.bgVis.TilemapBase == 0x9C00 {
		bgMapAddr = "9C00"
	}
	winMapAddr := "9800"
	if dw.bgVis.WindowTilemapBase == 0x9C00 {
		winMapAddr = "9C00"
	}
	tileDataAddr := "8000"
	if dw.bgVis.TileDataBase == 0x8800 {
		tileDataAddr = "8800"
	}
	mapInfo := fmt.Sprintf("BG Map:%s Win Map:%s Tiles:%s",
		bgMapAddr, winMapAddr, tileDataAddr,
	)
	DrawText(dw.renderer, mapInfo, 660, infoY+15, 1, 150, 150, 150)
}

func (dw *DebugWindow) renderTilemap() {
	// Convert uint32 buffer to byte buffer for SDL texture
	for i, pixel := range dw.layerBuffers.Background.Buffer {
		offset := i * 4
		if offset+3 < len(dw.tilemapPixelBuffer) {
			// SDL2 RGBA8888 format expects ABGR in memory
			dw.tilemapPixelBuffer[offset] = byte(pixel)         // Alpha (from AA)
			dw.tilemapPixelBuffer[offset+1] = byte(pixel >> 8)  // Blue (from BB)
			dw.tilemapPixelBuffer[offset+2] = byte(pixel >> 16) // Green (from GG)
			dw.tilemapPixelBuffer[offset+3] = byte(pixel >> 24) // Red (from RR)
		}
	}

	dw.bgTexture.Update(nil, unsafe.Pointer(&dw.tilemapPixelBuffer[0]), 256*4)

	srcRect := &sdl.Rect{0, 0, 256, 256}
	dstRect := &sdl.Rect{660, 45, 256, 256} // Pixel-perfect 1:1 display
	dw.renderer.Copy(dw.bgTexture, srcRect, dstRect)
}

func (dw *DebugWindow) renderPalettePanel() {
	dw.renderPanelLabel(990, 10, "Palettes")

	panelRect := &sdl.Rect{990, 35, 280, 130}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.paletteVis == nil {
		return
	}

	palettes := []struct {
		name string
		info debug.PaletteInfo
	}{
		{"BGP ", dw.paletteVis.BGP},
		{"OBP0", dw.paletteVis.OBP0},
		{"OBP1", dw.paletteVis.OBP1},
	}

	for i, pal := range palettes {
		y := int32(45 + i*35)
		x := int32(1000)

		DrawText(dw.renderer, pal.name, x, y, 1, 200, 200, 200)

		for j := 0; j < 4; j++ {
			colorX := x + 40 + int32(j*30)

			// Convert GBColor to RGBA using video package constants
			var rgba uint32
			switch pal.info.Colors[j] {
			case 0:
				rgba = uint32(video.WhiteColor)
			case 1:
				rgba = uint32(video.LightGreyColor)
			case 2:
				rgba = uint32(video.DarkGreyColor)
			case 3:
				rgba = uint32(video.BlackColor)
			default:
				rgba = 0xFFFF00FF // Error color (magenta)
			}

			// Extract RGBA components (format is 0xAABBGGRR)
			r := uint8(rgba >> 24)
			g := uint8(rgba >> 16)
			b := uint8(rgba >> 8)

			dw.renderer.SetDrawColor(r, g, b, 255)
			colorRect := &sdl.Rect{colorX, y, 25, 25}
			dw.renderer.FillRect(colorRect)

			dw.renderer.SetDrawColor(200, 200, 200, 255)
			dw.renderer.DrawRect(colorRect)
		}

		rawStr := fmt.Sprintf("0x%02X", pal.info.Raw)
		DrawText(dw.renderer, rawStr, x+170, y+8, 1, 150, 150, 150)
	}
}

func (dw *DebugWindow) renderDisassemblyPanel() {
	dw.renderPanelLabel(10, 350, "Disassembly")

	panelRect := &sdl.Rect{10, 375, 620, 410}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.debugData == nil || dw.debugData.CPU == nil || dw.debugData.Memory == nil {
		DrawText(dw.renderer, "No debug data available", 20, 390, 1, 100, 100, 100)
		return
	}

	pc := dw.debugData.CPU.PC

	// Only update disassembly if PC changed or cache is invalid
	if !dw.disasmCacheValid || dw.cachedPC != pc {
		disasmLines := debug.CreateDisassemblyWithBuffer(dw.debugData.Memory, pc, maxDisasmLines, dw.disasmBuffer)

		// Clear and rebuild cache
		if cap(dw.cachedDisasmLines) < len(disasmLines)*2 {
			dw.cachedDisasmLines = make([]string, 0, len(disasmLines)*2)
		} else {
			dw.cachedDisasmLines = dw.cachedDisasmLines[:0]
		}

		// Cache formatted strings
		for _, line := range disasmLines {
			if line.IsCurrent {
				dw.cachedDisasmLines = append(dw.cachedDisasmLines, "current")
			} else {
				dw.cachedDisasmLines = append(dw.cachedDisasmLines, "")
			}
			text := fmt.Sprintf("%04X: %s", line.Address, line.Instruction)
			dw.cachedDisasmLines = append(dw.cachedDisasmLines, text)
		}

		dw.cachedPC = pc
		dw.disasmCacheValid = true
	}

	// Render cached lines
	y := int32(385)
	lineHeight := int32(16)

	for i := 0; i < len(dw.cachedDisasmLines); i += 2 {
		if y+lineHeight > 750 { // Leave space for status line
			break
		}

		var r, g, b uint8
		if dw.cachedDisasmLines[i] == "current" {
			// Current instruction - bright yellow
			r, g, b = 255, 255, 100
			DrawText(dw.renderer, ">", 15, y, 1, 255, 255, 100)
		} else {
			// Normal instruction - light gray
			r, g, b = 180, 180, 180
		}
		DrawText(dw.renderer, dw.cachedDisasmLines[i+1], 30, y, 1, r, g, b)
		y += lineHeight
	}

	// Draw status line at bottom with background
	statusY := int32(760)
	statusBg := &sdl.Rect{10, statusY - 2, 620, 20}
	dw.renderer.SetDrawColor(20, 20, 20, 255)
	dw.renderer.FillRect(statusBg)

	var statusText string
	var statusR, statusG, statusB uint8
	switch dw.debugData.DebuggerState {
	case debug.DebuggerPaused:
		statusText = "PAUSED - SPACE: resume | N: step | F: frame"
		statusR, statusG, statusB = 255, 150, 150
	case debug.DebuggerStepInstruction:
		statusText = "STEPPING - N: next step | SPACE: resume"
		statusR, statusG, statusB = 255, 255, 100
	case debug.DebuggerStepFrame:
		statusText = "FRAME STEP - F: next frame | SPACE: resume"
		statusR, statusG, statusB = 150, 255, 150
	default: // DebuggerRunning
		statusText = "RUNNING - SPACE: pause | N: step | F: frame"
		statusR, statusG, statusB = 150, 255, 150
	}

	DrawText(dw.renderer, statusText, 20, statusY, 1, statusR, statusG, statusB)
}

func (dw *DebugWindow) renderSmallSpriteTile(tile video.Tile, x, y int32) {
	// Clear the buffer (only non-zero values)
	for i := range dw.spriteTileBuffer {
		if dw.spriteTileBuffer[i] != 0 {
			dw.spriteTileBuffer[i] = 0
		}
	}

	video.RenderTileToBuffer(&tile, dw.spriteTileBuffer, 0, 0, 8, dw.defaultPalette)

	// Draw the scaled tile
	for ty := 0; ty < 8; ty++ {
		for tx := 0; tx < 8; tx++ {
			pixel := dw.spriteTileBuffer[ty*8+tx]
			r := uint8(pixel >> 24)
			g := uint8(pixel >> 16)
			b := uint8(pixel >> 8)
			dw.renderer.SetDrawColor(r, g, b, 255)
			// Draw scaled pixels
			for sy := 0; sy < spriteScale; sy++ {
				for sx := 0; sx < spriteScale; sx++ {
					dw.renderer.DrawPoint(
						x+int32(tx*spriteScale+sx),
						y+int32(ty*spriteScale+sy),
					)
				}
			}
		}
	}
}

func (dw *DebugWindow) renderPanelLabel(x, y int32, text string) {
	const fontScale = 1
	const charWidth = 6
	const charHeight = 7
	const padding = 4

	labelWidth := int32(len(text)*charWidth*fontScale + padding*2)
	labelHeight := int32(charHeight*fontScale + padding*2)

	labelRect := &sdl.Rect{x, y, labelWidth, labelHeight}
	dw.renderer.SetDrawColor(60, 60, 60, 255)
	dw.renderer.FillRect(labelRect)
	dw.renderer.SetDrawColor(180, 180, 180, 255)
	dw.renderer.DrawRect(labelRect)

	DrawText(dw.renderer, text, x+padding, y+padding, fontScale, 200, 200, 200)
}

func (dw *DebugWindow) SetVisible(visible bool) {
	dw.visible = visible
	if visible {
		dw.window.Show()
		dw.needsUpdate = true
	} else {
		dw.window.Hide()
	}
}

func (dw *DebugWindow) IsVisible() bool {
	return dw.visible
}

func (dw *DebugWindow) IsInitialized() bool {
	return dw.window != nil
}

func (dw *DebugWindow) updateWaveformSamples() {
	if dw.audioData == nil {
		return
	}

	sampleCount := 128

	debug.GenerateWaveformSamples(
		dw.waveformSamples[0][:],
		dw.audioData.Channels.Ch1.DutyCycle,
		dw.audioData.Channels.Ch1.Frequency,
		dw.audioData.Channels.Ch1.Volume,
		dw.audioData.Channels.Ch1.Enabled,
		sampleCount,
	)

	debug.GenerateWaveformSamples(
		dw.waveformSamples[1][:],
		dw.audioData.Channels.Ch2.DutyCycle,
		dw.audioData.Channels.Ch2.Frequency,
		dw.audioData.Channels.Ch2.Volume,
		dw.audioData.Channels.Ch2.Enabled,
		sampleCount,
	)

	debug.GenerateWaveformSamples(
		dw.waveformSamples[2][:],
		0,
		dw.audioData.Channels.Ch3.Frequency,
		dw.audioData.Channels.Ch3.Volume,
		dw.audioData.Channels.Ch3.Enabled,
		sampleCount,
	)

	for i := 0; i < sampleCount; i++ {
		if dw.audioData.Channels.Ch4.Enabled && dw.audioData.Channels.Ch4.Volume > 0 {
			dw.waveformSamples[3][i] = (float32(i%7) - 3.5) / 3.5 * float32(dw.audioData.Channels.Ch4.Volume) / 15.0
		} else {
			dw.waveformSamples[3][i] = 0
		}
	}

	for i := 0; i < sampleCount; i++ {
		dw.waveformSamples[4][i] = (dw.waveformSamples[0][i] +
			dw.waveformSamples[1][i] +
			dw.waveformSamples[2][i] +
			dw.waveformSamples[3][i]) / 4.0
	}
}

func (dw *DebugWindow) renderAudioPanel() {
	dw.renderPanelLabel(650, 390, "Audio Channels")

	audioRect := &sdl.Rect{650, 415, 380, 160}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(audioRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(audioRect)

	if !dw.audioData.APUEnabled {
		dw.renderer.SetDrawColor(200, 100, 100, 255)
		DrawText(dw.renderer, "APU DISABLED", 730, 470, 2, 200, 100, 100)
		return
	}

	y := int32(425)
	lineHeight := int32(30)

	channels := []struct {
		name   string
		status debug.ChannelStatus
		color  [3]uint8
	}{
		{"Ch1 Square", dw.audioData.Channels.Ch1, [3]uint8{100, 200, 100}},
		{"Ch2 Square", dw.audioData.Channels.Ch2, [3]uint8{100, 150, 200}},
		{"Ch3 Wave  ", dw.audioData.Channels.Ch3, [3]uint8{200, 150, 100}},
		{"Ch4 Noise ", dw.audioData.Channels.Ch4, [3]uint8{200, 100, 200}},
	}

	for _, ch := range channels {
		DrawText(dw.renderer, ch.name, 660, y, 1, 180, 180, 180)

		if ch.status.Enabled {
			dw.renderer.SetDrawColor(ch.color[0], ch.color[1], ch.color[2], 255)
		} else {
			dw.renderer.SetDrawColor(80, 80, 80, 255)
		}

		statusRect := &sdl.Rect{750, y, 10, 15}
		dw.renderer.FillRect(statusRect)

		volumeWidth := int32(ch.status.Volume) * 10
		if volumeWidth > 0 {
			volumeRect := &sdl.Rect{770, y, volumeWidth, 15}
			dw.renderer.FillRect(volumeRect)
		}

		DrawText(dw.renderer, ch.status.Note, 950, y, 1, 200, 200, 200)

		y += lineHeight
	}
}

func (dw *DebugWindow) renderWaveforms() {
	dw.renderPanelLabel(650, 580, "Waveforms")

	waveRect := &sdl.Rect{650, 605, 620, 180}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(waveRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(waveRect)

	colors := [][3]uint8{
		{100, 200, 100},
		{100, 150, 200},
		{200, 150, 100},
		{200, 100, 200},
		{255, 255, 255},
	}

	waveHeight := int32(30)
	waveY := int32(615)
	waveStartX := int32(660)
	waveEndX := int32(1260)
	waveWidth := waveEndX - waveStartX

	channelNames := []string{"CH1", "CH2", "CH3", "CH4", "MIX"}

	for ch := 0; ch < 5; ch++ {
		DrawText(dw.renderer, channelNames[ch], waveStartX-35, waveY+8, 1, colors[ch][0], colors[ch][1], colors[ch][2])

		centerY := waveY + waveHeight/2

		dw.renderer.SetDrawColor(60, 60, 60, 255)
		dw.renderer.DrawLine(waveStartX, centerY, waveEndX, centerY)

		dw.renderer.SetDrawColor(colors[ch][0], colors[ch][1], colors[ch][2], 255)

		samplesPerPixel := float32(128) / float32(waveWidth)

		for x := int32(0); x < waveWidth-1; x++ {
			sampleIdx1 := int(float32(x) * samplesPerPixel)
			sampleIdx2 := int(float32(x+1) * samplesPerPixel)

			if sampleIdx1 >= 128 {
				sampleIdx1 = 127
			}
			if sampleIdx2 >= 128 {
				sampleIdx2 = 127
			}

			x1 := waveStartX + x
			x2 := waveStartX + x + 1
			y1 := centerY - int32(dw.waveformSamples[ch][sampleIdx1]*float32(waveHeight)/2)
			y2 := centerY - int32(dw.waveformSamples[ch][sampleIdx2]*float32(waveHeight)/2)

			dw.renderer.DrawLine(x1, y1, x2, y2)
		}

		waveY += waveHeight + 5
	}
}

func (dw *DebugWindow) Cleanup() error {
	if dw.spriteTexture != nil {
		dw.spriteTexture.Destroy()
	}
	if dw.bgTexture != nil {
		dw.bgTexture.Destroy()
	}
	if dw.renderer != nil {
		dw.renderer.Destroy()
	}
	if dw.window != nil {
		dw.window.Destroy()
	}
	return nil
}
