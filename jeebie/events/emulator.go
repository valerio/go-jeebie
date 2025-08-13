package events

import (
	"fmt"
	"log/slog"

	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// EventDrivenEmulator implements cycle-accurate emulation using an event loop
type EventDrivenEmulator struct {
	// Core components
	cpu *cpu.CPU
	gpu *video.GPU
	mem *memory.MMU

	// Event system
	scheduler *EventScheduler
	timer     *EventDrivenTimer

	// State tracking
	frameCount       uint64
	instructionCount uint64
	running          bool

	// Performance monitoring
	eventsProcessed uint64
}

// NewEventDrivenEmulator creates a new event-driven emulator
func NewEventDrivenEmulator(mem *memory.MMU) *EventDrivenEmulator {
	scheduler := NewEventScheduler(2048) // Large buffer for performance

	emulator := &EventDrivenEmulator{
		cpu:       cpu.New(mem),
		gpu:       video.NewGpu(mem),
		mem:       mem,
		scheduler: scheduler,
		running:   false,
	}

	// Create event-driven timer with memory interface
	emulator.timer = NewEventDrivenTimer(mem)

	return emulator
}

// RunEventLoop executes the main event-driven emulation loop
func (e *EventDrivenEmulator) RunEventLoop(maxFrames int) {
	e.scheduler.Start()
	e.running = true

	slog.Info("Starting event-driven emulation", "max_frames", maxFrames)

	// Bootstrap with first CPU instruction event
	e.scheduler.Schedule(CPUInstruction, 0, nil)

	for e.running && (maxFrames == 0 || int(e.frameCount) < maxFrames) {
		event, hasEvent := e.scheduler.GetNextEvent()

		if !hasEvent {
			// No events pending - this shouldn't happen in normal operation
			slog.Warn("No events pending - emulation stalled")
			break
		}

		// Update current cycle to event time
		e.scheduler.SetCurrentCycle(event.Cycle)

		// Process the event
		e.processEvent(event)
		e.eventsProcessed++

		// Log progress periodically
		if e.instructionCount > 0 && e.instructionCount%100000 == 0 {
			slog.Debug("Event loop progress",
				"instructions", e.instructionCount,
				"cycles", event.Cycle,
				"events_processed", e.eventsProcessed,
				"frame", e.frameCount)
		}
	}

	e.scheduler.Stop()
	slog.Info("Event-driven emulation completed",
		"frames", e.frameCount,
		"instructions", e.instructionCount,
		"events_processed", e.eventsProcessed)
}

// processEvent handles a single event
func (e *EventDrivenEmulator) processEvent(event GameBoyEvent) {
	switch event.EventType {
	case CPUInstruction:
		e.processCPUInstruction()

	case TimerTick:
		e.timer.ProcessTimerTick(e.scheduler)

	case TimerReload:
		e.timer.ProcessTimerReload()

	case TimerInterrupt:
		e.timer.ProcessTimerInterrupt()

	case VBlankStart:
		e.processVBlankStart()

	default:
		// Handle custom event types (like DIV increment)
		if event.EventType == EventType(100) { // DIV increment
			e.timer.ProcessDivIncrement()
		} else {
			slog.Warn("Unknown event type", "type", event.EventType)
		}
	}
}

// processCPUInstruction executes one CPU instruction and schedules follow-up events
func (e *EventDrivenEmulator) processCPUInstruction() {
	// Execute CPU instruction
	pc := e.cpu.GetPC()
	cycles := e.cpu.Tick()
	e.instructionCount++

	currentCycle := e.scheduler.GetCurrentCycle()
	nextCycle := currentCycle + uint64(cycles)

	// Log timer-related instructions for debugging
	if (e.instructionCount <= 1000 && e.instructionCount%50 == 0) || (e.instructionCount <= 10000 && e.instructionCount%500 == 0) {
		tima := e.mem.Read(0xFF05)
		tac := e.mem.Read(0xFF07)
		div := e.mem.Read(0xFF04)
		slog.Debug("CPU instruction executed",
			"pc", fmt.Sprintf("0x%04X", pc),
			"cycles", cycles,
			"total_cycles", nextCycle,
			"instruction_count", e.instructionCount,
			"tima", fmt.Sprintf("0x%02X", tima),
			"tac", fmt.Sprintf("0x%02X", tac),
			"div", fmt.Sprintf("0x%02X", div))
	}

	// Schedule timer events for the cycles consumed by this instruction
	e.timer.ScheduleEvents(e.scheduler, cycles)

	// Update GPU (keeping current batched approach for now)
	e.gpu.Tick(cycles)

	// Check if we completed a frame (70224 cycles)
	frameThreshold := (e.frameCount + 1) * 70224
	if nextCycle >= uint64(frameThreshold) {
		e.frameCount++
		e.processFrameComplete()
	}

	// Schedule next CPU instruction
	e.scheduler.Schedule(CPUInstruction, nextCycle, nil)
}

// processVBlankStart handles VBlank start events
func (e *EventDrivenEmulator) processVBlankStart() {
	// VBlank processing - placeholder for now
	slog.Debug("VBlank start", "cycle", e.scheduler.GetCurrentCycle())
}

// processFrameComplete handles frame completion
func (e *EventDrivenEmulator) processFrameComplete() {
	// Log frame completion
	if e.frameCount%60 == 0 {
		slog.Debug("Frame completed",
			"frame", e.frameCount,
			"cycle", e.scheduler.GetCurrentCycle(),
			"instructions", e.instructionCount)
	}
}

// Stop gracefully stops the emulator
func (e *EventDrivenEmulator) Stop() {
	e.running = false
}

// Getter methods for compatibility with existing interfaces
func (e *EventDrivenEmulator) GetCurrentFrame() *video.FrameBuffer {
	return e.gpu.GetFrameBuffer()
}

func (e *EventDrivenEmulator) GetCPU() *cpu.CPU {
	return e.cpu
}

func (e *EventDrivenEmulator) GetMMU() *memory.MMU {
	return e.mem
}

func (e *EventDrivenEmulator) GetInstructionCount() uint64 {
	return e.instructionCount
}

func (e *EventDrivenEmulator) GetFrameCount() uint64 {
	return e.frameCount
}

func (e *EventDrivenEmulator) GetEventCount() uint64 {
	return e.eventsProcessed
}

// HandleKeyPress forwards input to memory management unit
func (e *EventDrivenEmulator) HandleKeyPress(key memory.JoypadKey) {
	e.mem.HandleKeyPress(key)
}

// HandleKeyRelease forwards input to memory management unit
func (e *EventDrivenEmulator) HandleKeyRelease(key memory.JoypadKey) {
	e.mem.HandleKeyRelease(key)
}
