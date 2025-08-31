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
	debugData  *debug.Data // Full debug data for disassembly
	spriteVis  *debug.SpriteVisualizer
	bgVis      *debug.BackgroundVisualizer
	paletteVis *debug.PaletteVisualizer
	audioData  *debug.AudioData

	// Waveform visualization
	waveformSamples [5][128]float32 // Ch1-4 + Mix

	// Pre-allocated buffers to avoid allocations in hot loops
	tilemapPixelBuffer []byte // 256*256*4 bytes for tilemap rendering

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

	dw.window.Hide()
	return nil
}

func (dw *DebugWindow) UpdateData(debugData *debug.Data) {
	if debugData == nil {
		return
	}

	dw.debugData = debugData
	dw.spriteVis = debugData.SpriteVis
	dw.bgVis = debugData.BackgroundVis
	dw.paletteVis = debugData.PaletteVis
	dw.audioData = debugData.Audio
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

	panelRect := &sdl.Rect{10, 35, 420, 340}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.spriteVis == nil {
		return
	}

	// Show first 20 sprites regardless of visibility
	sprites := dw.spriteVis.Sprites
	maxDisplay := 20
	if len(sprites) < maxDisplay {
		maxDisplay = len(sprites)
	}

	for i := 0; i < maxDisplay && i < len(sprites); i++ {
		sprite := sprites[i]
		y := int32(45 + i*15)
		x := int32(20)

		dw.renderSmallSpriteTile(sprite.TileData, x, y)

		var paletteStr string
		if sprite.Info.Sprite.PaletteOBP1 {
			paletteStr = "OBP1"
		} else {
			paletteStr = "OBP0"
		}

		// Color code based on visibility
		textR, textG, textB := uint8(200), uint8(200), uint8(200)
		if !sprite.Info.IsVisible {
			textR, textG, textB = 100, 100, 100
		}

		info := fmt.Sprintf("#%02d T:%02X X:%3d Y:%3d %s",
			sprite.Info.Index,
			sprite.Info.Sprite.TileIndex,
			sprite.X,
			sprite.Y,
			paletteStr,
		)

		DrawText(dw.renderer, info, x+20, y, 1, textR, textG, textB)

		var flags []string
		if sprite.Info.Sprite.FlipX {
			flags = append(flags, "FX")
		}
		if sprite.Info.Sprite.FlipY {
			flags = append(flags, "FY")
		}
		if sprite.Info.Sprite.BehindBG {
			flags = append(flags, "BG")
		}

		if len(flags) > 0 {
			flagStr := ""
			for i, f := range flags {
				if i > 0 {
					flagStr += " "
				}
				flagStr += f
			}
			DrawText(dw.renderer, flagStr, x+280, y, 1, 150, 200, 255)
		}
	}
}

func (dw *DebugWindow) renderBackgroundPanel() {
	dw.renderPanelLabel(450, 10, "Background Tilemap")

	panelRect := &sdl.Rect{450, 35, 320, 320}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.bgVis == nil || !dw.bgVis.BGEnabled {
		DrawText(dw.renderer, "Background Disabled", 500, 180, 2, 100, 100, 100)
		return
	}

	dw.renderTilemap()

	scrollX := int32(dw.bgVis.ScrollX)
	scrollY := int32(dw.bgVis.ScrollY)
	viewportX := int32(460) + scrollX
	viewportY := int32(45) + scrollY
	viewportRect := &sdl.Rect{viewportX, viewportY, 160, 144}
	dw.renderer.SetDrawColor(255, 255, 0, 128)
	dw.renderer.DrawRect(viewportRect)

	if active, wx, wy := dw.bgVis.GetWindowViewport(); active {
		windowRect := &sdl.Rect{int32(460 + wx), int32(45 + wy), 160, 144}
		dw.renderer.SetDrawColor(0, 255, 255, 128)
		dw.renderer.DrawRect(windowRect)
	}

	infoY := int32(360)
	winStatus := "OFF"
	if dw.bgVis.WindowEnabled {
		winStatus = fmt.Sprintf("ON (X:%d Y:%d)", dw.bgVis.WindowX, dw.bgVis.WindowY)
	}
	info := fmt.Sprintf("SCX:%d SCY:%d | Win: %s",
		dw.bgVis.ScrollX, dw.bgVis.ScrollY, winStatus,
	)
	DrawText(dw.renderer, info, 460, infoY, 1, 200, 200, 200)

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
	DrawText(dw.renderer, mapInfo, 460, infoY+15, 1, 150, 150, 150)
}

func (dw *DebugWindow) renderTilemap() {
	// Note: We don't clear the buffer as tiles will overwrite all pixels

	for row := 0; row < 32; row++ {
		for col := 0; col < 32; col++ {
			tileIndex := dw.bgVis.Tilemap[row][col]
			var tile video.Tile

			// Use the same tile fetching logic as the GPU
			useSigned := dw.bgVis.TileDataBase == 0x8800
			tile = debug.GetTileForBackgroundIndex(dw.bgVis.TileData, tileIndex, useSigned)

			dw.renderTileToBuffer(tile, dw.tilemapPixelBuffer, row*8, col*8, 256)
		}
	}

	dw.bgTexture.Update(nil, unsafe.Pointer(&dw.tilemapPixelBuffer[0]), 256*4)

	srcRect := &sdl.Rect{0, 0, 256, 256}
	dstRect := &sdl.Rect{460, 45, 300, 300}
	dw.renderer.Copy(dw.bgTexture, srcRect, dstRect)
}

func (dw *DebugWindow) renderPalettePanel() {
	dw.renderPanelLabel(790, 10, "Palettes")

	panelRect := &sdl.Rect{790, 35, 280, 130}
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
		x := int32(800)

		DrawText(dw.renderer, pal.name, x, y, 1, 200, 200, 200)

		for j := 0; j < 4; j++ {
			colorX := x + 40 + int32(j*30)
			r, g, b, _ := dw.gbColorToRGBA(pal.info.Colors[j])

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
	dw.renderPanelLabel(10, 390, "Disassembly")

	panelRect := &sdl.Rect{10, 415, 420, 370}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(panelRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(panelRect)

	if dw.debugData == nil || dw.debugData.CPU == nil || dw.debugData.Memory == nil {
		DrawText(dw.renderer, "No debug data available", 20, 430, 1, 100, 100, 100)
		return
	}

	pc := dw.debugData.CPU.PC

	disasmLines := debug.CreateDisassembly(dw.debugData.Memory, pc, maxDisasmLines)

	// Render each line
	y := int32(425)
	lineHeight := int32(16)

	for _, line := range disasmLines {
		if y+lineHeight > 750 { // Leave space for status line
			break
		}

		var r, g, b uint8
		if line.IsCurrent {
			// Current instruction - bright yellow
			r, g, b = 255, 255, 100
			DrawText(dw.renderer, ">", 15, y, 1, 255, 255, 100)
		} else {
			// Normal instruction - light gray
			r, g, b = 180, 180, 180
		}
		text := fmt.Sprintf("%04X: %s", line.Address, line.Instruction)
		DrawText(dw.renderer, text, 30, y, 1, r, g, b)
		y += lineHeight
	}

	// Draw status line at bottom with background
	statusY := int32(760)
	statusBg := &sdl.Rect{10, statusY - 2, 420, 20}
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
	pixels := tile.Pixels()
	for py := 0; py < 8; py++ {
		for px := 0; px < 8; px++ {
			r, g, b, _ := dw.gbColorToRGBA(pixels[py][px])
			dw.renderer.SetDrawColor(r, g, b, 255)
			dw.renderer.DrawPoint(x+int32(px), y+int32(py))
		}
	}
}

func (dw *DebugWindow) renderTileToBuffer(tile video.Tile, buffer []byte, row, col, stride int) {
	pixels := tile.Pixels()
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			offset := ((row+y)*stride + (col + x)) * 4
			if offset+3 < len(buffer) {
				r, g, b, a := dw.gbColorToRGBA(pixels[y][x])
				// SDL2 RGBA8888 format is actually ABGR in memory
				buffer[offset] = a   // Alpha
				buffer[offset+1] = b // Blue
				buffer[offset+2] = g // Green
				buffer[offset+3] = r // Red
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

func (dw *DebugWindow) gbColorToRGBA(color video.GBColor) (r, g, b, a uint8) {
	switch int(color) {
	case 0:
		return 255, 255, 255, 255
	case 1:
		return 170, 170, 170, 255
	case 2:
		return 85, 85, 85, 255
	case 3:
		return 0, 0, 0, 255
	default:
		return 255, 0, 0, 255
	}
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
	dw.renderPanelLabel(450, 390, "Audio Channels")

	audioRect := &sdl.Rect{450, 415, 380, 160}
	dw.renderer.SetDrawColor(40, 40, 40, 255)
	dw.renderer.FillRect(audioRect)
	dw.renderer.SetDrawColor(100, 100, 100, 255)
	dw.renderer.DrawRect(audioRect)

	if !dw.audioData.APUEnabled {
		dw.renderer.SetDrawColor(200, 100, 100, 255)
		DrawText(dw.renderer, "APU DISABLED", 530, 470, 2, 200, 100, 100)
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
		DrawText(dw.renderer, ch.name, 460, y, 1, 180, 180, 180)

		if ch.status.Enabled {
			dw.renderer.SetDrawColor(ch.color[0], ch.color[1], ch.color[2], 255)
		} else {
			dw.renderer.SetDrawColor(80, 80, 80, 255)
		}

		statusRect := &sdl.Rect{550, y, 10, 15}
		dw.renderer.FillRect(statusRect)

		volumeWidth := int32(ch.status.Volume) * 10
		if volumeWidth > 0 {
			volumeRect := &sdl.Rect{570, y, volumeWidth, 15}
			dw.renderer.FillRect(volumeRect)
		}

		DrawText(dw.renderer, ch.status.Note, 750, y, 1, 200, 200, 200)

		y += lineHeight
	}
}

func (dw *DebugWindow) renderWaveforms() {
	dw.renderPanelLabel(450, 570, "Waveforms")

	waveRect := &sdl.Rect{450, 595, 620, 190}
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
	waveY := int32(605)
	waveStartX := int32(460)
	waveEndX := int32(1060)
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
