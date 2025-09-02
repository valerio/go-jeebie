package video

// LayerFramebuffer represents a single rendering layer's framebuffer
type LayerFramebuffer struct {
	Buffer []uint32 // RGBA pixels, same format as main framebuffer
	Width  int
	Height int
}

// RenderLayers contains separate framebuffers for each rendering layer
type RenderLayers struct {
	Background *LayerFramebuffer // 256x256 full tilemap
	Window     *LayerFramebuffer // 256x256 full tilemap
	Sprites    *LayerFramebuffer // 160x144 sprite layer
	Enabled    bool              // Whether layer rendering is active
}

// NewRenderLayers creates a new set of render layer framebuffers
func NewRenderLayers() *RenderLayers {
	return &RenderLayers{
		Background: &LayerFramebuffer{
			Buffer: make([]uint32, 256*256),
			Width:  256,
			Height: 256,
		},
		Window: &LayerFramebuffer{
			Buffer: make([]uint32, 256*256),
			Width:  256,
			Height: 256,
		},
		Sprites: &LayerFramebuffer{
			Buffer: make([]uint32, 160*144),
			Width:  160,
			Height: 144,
		},
		Enabled: false,
	}
}

// Clear clears all layer framebuffers to transparent
func (r *RenderLayers) Clear() {
	if !r.Enabled {
		return
	}

	// Clear with transparent black (0x00000000)
	for i := range r.Background.Buffer {
		r.Background.Buffer[i] = 0
	}
	for i := range r.Window.Buffer {
		r.Window.Buffer[i] = 0
	}
	for i := range r.Sprites.Buffer {
		r.Sprites.Buffer[i] = 0
	}
}
