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
	LightGreyColor         = 0x989898FF
	DarkGreyColor          = 0x4C4C4CFF
	BlackColor             = 0x000000FF
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
		width:  framebufferWidth,
		height: framebufferHeight,
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

// Clear resets the framebuffer to a black screen.
func (fb *FrameBuffer) Clear() {
	for i := range fb.buffer {
		fb.buffer[i] = 0
	}
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
