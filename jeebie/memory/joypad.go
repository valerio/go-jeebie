package memory

import "github.com/valerio/go-jeebie/jeebie/bit"

// JoypadKey represents a key on the Gameboy joypad
type JoypadKey uint8

const (
	JoypadRight JoypadKey = iota
	JoypadLeft
	JoypadUp
	JoypadDown
	JoypadA
	JoypadB
	JoypadSelect
	JoypadStart
)

// Joypad represents the Gameboy joypad
type Joypad struct {
	buttons uint8
	dpad    uint8
	line    uint8
}

// NewJoypad creates a new Joypad instance
func NewJoypad() *Joypad {
	return &Joypad{
		buttons: 0x0F,
		dpad:    0x0F,
	}
}

// Read returns the current state of the joypad
func (j *Joypad) Read() uint8 {
	switch j.line {
	case 0x10:
		return j.dpad
	case 0x20:
		return j.buttons
	default:
		return 0
	}
}

// Write sets the joypad line to be read
func (j *Joypad) Write(value uint8) {
	j.line = value & 0x30
}

// Press updates the joypad state when a key is pressed
func (j *Joypad) Press(key JoypadKey) {
	switch key {
	case JoypadRight:
		j.dpad = bit.Reset(0, j.dpad)
	case JoypadLeft:
		j.dpad = bit.Reset(1, j.dpad)
	case JoypadUp:
		j.dpad = bit.Reset(2, j.dpad)
	case JoypadDown:
		j.dpad = bit.Reset(3, j.dpad)
	case JoypadA:
		j.buttons = bit.Reset(0, j.buttons)
	case JoypadB:
		j.buttons = bit.Reset(1, j.buttons)
	case JoypadSelect:
		j.buttons = bit.Reset(2, j.buttons)
	case JoypadStart:
		j.buttons = bit.Reset(3, j.buttons)
	}
}

// Release updates the joypad state when a key is released
func (j *Joypad) Release(key JoypadKey) {
	switch key {
	case JoypadRight:
		j.dpad = bit.Set(0, j.dpad)
	case JoypadLeft:
		j.dpad = bit.Set(1, j.dpad)
	case JoypadUp:
		j.dpad = bit.Set(2, j.dpad)
	case JoypadDown:
		j.dpad = bit.Set(3, j.dpad)
	case JoypadA:
		j.buttons = bit.Set(0, j.buttons)
	case JoypadB:
		j.buttons = bit.Set(1, j.buttons)
	case JoypadSelect:
		j.buttons = bit.Set(2, j.buttons)
	case JoypadStart:
		j.buttons = bit.Set(3, j.buttons)
	}
}
