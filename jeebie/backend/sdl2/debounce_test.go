//go:build sdl2

package sdl2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
)

func TestSDL2Backend_DebugToggleDebouncing(t *testing.T) {
	// Create backend
	b := New()

	// Initialize
	err := b.Init(backend.BackendConfig{
		Title: "Test",
		Scale: 1,
	})
	require.NoError(t, err)
	defer b.Cleanup()

	// Create input handler for debouncing
	handler := input.NewHandler()

	// Simulate rapid F11 key presses
	toggleCount := 0
	originalToggle := b.ToggleDebugWindow
	b.ToggleDebugWindow = func() {
		toggleCount++
		originalToggle()
	}

	// Create a test frame
	frame := video.NewFrameBuffer()

	// Note: Without an event channel, we can't easily simulate SDL events
	// This test now verifies that the backend initializes and the input handler debounces correctly
	for i := 0; i < 5; i++ {
		// Create a simulated event
		testEvent := backend.InputEvent{
			Action: action.EmulatorDebugToggle,
			Type:   event.Press,
		}

		// Process through Update (no events without real SDL input)
		events, err := b.Update(frame)
		require.NoError(t, err)
		assert.Empty(t, events, "No events without SDL input")

		// Test debouncing with our simulated event
		if i == 0 {
			assert.True(t, handler.ProcessEvent(testEvent), "First press should be processed")
		} else {
			assert.False(t, handler.ProcessEvent(testEvent), "Rapid presses should be debounced")
		}

		// Small delay between presses (less than debounce time)
		time.Sleep(50 * time.Millisecond)
	}

	// Test that the toggle count remains as expected
	assert.Equal(t, 0, toggleCount, "No toggles without actual SDL events")
}

func TestSDL2Backend_EventFlow(t *testing.T) {
	// Create backend
	b := New()

	// Initialize
	err := b.Init(backend.BackendConfig{
		Title: "Test",
		Scale: 1,
	})
	require.NoError(t, err)
	defer b.Cleanup()

	// Create a test frame
	frame := video.NewFrameBuffer()

	// Without an event channel, we can't inject events directly
	// Just verify that Update works without errors

	// Update should work without errors
	events, err := b.Update(frame)
	require.NoError(t, err)

	// No events without actual SDL input
	assert.Empty(t, events, "No events without SDL input")
}
