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
	oamData  *debug.OAMData
	vramData *debug.VRAMData

	// Rendering state
	needsUpdate   bool
	colorLogCount int // For debugging color conversion
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

	dw.renderer.Present()
	dw.needsUpdate = false
	return nil
}

func (dw *DebugWindow) renderTileGrid() {
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

func (dw *DebugWindow) renderOAMInfo() {
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
