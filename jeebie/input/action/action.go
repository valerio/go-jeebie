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

	// Debug controls
	DebugLogLevelIncrease
	DebugLogLevelDecrease
)

// Category represents the category of an action for routing purposes
type Category int

const (
	CategoryGameInput Category = iota // Game Boy hardware controls
	CategoryEmulator                  // Core emulator features
	CategoryBackend                   // Backend-specific features
	CategoryAudio                     // Audio system controls
	CategoryDebug                     // Debug features
)

// ActionInfo contains metadata about an action
type ActionInfo struct {
	Action      Action
	Category    Category
	Debounce    bool // True if the action should only trigger once per key press
	Description string
}

// actionInfoMap contains metadata for all actions
var actionInfoMap = map[Action]ActionInfo{
	// Game Boy hardware controls
	GBButtonA:      {Action: GBButtonA, Category: CategoryGameInput, Debounce: false, Description: "A button"},
	GBButtonB:      {Action: GBButtonB, Category: CategoryGameInput, Debounce: false, Description: "B button"},
	GBButtonStart:  {Action: GBButtonStart, Category: CategoryGameInput, Debounce: false, Description: "Start button"},
	GBButtonSelect: {Action: GBButtonSelect, Category: CategoryGameInput, Debounce: false, Description: "Select button"},
	GBDPadUp:       {Action: GBDPadUp, Category: CategoryGameInput, Debounce: false, Description: "D-Pad Up"},
	GBDPadDown:     {Action: GBDPadDown, Category: CategoryGameInput, Debounce: false, Description: "D-Pad Down"},
	GBDPadLeft:     {Action: GBDPadLeft, Category: CategoryGameInput, Debounce: false, Description: "D-Pad Left"},
	GBDPadRight:    {Action: GBDPadRight, Category: CategoryGameInput, Debounce: false, Description: "D-Pad Right"},

	// Emulator features
	EmulatorDebugToggle:      {Action: EmulatorDebugToggle, Category: CategoryDebug, Debounce: true, Description: "Toggle debug display"},
	EmulatorDebugUpdate:      {Action: EmulatorDebugUpdate, Category: CategoryDebug, Debounce: false, Description: "Update debug display"},
	EmulatorSnapshot:         {Action: EmulatorSnapshot, Category: CategoryBackend, Debounce: true, Description: "Take snapshot"},
	EmulatorPauseToggle:      {Action: EmulatorPauseToggle, Category: CategoryEmulator, Debounce: true, Description: "Toggle pause"},
	EmulatorStepFrame:        {Action: EmulatorStepFrame, Category: CategoryEmulator, Debounce: true, Description: "Step one frame"},
	EmulatorStepInstruction:  {Action: EmulatorStepInstruction, Category: CategoryEmulator, Debounce: true, Description: "Step one instruction"},
	EmulatorTestPatternCycle: {Action: EmulatorTestPatternCycle, Category: CategoryBackend, Debounce: true, Description: "Cycle test patterns"},
	EmulatorQuit:             {Action: EmulatorQuit, Category: CategoryEmulator, Debounce: true, Description: "Quit"},

	// Audio debugging
	AudioToggleChannel1: {Action: AudioToggleChannel1, Category: CategoryAudio, Debounce: true, Description: "Toggle audio channel 1"},
	AudioToggleChannel2: {Action: AudioToggleChannel2, Category: CategoryAudio, Debounce: true, Description: "Toggle audio channel 2"},
	AudioToggleChannel3: {Action: AudioToggleChannel3, Category: CategoryAudio, Debounce: true, Description: "Toggle audio channel 3"},
	AudioToggleChannel4: {Action: AudioToggleChannel4, Category: CategoryAudio, Debounce: true, Description: "Toggle audio channel 4"},
	AudioSoloChannel1:   {Action: AudioSoloChannel1, Category: CategoryAudio, Debounce: true, Description: "Solo audio channel 1"},
	AudioSoloChannel2:   {Action: AudioSoloChannel2, Category: CategoryAudio, Debounce: true, Description: "Solo audio channel 2"},
	AudioSoloChannel3:   {Action: AudioSoloChannel3, Category: CategoryAudio, Debounce: true, Description: "Solo audio channel 3"},
	AudioSoloChannel4:   {Action: AudioSoloChannel4, Category: CategoryAudio, Debounce: true, Description: "Solo audio channel 4"},
	AudioShowStatus:     {Action: AudioShowStatus, Category: CategoryAudio, Debounce: true, Description: "Show audio status"},

	// Debug controls
	DebugLogLevelIncrease: {Action: DebugLogLevelIncrease, Category: CategoryDebug, Debounce: true, Description: "Log level up"},
	DebugLogLevelDecrease: {Action: DebugLogLevelDecrease, Category: CategoryDebug, Debounce: true, Description: "Log level down"},
}

// GetInfo returns metadata for an action
func GetInfo(a Action) ActionInfo {
	if info, ok := actionInfoMap[a]; ok {
		return info
	}
	// Return a default for unknown actions
	return ActionInfo{
		Action:      a,
		Category:    CategoryEmulator,
		Debounce:    false,
		Description: "Unknown action",
	}
}
