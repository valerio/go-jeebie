package jeebie

import (
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/timing"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// Emulator is the interface for all emulator implementations
type Emulator interface {
	RunUntilFrame() error
	GetCurrentFrame() *video.FrameBuffer
	HandleAction(act action.Action, pressed bool)
	ExtractDebugData() *debug.CompleteDebugData
	SetFrameLimiter(limiter timing.Limiter)
	ResetFrameTiming()
}

var _ Emulator = (*DMG)(nil)
