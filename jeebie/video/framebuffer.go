package video

type GBColor uint32

const (
	WhiteColor     GBColor = 0xFFFFFFFF
	LightGreyColor         = 0xFF989898
	DarkGreyColor          = 0xFF4C4C4C
	BlackColor             = 0xFF000000
)

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
