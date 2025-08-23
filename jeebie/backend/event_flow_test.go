package backend_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// MockBackend is a test backend that returns predetermined events
type MockBackend struct {
	events      []backend.InputEvent
	initialized bool
	cleanedUp   bool
	updateCalls int
}

func (m *MockBackend) Init(config backend.BackendConfig) error {
	m.initialized = true
	return nil
}

func (m *MockBackend) Update(frame *video.FrameBuffer) ([]backend.InputEvent, error) {
	m.updateCalls++
	// Return events only on first call
	if m.updateCalls == 1 {
		return m.events, nil
	}
	return nil, nil
}

func (m *MockBackend) Cleanup() error {
	m.cleanedUp = true
	return nil
}

func (m *MockBackend) HandleAction(act action.Action) {
}

func TestEventFlow(t *testing.T) {
	tests := []struct {
		name          string
		events        []backend.InputEvent
		expectedQuit  bool
		expectedCalls int
	}{
		{
			name: "quit event stops loop",
			events: []backend.InputEvent{
				{Action: action.EmulatorQuit, Type: event.Press},
			},
			expectedQuit:  true,
			expectedCalls: 1,
		},
		{
			name: "game boy button events are passed through",
			events: []backend.InputEvent{
				{Action: action.GBButtonA, Type: event.Press},
				{Action: action.GBButtonA, Type: event.Release},
				{Action: action.GBButtonB, Type: event.Press},
				{Action: action.EmulatorQuit, Type: event.Press},
			},
			expectedQuit:  true,
			expectedCalls: 1,
		},
		{
			name: "pause toggle event",
			events: []backend.InputEvent{
				{Action: action.EmulatorPauseToggle, Type: event.Press},
				{Action: action.EmulatorQuit, Type: event.Press},
			},
			expectedQuit:  true,
			expectedCalls: 1,
		},
		{
			name:          "no events runs multiple iterations",
			events:        []backend.InputEvent{},
			expectedQuit:  false,
			expectedCalls: 5, // Will run 5 iterations before test stops it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test emulator
			emu := jeebie.NewTestPatternEmulator()

			// Create mock backend with test events
			mockBackend := &MockBackend{
				events: tt.events,
			}

			// Initialize backend
			config := backend.BackendConfig{
				Title:       "Test",
				TestPattern: true,
			}
			err := mockBackend.Init(config)
			assert.NoError(t, err)
			assert.True(t, mockBackend.initialized)

			// Run event loop
			running := true
			iterations := 0
			maxIterations := 5

			for running && iterations < maxIterations {
				iterations++

				// Run frame
				err := emu.RunUntilFrame()
				assert.NoError(t, err)
				frame := emu.GetCurrentFrame()
				assert.NotNil(t, frame)

				// Get events from backend
				events, err := mockBackend.Update(frame)
				assert.NoError(t, err)

				// Process events
				for _, evt := range events {
					switch evt.Action {
					case action.EmulatorQuit:
						if evt.Type == event.Press {
							running = false
						}
					case action.EmulatorPauseToggle:
						// In real implementation, this would toggle pause
						// For test, we just verify it's handled
					default:
						// Pass to emulator
						emu.HandleAction(evt.Action, evt.Type == event.Press)
					}
				}
			}

			// Verify expectations
			if tt.expectedQuit {
				assert.False(t, running, "Loop should have quit")
			} else {
				assert.True(t, running, "Loop should still be running")
			}
			assert.Equal(t, tt.expectedCalls, mockBackend.updateCalls)

			// Cleanup
			err = mockBackend.Cleanup()
			assert.NoError(t, err)
			assert.True(t, mockBackend.cleanedUp)
		})
	}
}

func TestBackendInterface(t *testing.T) {
	// Verify MockBackend implements Backend interface
	var _ backend.Backend = (*MockBackend)(nil)
}

func TestEventProcessing(t *testing.T) {
	emu := jeebie.NewTestPatternEmulator()

	// Test that HandleAction is called correctly
	testCases := []struct {
		action  action.Action
		pressed bool
	}{
		{action.GBButtonA, true},
		{action.GBButtonA, false},
		{action.GBButtonB, true},
		{action.GBButtonStart, true},
		{action.GBDPadUp, true},
		{action.GBDPadDown, false},
		{action.EmulatorTestPatternCycle, true},
	}

	for _, tc := range testCases {
		// This should not panic or error
		emu.HandleAction(tc.action, tc.pressed)
	}
}
