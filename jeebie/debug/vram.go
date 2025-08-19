package debug

import (
	"fmt"

	"github.com/valerio/go-jeebie/jeebie/video"
)

const (
	VRAMBaseAddr     = 0x8000
	VRAMEndAddr      = 0x97FF
	TileDataSize     = 16
	TilePixelWidth   = 8
	TilePixelHeight  = 8
	TilePatternCount = 384
	TilesPerRow      = 16
	TileRows         = 24

	BackgroundTilemapAddr = 0x9800
	WindowTilemapAddr     = 0x9C00
	TilemapSize           = 0x400
)

type TilePattern struct {
	Index  int
	Pixels [TilePixelHeight][TilePixelWidth]video.GBColor
}

type TilemapInfo struct {
	BackgroundActive bool
	WindowActive     bool
	LCDCValue        uint8
}

type VRAMData struct {
	TilePatterns []TilePattern
	TilemapInfo  TilemapInfo
}

func ExtractVRAMData(reader MemoryReader) *VRAMData {
	return ExtractVRAMDataFromReader(reader)
}

func (data *VRAMData) GetTileGrid() [][]TilePattern {
	grid := make([][]TilePattern, TileRows)

	for row := 0; row < TileRows; row++ {
		grid[row] = make([]TilePattern, TilesPerRow)
		for col := 0; col < TilesPerRow; col++ {
			tileIndex := row*TilesPerRow + col
			if tileIndex < TilePatternCount {
				grid[row][col] = data.TilePatterns[tileIndex]
			}
		}
	}

	return grid
}

func (info *TilemapInfo) FormatSummary() string {
	bgStatus := "INACTIVE"
	if info.BackgroundActive {
		bgStatus = "ACTIVE"
	}

	winStatus := "INACTIVE"
	if info.WindowActive {
		winStatus = "ACTIVE"
	}

	return fmt.Sprintf("Background Map: 0x%04X [%s] | Window Map: 0x%04X [%s] | LCDC: 0x%02X",
		BackgroundTilemapAddr, bgStatus, WindowTilemapAddr, winStatus, info.LCDCValue)
}
