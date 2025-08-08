package memory

import (
	"log/slog"
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
	prevButtons uint8  // Previous button state for edge detection
	prevDpad    uint8  // Previous dpad state for edge detection
}

// NewJoypad creates a new Joypad instance
func NewJoypad() *Joypad {
	return &Joypad{
		buttons:     0x0F,
		dpad:        0x0F,
		line:        0x30,  // Default to neither button group selected
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
	
	result := j.line & 0x30  // Keep the selection bits
	
	if (j.line & 0x10) == 0 {
		// Direction pad selected
		result |= j.dpad & 0x0F
	}
	
	if (j.line & 0x20) == 0 {
		// Buttons selected  
		result |= j.buttons & 0x0F
	}
	
	// If both or neither are selected, return high impedance (all 1s)
	if (j.line & 0x30) == 0x30 || (j.line & 0x30) == 0x00 {
		result |= 0x0F
	}
	
	return result
}

// Write sets the joypad line to be read
func (j *Joypad) Write(value uint8) {
	newLine := value & 0x30
	if newLine != j.line {
		groupName := "unknown"
		switch newLine {
		case 0x10:
			groupName = "directions"
		case 0x20:
			groupName = "buttons"
		case 0x30:
			groupName = "none"
		case 0x00:
			groupName = "both"
		}
		slog.Debug("Joypad group selected", "value", newLine, "group", groupName)
	}
	j.line = newLine
}

// logKeyChange logs key press/release events only when state actually changes
func (j *Joypad) logKeyChange(key JoypadKey, pressed bool) {
	keyName := ""
	switch key {
	case JoypadRight:
		keyName = "RIGHT"
	case JoypadLeft:
		keyName = "LEFT"
	case JoypadUp:
		keyName = "UP"
	case JoypadDown:
		keyName = "DOWN"
	case JoypadA:
		keyName = "A"
	case JoypadB:
		keyName = "B"
	case JoypadSelect:
		keyName = "SELECT"
	case JoypadStart:
		keyName = "START"
	}
	
	if pressed {
		slog.Debug("Joypad key pressed", "key", keyName)
	} else {
		slog.Debug("Joypad key released", "key", keyName)
	}
}

// Press updates the joypad state when a key is pressed
func (j *Joypad) Press(key JoypadKey) {
	var wasPressed bool
	
	switch key {
	case JoypadRight:
		wasPressed = (j.dpad & 0x01) == 0
		j.dpad = bit.Reset(0, j.dpad)
	case JoypadLeft:
		wasPressed = (j.dpad & 0x02) == 0
		j.dpad = bit.Reset(1, j.dpad)
	case JoypadUp:
		wasPressed = (j.dpad & 0x04) == 0
		j.dpad = bit.Reset(2, j.dpad)
	case JoypadDown:
		wasPressed = (j.dpad & 0x08) == 0
		j.dpad = bit.Reset(3, j.dpad)
	case JoypadA:
		wasPressed = (j.buttons & 0x01) == 0
		j.buttons = bit.Reset(0, j.buttons)
	case JoypadB:
		wasPressed = (j.buttons & 0x02) == 0
		j.buttons = bit.Reset(1, j.buttons)
	case JoypadSelect:
		wasPressed = (j.buttons & 0x04) == 0
		j.buttons = bit.Reset(2, j.buttons)
	case JoypadStart:
		wasPressed = (j.buttons & 0x08) == 0
		j.buttons = bit.Reset(3, j.buttons)
	}
	
	// Only log if this is a new press (key wasn't already pressed)
	if !wasPressed {
		j.logKeyChange(key, true)
	}
}

// Release updates the joypad state when a key is released
func (j *Joypad) Release(key JoypadKey) {
	var wasPressed bool
	
	switch key {
	case JoypadRight:
		wasPressed = (j.dpad & 0x01) == 0
		j.dpad = bit.Set(0, j.dpad)
	case JoypadLeft:
		wasPressed = (j.dpad & 0x02) == 0
		j.dpad = bit.Set(1, j.dpad)
	case JoypadUp:
		wasPressed = (j.dpad & 0x04) == 0
		j.dpad = bit.Set(2, j.dpad)
	case JoypadDown:
		wasPressed = (j.dpad & 0x08) == 0
		j.dpad = bit.Set(3, j.dpad)
	case JoypadA:
		wasPressed = (j.buttons & 0x01) == 0
		j.buttons = bit.Set(0, j.buttons)
	case JoypadB:
		wasPressed = (j.buttons & 0x02) == 0
		j.buttons = bit.Set(1, j.buttons)
	case JoypadSelect:
		wasPressed = (j.buttons & 0x04) == 0
		j.buttons = bit.Set(2, j.buttons)
	case JoypadStart:
		wasPressed = (j.buttons & 0x08) == 0
		j.buttons = bit.Set(3, j.buttons)
	}
	
	// Only log if the key was actually pressed before (now being released)
	if wasPressed {
		j.logKeyChange(key, false)
	}
}
