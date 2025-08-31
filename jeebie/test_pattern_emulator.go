package jeebie

import (
	"github.com/valerio/go-jeebie/jeebie/audio"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/timing"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// TestPatternEmulator displays test patterns without actual emulation
type TestPatternEmulator struct {
	frameBuffer      *video.FrameBuffer
	patternType      int
	animationCounter int
	limiter          timing.Limiter
}

func NewTestPatternEmulator() Emulator {
	e := &TestPatternEmulator{
		frameBuffer: video.NewFrameBuffer(),
		patternType: 0,
		limiter:     timing.NewNoOpLimiter(),
	}
	e.generateTestPattern(0)
	return e
}

func (e *TestPatternEmulator) RunUntilFrame() error {
	e.animationCounter++
	if e.animationCounter%display.TestPatternAnimationFrames == 0 {
		e.animateTestPattern()
	}
	e.limiter.WaitForNextFrame()
	return nil
}

func (e *TestPatternEmulator) GetCurrentFrame() *video.FrameBuffer {
	return e.frameBuffer
}

func (e *TestPatternEmulator) HandleAction(act action.Action, pressed bool) {
	if act == action.EmulatorTestPatternCycle && pressed {
		e.CycleTestPattern()
	}
}

func (e *TestPatternEmulator) ExtractDebugData() *debug.Data {
	return &debug.Data{
		OAM:           nil,
		VRAM:          nil,
		CPU:           nil,
		Memory:        nil,
		DebuggerState: debug.DebuggerRunning,
	}
}

func (e *TestPatternEmulator) CycleTestPattern() {
	e.patternType = (e.patternType + 1) % display.TestPatternCount
	e.generateTestPattern(e.patternType)
}

func (e *TestPatternEmulator) generateTestPattern(patternType int) {
	switch patternType {
	case 0: // Checkerboard
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x/display.TestPatternTileSize)+(y/display.TestPatternTileSize))%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.BlackColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	case 1: // Gradient
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				// Map x position to one of the 4 Game Boy colors
				colorIndex := x * 4 / video.FramebufferWidth
				var color video.GBColor
				switch colorIndex {
				case 0:
					color = video.BlackColor
				case 1:
					color = video.DarkGreyColor
				case 2:
					color = video.LightGreyColor
				default:
					color = video.WhiteColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	case 2: // Vertical stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if (x/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Diagonal lines
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}

func (e *TestPatternEmulator) animateTestPattern() {
	frame := e.animationCounter / display.TestPatternAnimationFrames
	switch e.patternType {
	case 2: // Animate stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+frame*display.TestPatternStripeSpeed)/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Animate diagonal
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y+frame*display.TestPatternDiagonalSpeed)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				e.frameBuffer.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}

func (e *TestPatternEmulator) SetFrameLimiter(limiter timing.Limiter) {
	if limiter == nil {
		e.limiter = timing.NewNoOpLimiter()
	} else {
		e.limiter = limiter
	}
}

func (e *TestPatternEmulator) ResetFrameTiming() {
	e.limiter.Reset()
}

func (e *TestPatternEmulator) GetAudioProvider() audio.Provider {
	return nil // Test pattern has no audio
}

var _ Emulator = (*TestPatternEmulator)(nil)
