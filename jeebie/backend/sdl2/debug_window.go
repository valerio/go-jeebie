//go:build sdl2

package sdl2

import (
	"log/slog"
	"unsafe"

	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/video"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	DebugWindowWidth  = 800
	DebugWindowHeight = 600
	DebugWindowTitle  = "Game Boy Debug Tools"
)

type DebugWindow struct {
	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	visible  bool

	// Debug data
	oamData   *debug.OAMData
	vramData  *debug.VRAMData
	audioData *debug.AudioData

	// Rendering state
	needsUpdate   bool
	colorLogCount int // For debugging color conversion

	// Waveform visualization
	waveformSamples [5][128]float32 // Ch1-4 + Mix, 128 samples each
}

func NewDebugWindow() *DebugWindow {
	return &DebugWindow{
		visible:     false,
		needsUpdate: true,
	}
}

func (dw *DebugWindow) Init() error {
	window, err := sdl.CreateWindow(
		DebugWindowTitle,
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

	// Create texture for tile rendering
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_RGBA8888,
		sdl.TEXTUREACCESS_STREAMING,
		debug.TilesPerRow*debug.TilePixelWidth,
		debug.TileRows*debug.TilePixelHeight,
	)
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		return err
	}
	dw.texture = texture

	// Initially hide the window
	dw.window.Hide()
	return nil
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

func (dw *DebugWindow) UpdateData(oam *debug.OAMData, vram *debug.VRAMData) {
	slog.Info("Debug window updating data", "oam_nil", oam == nil, "vram_nil", vram == nil)
	dw.oamData = oam
	dw.vramData = vram
	dw.needsUpdate = true
}

func (dw *DebugWindow) UpdateAudioData(audio *debug.AudioData) {
	dw.audioData = audio
	if audio != nil {
		dw.updateWaveformSamples()
	}
	dw.needsUpdate = true
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

func (dw *DebugWindow) Render() error {
	if !dw.visible {
		return nil
	}
	if !dw.needsUpdate {
		return nil
	}

	slog.Info("Debug window rendering", "has_vram", dw.vramData != nil, "has_oam", dw.oamData != nil)

	dw.renderer.SetDrawColor(32, 32, 32, 255)
	dw.renderer.Clear()

	if dw.vramData != nil {
		dw.renderTileGrid()
	}

	if dw.oamData != nil {
		dw.renderOAMInfo()
	}

	if dw.audioData != nil {
		dw.renderAudioStatus()
		dw.renderWaveforms()
	}

	dw.renderer.Present()
	dw.needsUpdate = false
	return nil
}

func (dw *DebugWindow) renderTileGrid() {
	// Render label
	dw.renderLabel(400, 30, "VRAM Tiles")

	tileGrid := dw.vramData.GetTileGrid()
	slog.Info("Rendering tile grid", "grid_rows", len(tileGrid), "first_row_cols", len(tileGrid[0]))

	// Create pixel data for all tiles
	pixelData := make([]byte, debug.TilesPerRow*debug.TilePixelWidth*debug.TileRows*debug.TilePixelHeight*4)

	for row := 0; row < debug.TileRows; row++ {
		for col := 0; col < debug.TilesPerRow; col++ {
			if col >= len(tileGrid[row]) {
				continue
			}

			tile := tileGrid[row][col]
			dw.renderTileToPixels(tile, pixelData, row, col)
		}
	}

	// Update texture
	err := dw.texture.Update(nil, unsafe.Pointer(&pixelData[0]), debug.TilesPerRow*debug.TilePixelWidth*4)
	if err != nil {
		slog.Warn("Failed to update debug texture", "error", err)
	}

	// Show only first 8 rows with larger scaling for better visibility
	showRows := 8
	showCols := debug.TilesPerRow
	srcRect := &sdl.Rect{0, 0, int32(showCols * debug.TilePixelWidth), int32(showRows * debug.TilePixelHeight)}

	// Scale each 8x8 tile to 16x16 pixels (2x scaling)
	scaledWidth := showCols * debug.TilePixelWidth * 2   // 16 * 8 * 2 = 256
	scaledHeight := showRows * debug.TilePixelHeight * 2 // 8 * 8 * 2 = 128

	dstRect := &sdl.Rect{400, 50, int32(scaledWidth), int32(scaledHeight)}

	dw.renderer.Copy(dw.texture, srcRect, dstRect)
}

func (dw *DebugWindow) renderTileToPixels(tile video.Tile, pixelData []byte, row, col int) {
	baseOffset := (row*debug.TilePixelHeight*debug.TilesPerRow*debug.TilePixelWidth + col*debug.TilePixelWidth) * 4
	pixels := tile.Pixels()

	for y := 0; y < debug.TilePixelHeight; y++ {
		for x := 0; x < debug.TilePixelWidth; x++ {
			pixelOffset := baseOffset + (y*debug.TilesPerRow*debug.TilePixelWidth+x)*4

			if pixelOffset+3 >= len(pixelData) {
				continue
			}

			r, g, b, a := dw.gbColorToRGBA(pixels[y][x])
			pixelData[pixelOffset] = a   // Alpha (first byte)
			pixelData[pixelOffset+1] = b // Blue
			pixelData[pixelOffset+2] = g // Green
			pixelData[pixelOffset+3] = r // Red (last byte)
		}
	}
}

func (dw *DebugWindow) renderLabel(x, y int32, text string) {
	// Calculate proper label size based on font scale
	const fontScale = 2
	const charWidth = 6  // 5 pixels + 1 space per character
	const charHeight = 7 // Font is 7 pixels tall
	const padding = 4    // Padding on each side

	labelWidth := int32(len(text)*charWidth*fontScale + padding*2)
	labelHeight := int32(charHeight*fontScale + padding*2)

	// Render label box
	labelRect := &sdl.Rect{x, y, labelWidth, labelHeight}
	dw.renderer.SetDrawColor(70, 70, 70, 255)
	dw.renderer.FillRect(labelRect)
	dw.renderer.SetDrawColor(200, 200, 200, 255)
	dw.renderer.DrawRect(labelRect)

	// Draw the text centered in the box
	DrawText(dw.renderer, text, x+padding, y+padding, fontScale, 200, 200, 200)
}

func (dw *DebugWindow) renderOAMInfo() {
	// Render label
	dw.renderLabel(10, 30, "OAM/Sprites")

	// Background for the OAM panel
	oamRect := &sdl.Rect{10, 50, 350, 500}
	dw.renderer.SetDrawColor(48, 48, 48, 255)
	dw.renderer.FillRect(oamRect)

	// Draw border
	dw.renderer.SetDrawColor(128, 128, 128, 255)
	dw.renderer.DrawRect(oamRect)

	if dw.oamData == nil {
		return
	}

	// Count visible sprites for display
	visibleCount := 0
	for _, spriteInfo := range dw.oamData.Sprites {
		if spriteInfo.IsVisible {
			visibleCount++
		}
	}

	// For now, just draw some lines to show we have OAM data
	// Each visible sprite gets a small rectangle to indicate presence
	dw.renderer.SetDrawColor(200, 200, 200, 255)

	maxDisplay := 20 // Show first 20 sprites
	if len(dw.oamData.Sprites) < maxDisplay {
		maxDisplay = len(dw.oamData.Sprites)
	}

	for i := 0; i < maxDisplay; i++ {
		spriteInfo := dw.oamData.Sprites[i]
		y := int32(60 + i*20)

		// Draw sprite info as colored rectangles
		if spriteInfo.IsVisible {
			dw.renderer.SetDrawColor(100, 200, 100, 255) // Green for visible
		} else {
			dw.renderer.SetDrawColor(100, 100, 100, 255) // Gray for hidden
		}

		spriteRect := &sdl.Rect{20, y, 10, 15}
		dw.renderer.FillRect(spriteRect)

		// Show tile index as a second rectangle
		tileColor := uint8(spriteInfo.Sprite.TileIndex%200 + 55) // Vary color by tile index
		dw.renderer.SetDrawColor(tileColor, tileColor, tileColor, 255)
		tileRect := &sdl.Rect{40, y, 10, 15}
		dw.renderer.FillRect(tileRect)
	}
}

func (dw *DebugWindow) gbColorToRGBA(color video.GBColor) (r, g, b, a uint8) {
	// GB colors are raw integers 0-3, not the named constants
	switch int(color) {
	case 0: // Lightest (white in Game Boy context)
		return 255, 255, 255, 255
	case 1: // Light gray
		return 170, 170, 170, 255
	case 2: // Dark gray
		return 85, 85, 85, 255
	case 3: // Darkest (black in Game Boy context)
		return 0, 0, 0, 255
	default:
		return 255, 0, 0, 255 // Red for debugging unknown colors
	}
}

func (dw *DebugWindow) ProcessEvent(event sdl.Event) {
	// Handle debug window specific events if needed
}

func (dw *DebugWindow) renderAudioStatus() {
	// Render label
	dw.renderLabel(10, 280, "Audio Channels")

	audioRect := &sdl.Rect{10, 300, 350, 180}
	dw.renderer.SetDrawColor(48, 48, 48, 255)
	dw.renderer.FillRect(audioRect)

	dw.renderer.SetDrawColor(128, 128, 128, 255)
	dw.renderer.DrawRect(audioRect)

	if !dw.audioData.APUEnabled {
		dw.renderer.SetDrawColor(200, 100, 100, 255)
		disabledRect := &sdl.Rect{20, 320, 100, 20}
		dw.renderer.FillRect(disabledRect)
		return
	}

	y := int32(320)
	lineHeight := int32(25)

	channels := []struct {
		name   string
		status debug.ChannelStatus
		color  [3]uint8
	}{
		{"Ch1", dw.audioData.Channels.Ch1, [3]uint8{100, 200, 100}},
		{"Ch2", dw.audioData.Channels.Ch2, [3]uint8{100, 150, 200}},
		{"Ch3", dw.audioData.Channels.Ch3, [3]uint8{200, 150, 100}},
		{"Ch4", dw.audioData.Channels.Ch4, [3]uint8{200, 100, 200}},
	}

	for _, ch := range channels {
		if ch.status.Enabled {
			dw.renderer.SetDrawColor(ch.color[0], ch.color[1], ch.color[2], 255)
		} else {
			dw.renderer.SetDrawColor(80, 80, 80, 255)
		}

		statusRect := &sdl.Rect{20, y, 10, 15}
		dw.renderer.FillRect(statusRect)

		volumeWidth := int32(ch.status.Volume) * 8
		if volumeWidth > 0 {
			volumeRect := &sdl.Rect{40, y, volumeWidth, 15}
			dw.renderer.FillRect(volumeRect)
		}

		y += lineHeight
	}

	masterLeft := int32(dw.audioData.MasterVolume.Left) * 10
	masterRight := int32(dw.audioData.MasterVolume.Right) * 10

	dw.renderer.SetDrawColor(200, 200, 200, 255)
	leftRect := &sdl.Rect{20, y + 10, masterLeft, 10}
	rightRect := &sdl.Rect{20, y + 25, masterRight, 10}
	dw.renderer.FillRect(leftRect)
	dw.renderer.FillRect(rightRect)
}

func (dw *DebugWindow) renderWaveforms() {
	// Render label
	dw.renderLabel(400, 280, "Waveforms")

	waveRect := &sdl.Rect{400, 300, 380, 280}
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

	waveHeight := int32(50)
	waveY := int32(320)
	waveStartX := int32(410)
	waveEndX := int32(770)
	waveWidth := waveEndX - waveStartX

	// channelNames := []string{"CH1", "CH2", "CH3", "CH4", "MIX"} // Reserved for future text rendering

	for ch := 0; ch < 5; ch++ {
		// Draw channel indicator
		indicatorRect := &sdl.Rect{waveStartX - 35, waveY + 15, 30, 20}
		dw.renderer.SetDrawColor(colors[ch][0]/2, colors[ch][1]/2, colors[ch][2]/2, 255)
		dw.renderer.FillRect(indicatorRect)
		dw.renderer.SetDrawColor(colors[ch][0], colors[ch][1], colors[ch][2], 255)
		dw.renderer.DrawRect(indicatorRect)

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
	if dw.texture != nil {
		dw.texture.Destroy()
	}
	if dw.renderer != nil {
		dw.renderer.Destroy()
	}
	if dw.window != nil {
		dw.window.Destroy()
	}
	return nil
}
