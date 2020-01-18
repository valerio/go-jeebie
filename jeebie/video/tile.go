package video

import (
	"github.com/valerio/go-jeebie/jeebie/bit"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

type Tile struct {
	buffer []GBColor
}

func newTile(address uint16, mmu *memory.MMU) *Tile {
	buffer := make([]GBColor, 64)

	for tileLine := 0; tileLine < 8; tileLine++ {
		// each line is 2 bytes
		lineStartAddress := address + 2*uint16(tileLine)

		lowPixelLine := mmu.Read(lineStartAddress)
		highPixelLine := mmu.Read(lineStartAddress + 1)

		// compose colors pixel by pixel
		for pixel := 0; pixel < 8; pixel++ {
			pixelIndex := 7 - uint8(pixel)
			pixelColorValue := bit.GetBitValue(pixelIndex, highPixelLine)<<1 | bit.GetBitValue(pixelIndex, lowPixelLine)

			buffer[pixel*8+tileLine] = ByteToColor(pixelColorValue)
		}
	}

	return &Tile{
		buffer: make([]GBColor, 0),
	}
}

func (t *Tile) getPixel(x, y uint) GBColor {
	return t.buffer[(y*8)+x]
}
