package timing

import "time"

// Limiter controls frame rate timing for emulation.
type Limiter interface {
	// WaitForNextFrame blocks until it's time for the next frame.
	// Returns immediately if timing is behind schedule.
	WaitForNextFrame()

	// Reset resets the timing state, useful after pauses.
	Reset()
}

// NewNoOpLimiter returns a limiter that doesn't limit (for headless mode).
func NewNoOpLimiter() Limiter {
	return &noOpLimiter{}
}

type noOpLimiter struct{}

func (n *noOpLimiter) WaitForNextFrame() {}
func (n *noOpLimiter) Reset()            {}

// Constants for Game Boy timing
const (
	CyclesPerFrame = 70224
	CPUFrequency   = 4194304
)

// TargetFPS calculates the exact Game Boy frame rate.
func TargetFPS() float64 {
	return float64(CPUFrequency) / float64(CyclesPerFrame)
}

// FrameDuration returns the target duration of a single frame.
func FrameDuration() time.Duration {
	return time.Duration(float64(time.Second) / TargetFPS())
}
