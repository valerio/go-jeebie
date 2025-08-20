package input

import (
	"time"

	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
)

// Handler manages input processing with debouncing for UI actions
type Handler struct {
	lastActionTime map[action.Action]time.Time
	debounceDelay  time.Duration
}

func NewHandler() *Handler {
	return &Handler{
		lastActionTime: make(map[action.Action]time.Time),
		debounceDelay:  300 * time.Millisecond,
	}
}

// ProcessEvent processes an input event, applying debouncing for Press/Release events
// Returns true if the event should be handled, false if it was debounced
func (h *Handler) ProcessEvent(evt backend.InputEvent) bool {
	if evt.Type == event.Press || evt.Type == event.Release {
		now := time.Now()
		if lastTime, exists := h.lastActionTime[evt.Action]; exists {
			if now.Sub(lastTime) < h.debounceDelay {
				return false
			}
		}
		h.lastActionTime[evt.Action] = now
	}

	return true
}
