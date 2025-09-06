package memory

import (
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
	systemCounter uint16 // Internal 16-bit counter, DIV is upper 8 bits
	lastTimerBit  bool   // Previous state of timer bit for edge detection
	timaOverflow  int    // Cycles remaining in TIMA overflow state
	timaDelayInt  bool   // Delayed interrupt flag setting (1 M-cycle after TMA load)

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
	t.lastTimerBit = false
	t.timaOverflow = 0
	t.timaDelayInt = false
}

func (t *Timer) Tick(cycles int) {
	for range cycles {
		if t.timaDelayInt {
			if t.TimerInterruptHandler != nil {
				t.TimerInterruptHandler()
			}
			t.timaDelayInt = false
		}

		t.systemCounter++

		if t.timaOverflow > 0 {
			// In overflow state, wait 4 cycles before loading TIMA <- TMA and requesting IRQ
			t.timaOverflow--
			if t.timaOverflow == 0 {
				t.tima = t.tma
				t.timaDelayInt = true
			}
			continue
		}

		timerEnabled := bit.IsSet(2, t.tac)

		if timerEnabled {
			currentTimerBit := bit.IsSet16(tacLookup[t.tac&0x03], t.systemCounter)

			if t.lastTimerBit && !currentTimerBit {
				t.incrementTIMA()
			}

			t.lastTimerBit = currentTimerBit
		} else {
			t.lastTimerBit = false
		}
	}
}

func (t *Timer) incrementTIMA() {
	if t.tima == 0xFF {
		t.timaOverflow = 4
	}
	t.tima++
}

func (t *Timer) Read(address uint16) byte {
	switch address {
	case addr.DIV:
		return byte(t.systemCounter >> 8)
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
		t.systemCounter = 0 // DIV writes reset the counter
	case addr.TIMA:
		t.tima = value
	case addr.TMA:
		t.tma = value
	case addr.TAC:
		t.tac = value
	}
}
