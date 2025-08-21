package input

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
)

func TestHandler_Debouncing(t *testing.T) {
	tests := []struct {
		name           string
		action         action.Action
		eventType      event.Type
		timeBetween    time.Duration
		expectDebounce bool
	}{
		{
			name:           "UI action rapid press - should debounce",
			action:         action.EmulatorDebugToggle,
			eventType:      event.Press,
			timeBetween:    100 * time.Millisecond,
			expectDebounce: true,
		},
		{
			name:           "UI action slow press - should not debounce",
			action:         action.EmulatorDebugToggle,
			eventType:      event.Press,
			timeBetween:    400 * time.Millisecond,
			expectDebounce: false,
		},
		{
			name:           "Game Boy button rapid press - should not debounce",
			action:         action.GBButtonA,
			eventType:      event.Press,
			timeBetween:    10 * time.Millisecond,
			expectDebounce: false,
		},
		{
			name:           "UI action release event - should not debounce",
			action:         action.EmulatorDebugToggle,
			eventType:      event.Release,
			timeBetween:    10 * time.Millisecond,
			expectDebounce: false,
		},
		{
			name:           "Hold event type - should not debounce",
			action:         action.EmulatorDebugToggle,
			eventType:      event.Hold,
			timeBetween:    10 * time.Millisecond,
			expectDebounce: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()

			// First event should always go through
			evt1 := backend.InputEvent{
				Action: tt.action,
				Type:   tt.eventType,
			}
			assert.True(t, handler.ProcessEvent(evt1), "First event should always pass")

			// Wait specified time
			time.Sleep(tt.timeBetween)

			// Second event
			evt2 := backend.InputEvent{
				Action: tt.action,
				Type:   tt.eventType,
			}
			result := handler.ProcessEvent(evt2)

			if tt.expectDebounce {
				assert.False(t, result, "Second event should be debounced")
			} else {
				assert.True(t, result, "Second event should not be debounced")
			}
		})
	}
}

func TestHandler_MultipleActions(t *testing.T) {
	handler := NewHandler()

	// Different actions shouldn't interfere with each other
	evt1 := backend.InputEvent{
		Action: action.EmulatorDebugToggle,
		Type:   event.Press,
	}
	evt2 := backend.InputEvent{
		Action: action.EmulatorSnapshot,
		Type:   event.Press,
	}

	assert.True(t, handler.ProcessEvent(evt1), "First debug toggle should pass")
	assert.True(t, handler.ProcessEvent(evt2), "First snapshot should pass")

	// Rapid repeat of first action should be debounced
	assert.False(t, handler.ProcessEvent(evt1), "Rapid debug toggle should be debounced")

	// But the second action can still be repeated rapidly once
	assert.False(t, handler.ProcessEvent(evt2), "Rapid snapshot should be debounced")
}

func TestHandler_HoldEventType(t *testing.T) {
	handler := NewHandler()

	// Hold events should never be debounced according to the event type comments
	evt := backend.InputEvent{
		Action: action.EmulatorDebugToggle,
		Type:   event.Hold,
	}

	// Should always pass through, even in rapid succession
	for i := 0; i < 5; i++ {
		assert.True(t, handler.ProcessEvent(evt), "Hold event should always pass")
	}
}
