package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/video"
)

type mockMemoryReader struct {
	memory map[uint16]uint8
}

func newMockMemoryReader() *mockMemoryReader {
	return &mockMemoryReader{
		memory: make(map[uint16]uint8),
	}
}

func (m *mockMemoryReader) Read(addr uint16) uint8 {
	if val, ok := m.memory[addr]; ok {
		return val
	}
	return 0
}

func (m *mockMemoryReader) ReadBit(bit uint8, addr uint16) bool {
	return (m.Read(addr) & (1 << bit)) != 0
}

func (m *mockMemoryReader) write(addr uint16, val uint8) {
	m.memory[addr] = val
}

func TestExtractSpriteData(t *testing.T) {
	reader := newMockMemoryReader()

	reader.write(0xFF40, 0x04)
	reader.write(0xFF46, 0x00)
	reader.write(0xFF47, 0xE4)
	reader.write(0xFF48, 0xD0)
	reader.write(0xFF49, 0x90)

	reader.write(0xFE00, 16)
	reader.write(0xFE01, 8)
	reader.write(0xFE02, 0x10)
	reader.write(0xFE03, 0x00)

	vis := ExtractSpriteData(reader, 0)

	assert.NotNil(t, vis)
	assert.Equal(t, 16, vis.SpriteHeight)
	assert.Equal(t, uint8(0), vis.CurrentLine)
	assert.Equal(t, uint8(0xD0), vis.PaletteOBP0)
	assert.Equal(t, uint8(0x90), vis.PaletteOBP1)
	assert.Equal(t, 40, len(vis.Sprites))
}

func TestExtractBackgroundData(t *testing.T) {
	reader := newMockMemoryReader()

	reader.write(0xFF40, 0x91)
	reader.write(0xFF42, 20)
	reader.write(0xFF43, 10)
	reader.write(0xFF4A, 60)
	reader.write(0xFF4B, 50)
	reader.write(0xFF47, 0xE4)

	for i := uint16(0); i < 1024; i++ {
		reader.write(0x9800+i, uint8(i&0xFF))
	}

	vis := ExtractBackgroundData(reader)

	assert.NotNil(t, vis)
	assert.True(t, vis.BGEnabled)
	assert.Equal(t, uint8(10), vis.ScrollX)
	assert.Equal(t, uint8(20), vis.ScrollY)
	assert.Equal(t, uint16(0x9800), vis.TilemapBase)
	assert.Equal(t, uint16(0x8000), vis.TileDataBase)
}

func TestExtractPaletteData(t *testing.T) {
	reader := newMockMemoryReader()

	reader.write(0xFF47, 0xE4)
	reader.write(0xFF48, 0xD0)
	reader.write(0xFF49, 0x90)

	vis := ExtractPaletteData(reader)

	assert.NotNil(t, vis)
	assert.Equal(t, uint8(0xE4), vis.BGP.Raw)
	assert.Equal(t, uint8(0xD0), vis.OBP0.Raw)
	assert.Equal(t, uint8(0x90), vis.OBP1.Raw)

	assert.Equal(t, video.GBColor(0), vis.BGP.Colors[0])
	assert.Equal(t, video.GBColor(1), vis.BGP.Colors[1])
	assert.Equal(t, video.GBColor(2), vis.BGP.Colors[2])
	assert.Equal(t, video.GBColor(3), vis.BGP.Colors[3])
}

func TestGetViewportTiles(t *testing.T) {
	reader := newMockMemoryReader()

	reader.write(0xFF40, 0x91)
	reader.write(0xFF42, 0)
	reader.write(0xFF43, 0)

	for row := 0; row < 32; row++ {
		for col := 0; col < 32; col++ {
			addr := uint16(0x9800 + row*32 + col)
			reader.write(addr, uint8(row*32+col))
		}
	}

	vis := ExtractBackgroundData(reader)
	viewport := vis.GetViewportTiles()

	assert.Equal(t, 18, len(viewport))
	assert.Equal(t, 20, len(viewport[0]))
	assert.Equal(t, uint8(0), viewport[0][0])
	assert.Equal(t, uint8(1), viewport[0][1])
	assert.Equal(t, uint8(32), viewport[1][0])
}

func TestGetWindowViewport(t *testing.T) {
	reader := newMockMemoryReader()

	reader.write(0xFF40, 0xA1)
	reader.write(0xFF4A, 60)
	reader.write(0xFF4B, 50)

	vis := ExtractBackgroundData(reader)
	active, startX, startY := vis.GetWindowViewport()

	assert.True(t, active)
	assert.Equal(t, 43, startX)
	assert.Equal(t, 60, startY)

	reader.write(0xFF40, 0x81)
	vis = ExtractBackgroundData(reader)
	active, _, _ = vis.GetWindowViewport()
	assert.False(t, active)
}

func TestApplyPalette(t *testing.T) {
	palette := PaletteInfo{
		Raw: 0xE4,
		Colors: [4]video.GBColor{
			video.GBColor(0),
			video.GBColor(1),
			video.GBColor(2),
			video.GBColor(3),
		},
	}

	assert.Equal(t, video.GBColor(0), ApplyPalette(0, palette))
	assert.Equal(t, video.GBColor(1), ApplyPalette(1, palette))
	assert.Equal(t, video.GBColor(2), ApplyPalette(2, palette))
	assert.Equal(t, video.GBColor(3), ApplyPalette(3, palette))
}
