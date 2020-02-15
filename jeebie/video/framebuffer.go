package video

import "math/rand"

type GBColor uint32

const (
	framebufferWidth  = 160
	framebufferHeight = 144
	framebufferSize   = framebufferWidth * framebufferHeight
)

const (
	WhiteColor     GBColor = 0xFFFFFFFF
	LightGreyColor         = 0xFF989898
	DarkGreyColor          = 0xFF4C4C4C
	BlackColor             = 0xFF000000
)

func ByteToColor(value byte) GBColor {
	switch value {
	case 0:
		return BlackColor
	case 1:
		return DarkGreyColor
	case 2:
		return LightGreyColor
	case 3:
		return WhiteColor
	}

	return 0
}

type FrameBuffer struct {
	width  uint
	height uint
	buffer []uint32
}

func NewFrameBuffer() *FrameBuffer {
	colorSlice := make([]uint32, framebufferSize, framebufferSize)

	return &FrameBuffer{
		width:  width,
		height: height,
		buffer: colorSlice,
	}
}

func (fb FrameBuffer) GetPixel(x, y uint) uint32 {
	return fb.buffer[y*fb.width+x]
}

func (fb *FrameBuffer) SetPixel(x, y uint, color GBColor) {
	fb.buffer[y*fb.width+x] = uint32(color)
}

func (fb *FrameBuffer) ToSlice() []uint32 {
	return fb.buffer
}

func (fb *FrameBuffer) DrawNoise() {
	// placeholder: draws random pixels
	for i := 0; i < len(fb.buffer); i++ {

		var color GBColor
		switch rand.Uint32() % 4 {
		case 0:
			color = WhiteColor
			break
		case 1:
			color = BlackColor
			break
		case 2:
			color = LightGreyColor
			break
		case 3:
			color = DarkGreyColor
			break
		default:
			color = BlackColor
		}

		fb.buffer[i] = uint32(color)
	}
}
