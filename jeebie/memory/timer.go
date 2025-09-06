package memory

import (
	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// Timer encapsulates the Game Boy timer/DIV/TIMA/TMA/TAC behavior.
type Timer struct {
	systemCounter uint16 // Internal 16-bit counter, DIV is upper 8 bits
	lastTimerBit  bool   // Previous state of timer bit for edge detection
	timaOverflow  int    // Cycles remaining in TIMA overflow state
	timaDelayInt  bool   // Delayed interrupt flag setting (1 M-cycle after TMA load)

	// Timer registers
	div  byte
	tima byte
	tma  byte
	tac  byte

	// IRQ requester callback
	TimerInterruptHandler func()
}

// SetSeed initializes the internal divider counter and writes DIV accordingly.
func (t *Timer) SetSeed(seed uint16) {
	t.systemCounter = seed
	t.lastTimerBit = false
	t.timaOverflow = 0
	t.timaDelayInt = false
	t.div = byte(t.systemCounter >> 8)
}

// Tick advances the timer by the specified number of CPU cycles.
// Logic matches the previous DMG.updateTimers implementation.
func (t *Timer) Tick(cycles int) {
	if t.timaDelayInt {
		if t.TimerInterruptHandler != nil {
			t.TimerInterruptHandler()
		}
		t.timaDelayInt = false
	}

	if t.timaOverflow > 0 {
		t.timaOverflow -= cycles
		if t.timaOverflow <= 0 {
			t.tima = t.tma
			t.timaDelayInt = true
			t.timaOverflow = 0
		}
	}

	for range cycles {
		t.systemCounter++
		t.div = byte(t.systemCounter >> 8)

		if t.timaOverflow > 0 {
			continue
		}

		tac := t.tac
		timerEnabled := (tac & 0x04) != 0

		if timerEnabled {
			var bitPosition uint16
			switch tac & 0x03 {
			case 0x00:
				bitPosition = 9
			case 0x01:
				bitPosition = 3
			case 0x02:
				bitPosition = 5
			case 0x03:
				bitPosition = 7
			}

            currentTimerBit := bit.IsSet16(bitPosition, t.systemCounter)

			if t.lastTimerBit && !currentTimerBit {
				if t.tima == 0xFF {
					t.tima = 0x00
					t.timaOverflow = 4
				} else {
					t.tima++
				}
			}

			t.lastTimerBit = currentTimerBit
		} else {
			t.lastTimerBit = false
		}
	}
}

func (t *Timer) Read(address uint16) byte {
	switch address {
	case addr.DIV:
		return t.div
	case addr.TIMA:
		return t.tima
	case addr.TMA:
		return t.tma
	case addr.TAC:
		return t.tac
	default:
		return 0xFF
	}
}

func (t *Timer) Write(address uint16, value byte) {
	switch address {
	case addr.DIV:
		// Writing to DIV resets the divider, upper byte becomes 0
		t.systemCounter = 0
		t.div = 0
	case addr.TIMA:
		t.tima = value
	case addr.TMA:
		t.tma = value
	case addr.TAC:
		t.tac = value
	}
}
