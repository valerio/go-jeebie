package memory

import (
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/bit"
)

// tacLookup maps TAC input clock select (bits 1–0) to the bit position
// of the 16‑bit internal divider (systemCounter) used as the timer’s
// clock source. The timer increments on falling edges of this selected
// bit when the timer is enabled (TAC bit 2 = 1).
//
// Mapping per Pan Docs (DMG):
//
//	00 -> bit 9  (4096 Hz)
//	01 -> bit 3  (262144 Hz)
//	10 -> bit 5  (65536 Hz)
//	11 -> bit 7  (16384 Hz)
var tacLookup = [4]uint16{9, 3, 5, 7}

// Timer encapsulates the Game Boy timer/DIV/TIMA/TMA/TAC behavior.
type Timer struct {
	systemCounter     uint16 // Internal 16-bit counter, DIV is upper 8 bits
	lastTimerBitIsSet bool   // Previous state of timer bit for edge detection (set == 1)
	timaOverflow      int    // Cycles remaining in TIMA overflow state

	// Timer registers
	tima byte
	tma  byte
	tac  byte

	// IRQ requester callback
	TimerInterruptHandler func()
}

// SetSeed initializes the internal divider counter and writes DIV accordingly.
func (t *Timer) SetSeed(seed uint16) {
	t.systemCounter = seed
}

func (t *Timer) Tick(cycles int) {
	for range cycles {
		t.systemCounter++

		if t.timaOverflow > 0 {
			// In overflow state, wait 4 cycles before loading TIMA <- TMA and requesting IRQ
			t.timaOverflow--
			if t.timaOverflow == 0 {
				t.tima = t.tma
				t.TimerInterruptHandler()
			}
			continue
		}

		timerEnabled := bit.IsSet(2, t.tac)

		if timerEnabled {
			currentTimerBitIsSet := bit.IsSet16(tacLookup[t.tac&0x03], t.systemCounter)
			if t.lastTimerBitIsSet && !currentTimerBitIsSet {
				t.incrementTIMA()
			}

			t.lastTimerBitIsSet = currentTimerBitIsSet
		} else {
			t.lastTimerBitIsSet = false
		}
	}
}

func (t *Timer) incrementTIMA() {
	oldTima := t.tima
	if t.tima == 0xFF {
		t.timaOverflow = 4
	}
	t.tima++
	slog.Debug("TIMA incremented", "old", oldTima, "new", t.tima, "systemCounter", t.systemCounter)
}

func (t *Timer) Read(address uint16) byte {
	switch address {
	case addr.DIV:
		return byte(t.systemCounter >> 8)
	case addr.TIMA:
		slog.Debug("TIMA read", "value", t.tima, "systemCounter", t.systemCounter)
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
		// DIV writes reset the internal counter to 0, this means one of the bits
		// used for timer input (if enabled) could go from 1 -> 0 (falling edge)
		// We need to detect this and increment TIMA.
		timerEnabled := bit.IsSet(2, t.tac)
		wasSet := bit.IsSet16(tacLookup[t.tac&0x03], t.systemCounter)
		if timerEnabled && wasSet {
			t.incrementTIMA()
		}
		t.systemCounter = 0
		t.lastTimerBitIsSet = false
	case addr.TIMA:
		if t.timaOverflow > 0 {
			t.timaOverflow = 0
		}
		t.tima = value
	case addr.TMA:
		t.tma = value
	case addr.TAC:
		// Writing TAC can also cause a falling edge on the timer input.
		// Similar to DIV, we detect and increment TIMA.
		oldTac, oldEnabled := t.tac, bit.IsSet(2, t.tac)
		newTac, newEnabled := value, bit.IsSet(2, value)

		oldBitWasSet := bit.IsSet16(tacLookup[oldTac&0x03], t.systemCounter)
		newBitIsSet := bit.IsSet16(tacLookup[newTac&0x03], t.systemCounter)

		// If the timer input transitions 1 -> 0 due to this write while it
		// was previously enabled, increment TIMA.
		if oldEnabled && oldBitWasSet && (!newEnabled || !newBitIsSet) {
			t.incrementTIMA()
		}

		t.tac = newTac
		// Resync edge detector to new configuration
		if newEnabled {
			t.lastTimerBitIsSet = newBitIsSet
		} else {
			t.lastTimerBitIsSet = false
		}
	}
}
