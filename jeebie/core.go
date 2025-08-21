package jeebie

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/valerio/go-jeebie/jeebie/addr"
	"github.com/valerio/go-jeebie/jeebie/cpu"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/timing"
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

// TestCompletionDetector tracks execution patterns to detect when tests finish
type TestCompletionDetector struct {
	MaxCycles       uint64 // Safety timeout in total cycles
	MaxFrames       uint64 // Alternative safety timeout in frames
	PatternCycles   uint64 // Cycles to confirm loop pattern
	LastPC          uint16 // Track PC for loop detection
	LoopCount       int    // Count consecutive loops at same PC
	MinLoopCount    int    // Minimum loops to confirm completion
	LastInstruction uint8  // Last executed instruction
	LastOperand     uint8  // Last operand for JR -2 detection
	Enabled         bool   // Whether detection is active
}

// NewTestCompletionDetector creates a detector with reasonable defaults
func NewTestCompletionDetector() *TestCompletionDetector {
	return &TestCompletionDetector{
		MaxCycles:     70224 * 1000, // ~1000 frames worth of cycles
		MaxFrames:     1000,         // 1000 frames max
		PatternCycles: 70224,        // At least one frame of looping
		MinLoopCount:  100,          // 100 consecutive loops
		Enabled:       true,
	}
}

// DMG represents the Game Boy emulator (Dot Matrix Game)
type DMG struct {
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

	// Test completion detection
	completionDetector *TestCompletionDetector

	// Frame timing
	limiter timing.Limiter
}

func (e *DMG) init(mem *memory.MMU) {
	e.cpu = cpu.New(mem)
	e.gpu = video.NewGpu(mem)
	e.mem = mem

	e.systemCounter = 0xABCC
	e.lastTimerBit = false
	e.timaOverflow = 0
	e.timaDelayInt = false
	e.completionDetector = NewTestCompletionDetector()
	e.limiter = timing.NewNoOpLimiter()
	mem.Write(addr.DIV, byte(e.systemCounter>>8))
}

// NewWithFile creates a new emulator instance and loads the file specified into it.
func NewWithFile(path string) (*DMG, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	e := &DMG{}
	e.init(memory.NewWithCartridge(memory.NewCartridgeWithData(data)))

	return e, nil
}

func (e *DMG) RunUntilFrame() error {
	e.debuggerMutex.RLock()
	state := e.debuggerState
	e.debuggerMutex.RUnlock()

	// Handle paused state - don't execute anything
	if state == DebuggerPaused {
		return nil
	}

	// Handle step instruction - execute one instruction then pause
	if state == DebuggerStep {
		e.debuggerMutex.Lock()
		if e.stepRequested {
			e.stepRequested = false
			e.debuggerMutex.Unlock()

			// Execute one CPU instruction
			cycles := e.cpu.Tick()
			e.updateTimers(cycles)
			e.gpu.Tick(cycles)
			e.instructionCount++

			// Pause after execution
			e.SetDebuggerState(DebuggerPaused)
		} else {
			e.debuggerMutex.Unlock()
		}
		return nil
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
			e.SetDebuggerState(DebuggerPaused)
		}
		return nil
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
			e.limiter.WaitForNextFrame()
			return nil
		}
	}
}

func (e *DMG) GetCurrentFrame() *video.FrameBuffer {
	return e.gpu.GetFrameBuffer()
}

func (e *DMG) HandleKeyPress(key memory.JoypadKey) {
	e.mem.HandleKeyPress(key)
}

func (e *DMG) HandleKeyRelease(key memory.JoypadKey) {
	e.mem.HandleKeyRelease(key)
}

func (e *DMG) HandleAction(act action.Action, pressed bool) {
	switch act {
	case action.EmulatorPauseToggle:
		if pressed {
			if e.debuggerState == DebuggerPaused {
				e.debuggerState = DebuggerRunning
			} else {
				e.debuggerState = DebuggerPaused
			}
		}
		return
	case action.EmulatorStepFrame:
		if pressed && e.debuggerState == DebuggerPaused {
			e.debuggerState = DebuggerStepFrame
			e.frameRequested = true
		}
		return
	case action.EmulatorStepInstruction:
		if pressed && e.debuggerState == DebuggerPaused {
			e.debuggerState = DebuggerStep
			e.stepRequested = true
		}
		return
	}

	var key memory.JoypadKey
	switch act {
	case action.GBButtonA:
		key = memory.JoypadA
	case action.GBButtonB:
		key = memory.JoypadB
	case action.GBButtonStart:
		key = memory.JoypadStart
	case action.GBButtonSelect:
		key = memory.JoypadSelect
	case action.GBDPadUp:
		key = memory.JoypadUp
	case action.GBDPadDown:
		key = memory.JoypadDown
	case action.GBDPadLeft:
		key = memory.JoypadLeft
	case action.GBDPadRight:
		key = memory.JoypadRight
	default:
		return
	}

	if pressed {
		e.mem.HandleKeyPress(key)
	} else {
		e.mem.HandleKeyRelease(key)
	}
}

// Debugger control methods (internal use)
func (e *DMG) SetDebuggerState(state DebuggerState) {
	e.debuggerMutex.Lock()
	defer e.debuggerMutex.Unlock()
	e.debuggerState = state
}

func (e *DMG) GetInstructionCount() uint64 {
	return e.instructionCount
}

func (e *DMG) GetFrameCount() uint64 {
	return e.frameCount
}

func (e *DMG) ExtractDebugData() *debug.CompleteDebugData {
	if e.mem == nil || e.cpu == nil {
		return nil
	}

	spriteHeight := 8
	if e.mem.ReadBit(2, addr.LCDC) {
		spriteHeight = 16
	}

	// Get current scanline
	currentLine := int(e.mem.Read(addr.LY))
	oamData := debug.ExtractOAMData(e.mem, currentLine, spriteHeight)
	vramData := debug.ExtractVRAMData(e.mem)

	cpuState := &debug.CPUState{
		A:      e.cpu.GetA(),
		F:      e.cpu.GetF(),
		B:      e.cpu.GetB(),
		C:      e.cpu.GetC(),
		D:      e.cpu.GetD(),
		E:      e.cpu.GetE(),
		H:      e.cpu.GetH(),
		L:      e.cpu.GetL(),
		SP:     e.cpu.GetSP(),
		PC:     e.cpu.GetPC(),
		IME:    e.cpu.GetIME(),
		Cycles: e.instructionCount,
	}

	const snapshotSize = 200 // Enough for disassembly before and after PC
	const beforePC = 50      // Bytes before PC to capture
	pc := e.cpu.GetPC()

	// Calculate start address, handling underflow
	startAddr := pc
	if pc >= beforePC {
		startAddr = pc - beforePC
	} else {
		startAddr = 0
	}

	memSnapshot := &debug.MemorySnapshot{
		StartAddr: startAddr,
		Bytes:     make([]uint8, snapshotSize),
	}
	for i := 0; i < snapshotSize; i++ {
		addr := startAddr + uint16(i)
		if addr < 0x8000 || (addr >= 0xA000 && addr < 0xE000) || addr >= 0xFE00 {
			// Safe to read from these areas
			memSnapshot.Bytes[i] = e.mem.Read(addr)
		} else {
			// VRAM/OAM might be inaccessible, use NOP
			memSnapshot.Bytes[i] = 0x00
		}
	}

	var debuggerState debug.DebuggerState
	switch e.debuggerState {
	case DebuggerPaused:
		debuggerState = debug.DebuggerPaused
	case DebuggerStep:
		debuggerState = debug.DebuggerStepInstruction
	case DebuggerStepFrame:
		debuggerState = debug.DebuggerStepFrame
	default:
		debuggerState = debug.DebuggerRunning
	}

	return &debug.CompleteDebugData{
		OAM:             oamData,
		VRAM:            vramData,
		CPU:             cpuState,
		Memory:          memSnapshot,
		DebuggerState:   debuggerState,
		InterruptEnable: e.mem.Read(addr.IE),
		InterruptFlags:  e.mem.Read(addr.IF),
	}
}

func (e *DMG) updateTimers(cycles int) {
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

// UpdateCompletionDetection updates the completion detector with current execution state
func (e *DMG) UpdateCompletionDetection() {
	if !e.completionDetector.Enabled {
		return
	}

	currentPC := e.cpu.GetPC()

	// Check for JR -2 pattern (0x18, 0xFE)
	if currentPC == e.completionDetector.LastPC {
		// Same PC, increment loop count
		e.completionDetector.LoopCount++
	} else {
		// Different PC, reset loop count
		e.completionDetector.LoopCount = 0
		e.completionDetector.LastPC = currentPC
	}

	// Additional check for JR -2 instruction pattern
	instruction := e.mem.Read(currentPC)
	if instruction == 0x18 { // JR instruction
		operand := e.mem.Read(currentPC + 1)
		if operand == 0xFE { // -2 in two's complement
			e.completionDetector.LastInstruction = instruction
			e.completionDetector.LastOperand = operand
		}
	}
}

// IsTestComplete checks if the test appears to have completed
func (e *DMG) IsTestComplete() bool {
	if !e.completionDetector.Enabled {
		return false
	}

	detector := e.completionDetector

	// Safety timeout based on total cycles
	totalCycles := e.cpu.GetCycles()
	if totalCycles >= detector.MaxCycles {
		slog.Debug("Test completion: cycle timeout reached", "cycles", totalCycles, "max", detector.MaxCycles)
		return true
	}

	// Safety timeout based on frames
	if e.frameCount >= detector.MaxFrames {
		slog.Debug("Test completion: frame timeout reached", "frames", e.frameCount, "max", detector.MaxFrames)
		return true
	}

	// Check for JR -2 loop pattern
	if detector.LoopCount >= detector.MinLoopCount {
		currentPC := e.cpu.GetPC()
		instruction := e.mem.Read(currentPC)
		operand := e.mem.Read(currentPC + 1)

		if instruction == 0x18 && operand == 0xFE {
			slog.Debug("Test completion: JR -2 loop detected", "pc", fmt.Sprintf("0x%04X", currentPC), "loops", detector.LoopCount)
			return true
		}
	}

	return false
}

// RunUntilComplete runs the emulator until test completion is detected
func (e *DMG) RunUntilComplete() {
	for !e.IsTestComplete() {
		e.UpdateCompletionDetection()
		e.RunUntilFrame()
	}

	slog.Info("Test completed", "frames", e.frameCount, "instructions", e.instructionCount)
}

// EnableCompletionDetection enables or disables test completion detection
func (e *DMG) EnableCompletionDetection(enabled bool) {
	e.completionDetector.Enabled = enabled
}

// ConfigureCompletionDetection allows customizing completion detection parameters
func (e *DMG) ConfigureCompletionDetection(maxFrames uint64, minLoopCount int) {
	e.completionDetector.MaxFrames = maxFrames
	e.completionDetector.MaxCycles = maxFrames * 70224
	e.completionDetector.MinLoopCount = minLoopCount
}

// SetFrameLimiter sets the frame rate limiter for the emulator.
// Pass nil to disable frame limiting (useful for headless mode).
func (e *DMG) SetFrameLimiter(limiter timing.Limiter) {
	if limiter == nil {
		e.limiter = timing.NewNoOpLimiter()
	} else {
		e.limiter = limiter
	}
}

// ResetFrameTiming resets the frame limiter timing.
// Useful after pauses or when resuming emulation.
func (e *DMG) ResetFrameTiming() {
	e.limiter.Reset()
}
