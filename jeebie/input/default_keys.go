package input

import "github.com/valerio/go-jeebie/jeebie/input/action"

// DefaultKeyMap provides default key mappings that work across backends.
// Backends can use these mappings as a base and override/extend as needed.
var DefaultKeyMap = map[string]action.Action{
	// Game Boy controls
	"z":      action.GBButtonA,
	"x":      action.GBButtonB,
	"Enter":  action.GBButtonStart,
	"Shift":  action.GBButtonSelect,
	"Select": action.GBButtonSelect,
	"Up":     action.GBDPadUp,
	"Down":   action.GBDPadDown,
	"Left":   action.GBDPadLeft,
	"Right":  action.GBDPadRight,

	// Alternative arrow keys (WASD)
	"w": action.GBDPadUp,
	"s": action.GBDPadDown,
	"a": action.GBDPadLeft,
	"d": action.GBDPadRight,

	// Emulator controls
	"Space":  action.EmulatorPauseToggle,
	"p":      action.EmulatorPauseToggle, // Alternative key
	"r":      action.EmulatorPauseToggle, // Alternative key for pause/resume
	"o":      action.EmulatorStepFrame,
	"f":      action.EmulatorStepFrame, // Alternative key for step frame
	"i":      action.EmulatorStepInstruction,
	"n":      action.EmulatorStepInstruction, // Alternative key for step instruction
	"F9":     action.EmulatorSnapshot,
	"F10":    action.EmulatorDebugToggle,
	"F11":    action.EmulatorDebugUpdate,
	"F12":    action.EmulatorTestPatternCycle,
	"Escape": action.EmulatorQuit,
	"q":      action.EmulatorQuit,

	// Audio debug controls
	"F1": action.AudioToggleChannel1,
	"F2": action.AudioToggleChannel2,
	"F3": action.AudioToggleChannel3,
	"F4": action.AudioToggleChannel4,
	"1":  action.AudioSoloChannel1,
	"2":  action.AudioSoloChannel2,
	"3":  action.AudioSoloChannel3,
	"4":  action.AudioSoloChannel4,
	"F5": action.AudioShowStatus,

	// Debug controls
	"+": action.DebugLogLevelIncrease,
	"=": action.DebugLogLevelIncrease, // Alternative without shift
	"-": action.DebugLogLevelDecrease,
	"_": action.DebugLogLevelDecrease, // Alternative with shift
}

// GetDefaultMapping returns the default action for a key, if one exists
func GetDefaultMapping(key string) (action.Action, bool) {
	act, ok := DefaultKeyMap[key]
	return act, ok
}
