package action

// Action represents input actions that can be performed in the emulator
type Action int

const (
	// Game Boy hardware controls
	GBButtonA Action = iota
	GBButtonB
	GBButtonStart
	GBButtonSelect
	GBDPadUp
	GBDPadDown
	GBDPadLeft
	GBDPadRight

	// Emulator features
	EmulatorDebugToggle
	EmulatorDebugUpdate
	EmulatorSnapshot
	EmulatorPauseToggle
	EmulatorStepFrame
	EmulatorStepInstruction
	EmulatorTestPatternCycle
	EmulatorQuit

	// Audio debugging
	AudioToggleChannel1
	AudioToggleChannel2
	AudioToggleChannel3
	AudioToggleChannel4
	AudioSoloChannel1
	AudioSoloChannel2
	AudioSoloChannel3
	AudioSoloChannel4
	AudioShowStatus
)
