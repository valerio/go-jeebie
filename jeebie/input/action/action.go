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
)
