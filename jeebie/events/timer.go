package events

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

const CPU_CLOCK_HZ = 4194304 // 4.194304 MHz

// Timer frequencies in M-cycles per increment (corrected from current implementation)
var timerCyclePeriods = map[uint8]int{
	0: 1024, // 4096 Hz   - CPU_CLOCK / 1024
	1: 16,   // 262144 Hz - CPU_CLOCK / 16
	2: 64,   // 65536 Hz  - CPU_CLOCK / 64
	3: 256,  // 16384 Hz  - CPU_CLOCK / 256
}

// DIV register increments every 256 cycles (16384 Hz)
const DIV_PERIOD = 256

// TimerOverflowState tracks the 3-cycle overflow sequence
type TimerOverflowState struct {
	Active         bool
	OverflowCycle  uint64
	LoadCycle      uint64
	InterruptCycle uint64
}

// EventDrivenTimer implements cycle-accurate Game Boy timer using events
type EventDrivenTimer struct {
	// Memory interface for register access
	memory *memory.MMU

	// Timer state
	systemCounter    uint16 // Internal counter for DIV register
	nextDivIncrement uint64 // When to increment DIV next
	nextTimerTick    uint64 // When TIMA should increment next
	overflowState    TimerOverflowState

	// Register write detection
	tacChanged  bool
	timaWritten bool
	divReset    bool
}

// NewEventDrivenTimer creates a new event-driven timer system
func NewEventDrivenTimer(memory *memory.MMU) *EventDrivenTimer {
	return &EventDrivenTimer{
		memory:           memory,
		systemCounter:    0,
		nextDivIncrement: DIV_PERIOD,
		nextTimerTick:    0,
		overflowState:    TimerOverflowState{},
	}
}

// ScheduleEvents schedules timer-related events for the next instruction cycles
func (t *EventDrivenTimer) ScheduleEvents(scheduler *EventScheduler, cycles int) {
	currentCycle := scheduler.GetCurrentCycle()
	endCycle := currentCycle + uint64(cycles)

	// Schedule DIV register increments
	for t.nextDivIncrement <= endCycle {
		scheduler.Schedule(EventType(100), t.nextDivIncrement, "div_increment") // Custom event type
		t.nextDivIncrement += DIV_PERIOD
	}

	// Schedule TIMA increments if timer is enabled
	if t.isTimerEnabled() {
		frequency := t.getTimerFrequency()

		// Calculate next TIMA tick if not set
		if t.nextTimerTick == 0 {
			t.nextTimerTick = currentCycle + uint64(frequency)
		}

		// Schedule all TIMA ticks within this instruction's cycles
		for t.nextTimerTick <= endCycle {
			scheduler.Schedule(TimerTick, t.nextTimerTick, nil)
			t.nextTimerTick += uint64(frequency)
		}
	}

	// Schedule overflow sequence events if active
	if t.overflowState.Active {
		if t.overflowState.LoadCycle <= endCycle {
			scheduler.Schedule(TimerReload, t.overflowState.LoadCycle, nil)
		}
		if t.overflowState.InterruptCycle <= endCycle {
			scheduler.Schedule(TimerInterrupt, t.overflowState.InterruptCycle, nil)
		}
	}
}

// ProcessDivIncrement handles DIV register increment events
func (t *EventDrivenTimer) ProcessDivIncrement() {
	t.systemCounter++
	currentDiv := t.memory.Read(addr.DIV)
	newDiv := currentDiv + 1
	t.memory.Write(addr.DIV, newDiv)

	if currentDiv <= 5 || currentDiv%64 == 0 { // Log early DIV increments and periodic updates
		slog.Debug("DIV increment", "old", fmt.Sprintf("0x%02X", currentDiv), "new", fmt.Sprintf("0x%02X", newDiv), "system_counter", t.systemCounter)
	}
}

// ProcessTimerTick handles TIMA increment events with overflow detection
func (t *EventDrivenTimer) ProcessTimerTick(scheduler *EventScheduler) {
	if !t.isTimerEnabled() {
		return
	}

	currentTima := t.memory.Read(addr.TIMA)
	tac := t.memory.Read(addr.TAC)

	if currentTima == 0xFF {
		// Start overflow sequence
		currentCycle := scheduler.GetCurrentCycle()
		tma := t.memory.Read(addr.TMA)

		slog.Debug("TIMA overflow starting",
			"cycle", currentCycle,
			"tima", fmt.Sprintf("0x%02X", currentTima),
			"tma", fmt.Sprintf("0x%02X", tma),
			"tac", fmt.Sprintf("0x%02X", tac))

		t.overflowState = TimerOverflowState{
			Active:         true,
			OverflowCycle:  currentCycle,
			LoadCycle:      currentCycle + 1, // TMA loaded after 1 cycle
			InterruptCycle: currentCycle + 2, // Interrupt fired after 2 cycles
		}

		// TIMA becomes 0x00 immediately
		t.memory.Write(addr.TIMA, 0x00)

		// Schedule the reload and interrupt events
		scheduler.Schedule(TimerReload, t.overflowState.LoadCycle, nil)
		scheduler.Schedule(TimerInterrupt, t.overflowState.InterruptCycle, nil)

	} else {
		// Normal increment
		newTima := currentTima + 1
		t.memory.Write(addr.TIMA, newTima)

		if currentTima <= 5 || currentTima%32 == 0 || newTima >= 0xF0 { // Log early, periodic, and near-overflow TIMA values
			frequency := t.getTimerFrequency()
			slog.Debug("TIMA increment",
				"old", fmt.Sprintf("0x%02X", currentTima),
				"new", fmt.Sprintf("0x%02X", newTima),
				"tac", fmt.Sprintf("0x%02X", tac),
				"frequency", frequency,
				"cycle", scheduler.GetCurrentCycle())
		}
	}
}

// ProcessTimerReload handles the TMA -> TIMA load during overflow
func (t *EventDrivenTimer) ProcessTimerReload() {
	if !t.overflowState.Active {
		return
	}

	tma := t.memory.Read(addr.TMA)
	t.memory.Write(addr.TIMA, tma)

	slog.Debug("TIMA reload from TMA",
		"tma_value", fmt.Sprintf("0x%02X", tma),
		"load_cycle", t.overflowState.LoadCycle)
}

// ProcessTimerInterrupt handles timer interrupt generation
func (t *EventDrivenTimer) ProcessTimerInterrupt() {
	if !t.overflowState.Active {
		return
	}

	// Request timer interrupt
	t.memory.RequestInterrupt(addr.TimerInterrupt)

	slog.Debug("Timer interrupt requested",
		"interrupt_cycle", t.overflowState.InterruptCycle,
		"IF", fmt.Sprintf("0x%02X", t.memory.Read(0xFF0F)))

	// Clear overflow state
	t.overflowState.Active = false
}

// HandleTimaWrite handles edge cases when TIMA is written during overflow
func (t *EventDrivenTimer) HandleTimaWrite(scheduler *EventScheduler, value uint8) {
	currentCycle := scheduler.GetCurrentCycle()

	if t.overflowState.Active {
		if currentCycle == t.overflowState.OverflowCycle {
			// Writing during overflow cycle prevents TMA load and interrupt
			t.overflowState.Active = false
		} else if currentCycle == t.overflowState.LoadCycle {
			// Writing during load cycle is ignored (hardware quirk)
			return
		}
	}

	// Reset timer tick scheduling due to manual TIMA write
	t.nextTimerTick = currentCycle + uint64(t.getTimerFrequency())
}

// HandleTacWrite handles TAC register writes that can trigger immediate ticks
func (t *EventDrivenTimer) HandleTacWrite(scheduler *EventScheduler, oldValue, newValue uint8) {
	// Check if timer enable bit changed
	wasEnabled := (oldValue & 0x04) != 0
	nowEnabled := (newValue & 0x04) != 0

	// Check if frequency changed
	oldFreq := oldValue & 0x03
	newFreq := newValue & 0x03

	if nowEnabled {
		// Recalculate next timer tick with new frequency
		t.nextTimerTick = scheduler.GetCurrentCycle() + uint64(t.getTimerFrequency())
	} else {
		// Timer disabled, clear next tick
		t.nextTimerTick = 0
	}

	// Edge case: changing frequency or disabling can trigger immediate tick
	if wasEnabled && (oldFreq != newFreq || !nowEnabled) {
		// This is a complex hardware behavior - simplified for now
		// In real hardware, this involves falling edge detection on system counter bits
	}
}

// HandleDivWrite handles DIV register writes (always resets to 0)
func (t *EventDrivenTimer) HandleDivWrite(scheduler *EventScheduler) {
	// Reset DIV to 0
	t.memory.Write(addr.DIV, 0x00)

	// Reset system counter and reschedule DIV increments
	t.systemCounter = 0
	t.nextDivIncrement = scheduler.GetCurrentCycle() + DIV_PERIOD

	// Writing to DIV can also affect timer due to system counter reset
	// This can cause immediate timer tick - hardware quirk
}

// Helper functions
func (t *EventDrivenTimer) isTimerEnabled() bool {
	tac := t.memory.Read(addr.TAC)
	return (tac & 0x04) != 0
}

func (t *EventDrivenTimer) getTimerFrequency() int {
	tac := t.memory.Read(addr.TAC)
	freqSelect := tac & 0x03
	return timerCyclePeriods[freqSelect]
}
