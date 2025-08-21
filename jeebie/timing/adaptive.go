package timing

import (
	"log/slog"
	"time"
)

// AdaptiveLimiter uses precise timing with drift compensation.
// Combines sleep for efficiency with busy-waiting for accuracy.
type AdaptiveLimiter struct {
	targetFrameTime time.Duration
	nextFrameTime   time.Time
	frameCounter    int64
}

func NewAdaptiveLimiter() *AdaptiveLimiter {
	return &AdaptiveLimiter{
		targetFrameTime: FrameDuration(),
		nextFrameTime:   time.Now(),
	}
}

func (a *AdaptiveLimiter) WaitForNextFrame() {
	now := time.Now()
	sleepTime := a.nextFrameTime.Sub(now)

	if sleepTime > 0 {
		if sleepTime < 2*time.Millisecond {
			for time.Now().Before(a.nextFrameTime) {
				// busy-wait for times under 2ms, higher accuracy.
			}
		} else {
			time.Sleep(sleepTime - time.Millisecond)
			for time.Now().Before(a.nextFrameTime) {
			}
		}
	} else if sleepTime < -5*time.Millisecond {
		a.nextFrameTime = now
	}

	a.nextFrameTime = a.nextFrameTime.Add(a.targetFrameTime)
	a.frameCounter++

	if a.frameCounter%60 == 0 {
		actualTime := time.Now()
		expectedTime := a.nextFrameTime
		drift := actualTime.Sub(expectedTime)

		if drift.Abs() > 10*time.Millisecond {
			a.nextFrameTime = a.nextFrameTime.Add(drift / 10)
			slog.Debug("Frame timing drift correction",
				"drift_ms", drift.Milliseconds(),
				"fps", float64(a.frameCounter)*float64(time.Second)/float64(actualTime.Sub(a.nextFrameTime.Add(-time.Duration(a.frameCounter)*a.targetFrameTime))))
		}
	}
}

func (a *AdaptiveLimiter) Reset() {
	a.nextFrameTime = time.Now()
	a.frameCounter = 0
}
