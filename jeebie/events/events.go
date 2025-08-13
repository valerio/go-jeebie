package events

// EventType represents the different types of events in the Game Boy system
type EventType int

const (
	CPUInstruction EventType = iota
	TimerTick
	TimerOverflow
	TimerReload
	TimerInterrupt
	VBlankStart
	VBlankEnd
	HBlankStart
	PPUModeChange
)

// GameBoyEvent represents a scheduled event in the emulator
type GameBoyEvent struct {
	Cycle     uint64      // Absolute cycle when this event should fire
	EventType EventType   // Type of event
	Data      interface{} // Optional event-specific data
}

// EventScheduler manages the event queue and scheduling
type EventScheduler struct {
	events       chan GameBoyEvent
	currentCycle uint64
	running      bool
}

// NewEventScheduler creates a new event scheduler with a buffered channel
func NewEventScheduler(bufferSize int) *EventScheduler {
	return &EventScheduler{
		events:       make(chan GameBoyEvent, bufferSize),
		currentCycle: 0,
		running:      false,
	}
}

// Schedule adds an event to the event queue
func (s *EventScheduler) Schedule(eventType EventType, cycle uint64, data interface{}) {
	if !s.running {
		return
	}

	select {
	case s.events <- GameBoyEvent{
		Cycle:     cycle,
		EventType: eventType,
		Data:      data,
	}:
		// Event scheduled successfully
	default:
		// Channel buffer full - this shouldn't happen with proper sizing
		panic("Event queue overflow - increase buffer size")
	}
}

// ScheduleRelative schedules an event relative to the current cycle
func (s *EventScheduler) ScheduleRelative(eventType EventType, cyclesFromNow uint64, data interface{}) {
	s.Schedule(eventType, s.currentCycle+cyclesFromNow, data)
}

// GetNextEvent returns the next event from the queue, blocking if none available
func (s *EventScheduler) GetNextEvent() (GameBoyEvent, bool) {
	if !s.running {
		return GameBoyEvent{}, false
	}

	select {
	case event := <-s.events:
		return event, true
	default:
		return GameBoyEvent{}, false
	}
}

// Start begins event processing
func (s *EventScheduler) Start() {
	s.running = true
}

// Stop halts event processing and drains the queue
func (s *EventScheduler) Stop() {
	s.running = false

	// Drain any remaining events
	for {
		select {
		case <-s.events:
			// Continue draining
		default:
			return
		}
	}
}

// GetCurrentCycle returns the current cycle count
func (s *EventScheduler) GetCurrentCycle() uint64 {
	return s.currentCycle
}

// SetCurrentCycle updates the current cycle count
func (s *EventScheduler) SetCurrentCycle(cycle uint64) {
	s.currentCycle = cycle
}

// EventCount returns the number of pending events
func (s *EventScheduler) EventCount() int {
	return len(s.events)
}
