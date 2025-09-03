package video

type GBColor uint32

const (
	FramebufferWidth  = 160
	FramebufferHeight = 144
	FramebufferSize   = FramebufferWidth * FramebufferHeight
)

const (
	WhiteColor     GBColor = 0xFFFFFFFF
	LightGreyColor GBColor = 0x989898FF
	DarkGreyColor  GBColor = 0x4C4C4CFF
	BlackColor     GBColor = 0x000000FF
)

func ByteToColor(value byte) GBColor {
	switch value {
	case 0:
		return WhiteColor // 00 = white (lightest)
	case 1:
		return LightGreyColor // 01 = light grey
	case 2:
		return DarkGreyColor // 10 = dark grey
	case 3:
		return BlackColor // 11 = black (darkest)
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
