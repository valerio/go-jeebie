package headless_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/backend/headless"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
)

func TestHeadlessBackend(t *testing.T) {
	t.Run("normal operation", func(t *testing.T) {
		// Create headless backend for 3 frames
		h := headless.New(3, headless.SnapshotConfig{})

		// Initialize
		config := backend.BackendConfig{
			Title: "Test",
		}
		err := h.Init(config)
		assert.NoError(t, err)

		// Create a test frame
		frame := video.NewFrameBuffer()

		// Run for 3 frames
		for i := 0; i < 3; i++ {
			events, err := h.Update(frame)
			assert.NoError(t, err)

			if i < 2 {
				// Should not quit before reaching max frames
				assert.Empty(t, events)
			} else {
				// Should send quit event on last frame
				assert.Len(t, events, 1)
				assert.Equal(t, action.EmulatorQuit, events[0].Action)
				assert.Equal(t, event.Press, events[0].Type)
			}
		}

		// Cleanup
		err = h.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("test pattern mode", func(t *testing.T) {
		h := headless.New(1, headless.SnapshotConfig{})

		config := backend.BackendConfig{
			Title:       "Test",
			TestPattern: true,
		}
		err := h.Init(config)
		assert.NoError(t, err)

		frame := video.NewFrameBuffer()

		// Should quit immediately in test pattern mode
		events, err := h.Update(frame)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, action.EmulatorQuit, events[0].Action)

		err = h.Cleanup()
		assert.NoError(t, err)
	})
}

func TestHeadlessImplementsBackend(t *testing.T) {
	// Compile-time check that headless.Backend implements backend.Backend
	var _ backend.Backend = (*headless.Backend)(nil)
}
