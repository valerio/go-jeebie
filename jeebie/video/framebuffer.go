package video

import "math/rand"

type GBColor uint32

const (
	FramebufferWidth  = 160
	FramebufferHeight = 144
	FramebufferSize   = FramebufferWidth * FramebufferHeight
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
	colorSlice := make([]uint32, FramebufferSize)

	return &FrameBuffer{
		width:  FramebufferWidth,
		height: FramebufferHeight,
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
		case 1:
			color = BlackColor
		case 2:
			color = LightGreyColor
		case 3:
			color = DarkGreyColor
		default:
			color = BlackColor
		}

		fb.buffer[i] = uint32(color)
	}
}

// ToBinaryData returns the framebuffer as raw binary data for test comparison
func (fb *FrameBuffer) ToBinaryData() []byte {
	data := make([]byte, len(fb.buffer)*4)
	for i, pixel := range fb.buffer {
		// Convert uint32 pixel to 4 bytes (RGBA format)
		data[i*4] = byte(pixel >> 24)   // R
		data[i*4+1] = byte(pixel >> 16) // G
		data[i*4+2] = byte(pixel >> 8)  // B
		data[i*4+3] = byte(pixel)       // A
	}
	return data
}

// ToGrayscale converts the framebuffer to grayscale values for simpler comparison
func (fb *FrameBuffer) ToGrayscale() []byte {
	data := make([]byte, len(fb.buffer))
	for i, pixel := range fb.buffer {
		// Convert Game Boy colors to grayscale values (0-3)
		switch GBColor(pixel) {
		case BlackColor:
			data[i] = 0
		case DarkGreyColor:
			data[i] = 1
		case LightGreyColor:
			data[i] = 2
		case WhiteColor:
			data[i] = 3
		default:
			data[i] = 0
		}
	}
	return data
}
