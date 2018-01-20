package video

type GBColor uint32

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

// NewFrameBuffer creates a frame buffer with the specified size.
func NewFrameBuffer(width, height uint) *FrameBuffer {
	colorSlice := make([]uint32, width*height, width*height)

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
