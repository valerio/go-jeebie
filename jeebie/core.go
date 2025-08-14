package jeebie

import (
	"fmt"
	"io/ioutil"
	"log/slog"
	"sync"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// DebuggerState represents the current debugger mode
type DebuggerState int

const (
	DebuggerRunning   DebuggerState = iota // Normal execution
	DebuggerPaused                         // Paused, waiting for commands
	DebuggerStep                           // Execute one instruction then pause
	DebuggerStepFrame                      // Execute one frame then pause
)

// Emulator represents the root struct and entry point for running the emulation
type Emulator struct {
	cpu *cpu.CPU
	gpu *video.GPU
	mem *memory.MMU

	// Timer state
	systemCounter uint16 // Internal 16-bit counter, DIV is upper 8 bits
	lastTimerBit  bool   // Previous state of timer bit for edge detection
	timaOverflow  int    // Cycles remaining in TIMA overflow state
	timaDelayInt  bool   // Delayed interrupt flag setting (1 M-cycle after TMA load)

	// Debugger state
	debuggerState    DebuggerState
	debuggerMutex    sync.RWMutex
	stepRequested    bool
	frameRequested   bool
	instructionCount uint64
	frameCount       uint64
}

func (e *Emulator) init(mem *memory.MMU) {
	e.cpu = cpu.New(mem)
	e.gpu = video.NewGpu(mem)
	e.mem = mem

	e.systemCounter = 0xABCC
	e.lastTimerBit = false
	e.timaOverflow = 0
	e.timaDelayInt = false
	mem.Write(addr.DIV, byte(e.systemCounter>>8))
}

// New creates a new emulator instance
func New() *Emulator {
	e := &Emulator{}
	e.init(memory.NewWithCartridge(memory.NewCartridge()))

	return e
}

// NewWithFile creates a new emulator instance and loads the file specified into it.
func NewWithFile(path string) (*Emulator, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	slog.Debug("Loaded ROM data", "size", len(data))

	e := &Emulator{}
	e.init(memory.NewWithCartridge(memory.NewCartridgeWithData(data)))

	return e, nil
}

func (e *Emulator) RunUntilFrame() {
	e.debuggerMutex.RLock()
	state := e.debuggerState
	e.debuggerMutex.RUnlock()

	// Handle paused state - don't execute anything
	if state == DebuggerPaused {
		return
	}

	// Handle step instruction - execute one instruction then pause
	if state == DebuggerStep {
		e.debuggerMutex.Lock()
		if e.stepRequested {
			e.stepRequested = false
			e.debuggerMutex.Unlock()

			// Execute one CPU instruction
			oldPC := e.cpu.GetPC()
			cycles := e.cpu.Tick()
			e.updateTimers(cycles)
			e.gpu.Tick(cycles)
			e.instructionCount++

			// Log the executed instruction
			slog.Debug("Step executed", "pc", fmt.Sprintf("0x%04X", oldPC), "new_pc", fmt.Sprintf("0x%04X", e.cpu.GetPC()))

			// Pause after execution
			e.SetDebuggerState(DebuggerPaused)
		} else {
			e.debuggerMutex.Unlock()
		}
		return
	}

	// Handle step frame - execute one frame then pause
	if state == DebuggerStepFrame {
		e.debuggerMutex.Lock()
		frameRequested := e.frameRequested
		if frameRequested {
			e.frameRequested = false
		}
		e.debuggerMutex.Unlock()

		if frameRequested {
			// Execute one full frame
			total := 0
			for {
				cycles := e.cpu.Tick()
				e.updateTimers(cycles)
				e.gpu.Tick(cycles)
				e.instructionCount++
				total += cycles

				if total >= 70224 {
					break
				}
			}
			e.frameCount++
			slog.Debug("Frame step completed", "frame", e.frameCount, "instructions", e.instructionCount)
			e.SetDebuggerState(DebuggerPaused)
		}
		return
	}

	// Normal execution (DebuggerRunning)
	total := 0
	for {
		cycles := e.cpu.Tick()
		e.updateTimers(cycles)
		e.gpu.Tick(cycles)
		e.instructionCount++

		total += cycles

		if total >= 70224 {
			e.frameCount++
			// Log every 60 frames (once per second at 60 FPS) only when running
			if e.frameCount%60 == 0 {
				slog.Debug("Frame completed", "frame", e.frameCount, "pc", fmt.Sprintf("0x%04X", e.cpu.GetPC()))
			}
			return
		}
	}
}

func (e *Emulator) GetCurrentFrame() *video.FrameBuffer {
	return e.gpu.GetFrameBuffer()
}

func (e *Emulator) HandleKeyPress(key memory.JoypadKey) {
	e.mem.HandleKeyPress(key)
}

func (e *Emulator) HandleKeyRelease(key memory.JoypadKey) {
	e.mem.HandleKeyRelease(key)
}

func (e *Emulator) GetCPU() *cpu.CPU {
	return e.cpu
}

// Debugger control methods
func (e *Emulator) SetDebuggerState(state DebuggerState) {
	e.debuggerMutex.Lock()
	defer e.debuggerMutex.Unlock()
	e.debuggerState = state
	slog.Debug("Debugger state changed", "state", state)
}

func (e *Emulator) GetDebuggerState() DebuggerState {
	e.debuggerMutex.RLock()
	defer e.debuggerMutex.RUnlock()
	return e.debuggerState
}

func (e *Emulator) DebuggerPause() {
	e.SetDebuggerState(DebuggerPaused)
	slog.Info("Emulator paused")
}

func (e *Emulator) DebuggerResume() {
	e.SetDebuggerState(DebuggerRunning)
	slog.Info("Emulator resumed")
}

func (e *Emulator) DebuggerStepInstruction() {
	e.debuggerMutex.Lock()
	defer e.debuggerMutex.Unlock()
	e.stepRequested = true
	e.debuggerState = DebuggerStep
	slog.Info("Step instruction requested")
}

func (e *Emulator) DebuggerStepFrame() {
	e.debuggerMutex.Lock()
	defer e.debuggerMutex.Unlock()
	e.frameRequested = true
	e.debuggerState = DebuggerStepFrame
	slog.Info("Step frame requested")
}

func (e *Emulator) GetInstructionCount() uint64 {
	return e.instructionCount
}

func (e *Emulator) GetFrameCount() uint64 {
	return e.frameCount
}

func (e *Emulator) GetMMU() *memory.MMU {
	return e.mem
}

func (e *Emulator) updateTimers(cycles int) {
	if e.timaDelayInt {
		e.mem.RequestInterrupt(addr.TimerInterrupt)
		e.timaDelayInt = false
	}

	if e.timaOverflow > 0 {
		e.timaOverflow -= cycles
		if e.timaOverflow <= 0 {
			tma := e.mem.Read(addr.TMA)
			e.mem.Write(addr.TIMA, tma)
			e.timaDelayInt = true
			e.timaOverflow = 0
		}
	}

	for i := 0; i < cycles; i++ {
		e.systemCounter++
		e.mem.Write(addr.DIV, byte(e.systemCounter>>8))

		if e.timaOverflow > 0 {
			continue
		}

		tac := e.mem.Read(addr.TAC)
		timerEnabled := (tac & 0x04) != 0

		if timerEnabled {
			var bitPosition uint
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

			currentTimerBit := (e.systemCounter & (1 << bitPosition)) != 0

			if e.lastTimerBit && !currentTimerBit {
				currentTima := e.mem.Read(addr.TIMA)
				if currentTima == 0xFF {
					e.mem.Write(addr.TIMA, 0x00)
					e.timaOverflow = 4
				} else {
					e.mem.Write(addr.TIMA, currentTima+1)
				}
			}

			e.lastTimerBit = currentTimerBit
		} else {
			e.lastTimerBit = false
		}
	}
}
