package input

import (
	"time"

	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

const (
	// debounceDuration is the minimum time between debounced events
	debounceDuration = 300 * time.Millisecond
)

// Manager handles input actions and their associated callbacks
type Manager struct {
	handlers      map[action.Action]map[event.Type][]func()
	lastTriggered map[action.Action]map[event.Type]time.Time
	joypad        *memory.Joypad
}

func NewManager(j *memory.Joypad) *Manager {
	return &Manager{
		handlers:      make(map[action.Action]map[event.Type][]func()),
		lastTriggered: make(map[action.Action]map[event.Type]time.Time),
		joypad:        j,
	}
}

// On registers a callback for a specific action and event type
func (m *Manager) On(act action.Action, evt event.Type, callback func()) {
	if m.handlers[act] == nil {
		m.handlers[act] = make(map[event.Type][]func())
	}
	if m.lastTriggered[act] == nil {
		m.lastTriggered[act] = make(map[event.Type]time.Time)
	}

	m.handlers[act][evt] = append(m.handlers[act][evt], callback)
}

// Trigger handles the given action and event type.
func (m *Manager) Trigger(act action.Action, evt event.Type) {
	// Debounce Press and Release events
	if evt == event.Press || evt == event.Release {
		now := time.Now()
		if m.lastTriggered[act] == nil {
			m.lastTriggered[act] = make(map[event.Type]time.Time)
		}
		lastTime := m.lastTriggered[act][evt]
		if now.Sub(lastTime) < debounceDuration {
			return
		}
		m.lastTriggered[act][evt] = now
	}

	// GB controls, written directly to memory (joypad)
	if m.joypad != nil {
		joypadKey := m.getJoypadKey(act)
		if joypadKey != 0 { // Only handle actual GB controls
			switch evt {
			case event.Press:
				m.joypad.Press(joypadKey)
			case event.Release:
				m.joypad.Release(joypadKey)
			}
			return // Only return for GB controls
		}
	}

	// Other emulator actions
	if m.handlers[act] != nil && len(m.handlers[act][evt]) > 0 {
		for _, callback := range m.handlers[act][evt] {
			callback()
		}
	}
}

// getJoypadKey maps Game Boy actions to joypad keys
func (m *Manager) getJoypadKey(act action.Action) memory.JoypadKey {
	switch act {
	case action.GBButtonA:
		return memory.JoypadA
	case action.GBButtonB:
		return memory.JoypadB
	case action.GBButtonStart:
		return memory.JoypadStart
	case action.GBButtonSelect:
		return memory.JoypadSelect
	case action.GBDPadUp:
		return memory.JoypadUp
	case action.GBDPadDown:
		return memory.JoypadDown
	case action.GBDPadLeft:
		return memory.JoypadLeft
	case action.GBDPadRight:
		return memory.JoypadRight
	default:
		return 0
	}
}
