package backend_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/backend/headless"
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// TestDebouncing verifies that the debouncing flow works correctly:
// Backend -> Events -> InputHandler (debounce) -> Actions
func TestDebouncing(t *testing.T) {
	// Create a simple test backend that just returns queued events
	type testBackend struct {
		eventQueue []backend.InputEvent
	}

	tb := &testBackend{}

	// Simulate rapid button presses
	for i := 0; i < 5; i++ {
		tb.eventQueue = append(tb.eventQueue, backend.InputEvent{
			Action: action.EmulatorPauseToggle,
			Type:   event.Press,
		})
	}

	// Create input handler for debouncing
	handler := input.NewHandler()

	// Process events
	processedCount := 0
	for i, evt := range tb.eventQueue {
		if handler.ProcessEvent(evt) {
			processedCount++
		}

		// Only first event should pass through
		if i == 0 {
			assert.True(t, handler.ProcessEvent(evt) == false, "Same event immediately should be debounced")
		}
	}

	// Only one event should have been processed due to debouncing
	assert.Equal(t, 1, processedCount, "Only first press should be processed, rest debounced")
}

// TestDebouncingWithDelay verifies debouncing respects time delays
func TestDebouncingWithDelay(t *testing.T) {
	handler := input.NewHandler()

	evt := backend.InputEvent{
		Action: action.EmulatorPauseToggle,
		Type:   event.Press,
	}

	// First press should go through
	assert.True(t, handler.ProcessEvent(evt), "First press should pass")

	// Immediate second press should be debounced
	assert.False(t, handler.ProcessEvent(evt), "Immediate press should be debounced")

	// Wait for debounce period
	time.Sleep(350 * time.Millisecond)

	// Now it should go through again
	assert.True(t, handler.ProcessEvent(evt), "Press after debounce period should pass")
}

// TestHeadlessWithDebouncing tests the headless backend with input handler
func TestHeadlessWithDebouncing(t *testing.T) {
	b := headless.New(3, headless.SnapshotConfig{})

	err := b.Init(backend.BackendConfig{
		Title: "Test",
	})
	require.NoError(t, err)
	defer b.Cleanup()

	handler := input.NewHandler()
	frame := video.NewFrameBuffer()

	// Headless backend doesn't generate events, so we test it returns empty
	for i := 0; i < 3; i++ {
		events, err := b.Update(frame)
		require.NoError(t, err)

		// Process any events through handler
		for _, evt := range events {
			handler.ProcessEvent(evt)
		}

		// Headless generates a quit event on the last frame
		if i == 2 {
			assert.Len(t, events, 1, "Should have quit event on last frame")
			assert.Equal(t, action.EmulatorQuit, events[0].Action)
		} else {
			assert.Empty(t, events, "No events on non-final frames")
		}
	}
}
