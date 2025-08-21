package timing

import "time"

// TickerLimiter uses time.Ticker for simple, consistent frame timing.
// Less accurate than AdaptiveLimiter but simpler and good enough for most cases.
type TickerLimiter struct {
	ticker *time.Ticker
	ch     <-chan time.Time
}

func NewTickerLimiter() *TickerLimiter {
	ticker := time.NewTicker(FrameDuration())
	return &TickerLimiter{
		ticker: ticker,
		ch:     ticker.C,
	}
}

func (t *TickerLimiter) WaitForNextFrame() {
	<-t.ch
}

func (t *TickerLimiter) Reset() {
	t.ticker.Reset(FrameDuration())
}

func (t *TickerLimiter) Stop() {
	t.ticker.Stop()
}
