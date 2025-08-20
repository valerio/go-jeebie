package memory

import (
	"github.com/valerio/go-jeebie/jeebie/bit"
)

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
	buttons     uint8
	dpad        uint8
	line        uint8
	prevButtons uint8 // Previous button state for edge detection
	prevDpad    uint8 // Previous dpad state for edge detection
}

// NewJoypad creates a new Joypad instance
func NewJoypad() *Joypad {
	return &Joypad{
		buttons:     0x0F,
		dpad:        0x0F,
		line:        0x30, // Default to neither button group selected
		prevButtons: 0x0F,
		prevDpad:    0x0F,
	}
}

// Read returns the current state of the joypad
func (j *Joypad) Read() uint8 {
	// Bits 4-5 are the selection lines (input from CPU)
	// Bits 0-3 are the button states (output to CPU)
	// When bit 4 is 0: read direction pad
	// When bit 5 is 0: read buttons

	result := j.line & 0x30 // Keep the selection bits

	if (j.line & 0x10) == 0 {
		// Direction pad selected
		result |= j.dpad & 0x0F
	}

	if (j.line & 0x20) == 0 {
		// Buttons selected
		result |= j.buttons & 0x0F
	}

	// If both or neither are selected, return high impedance (all 1s)
	if (j.line&0x30) == 0x30 || (j.line&0x30) == 0x00 {
		result |= 0x0F
	}

	return result
}

// Write sets the joypad line to be read
func (j *Joypad) Write(value uint8) {
	newLine := value & 0x30
	j.line = newLine
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
