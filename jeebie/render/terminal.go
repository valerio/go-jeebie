package render

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/disasm"
	"github.com/valerio/go-jeebie/jeebie/memory"
)

const (
	width     = 160
	height    = 144
	scaleX    = 1  // Reduce from 2 to 1 for more compact display
	scaleY    = 1
	frameTime = time.Second / 60
	
	// Layout constants
	gameAreaWidth  = width * scaleX  // 160
	gameAreaHeight = height * scaleY // 144
	registerHeight = 7               // Lines for CPU registers + status
	disasmHeight   = 9               // Lines for disassembly (4 before + 1 current + 4 after)
	minTermWidth   = 100             // Very compact mode for small terminals  
	minTermHeight  = 35              // Increase to accommodate disasm window
)

var shadeChars = []rune{'█', '▓', '▒', '░'}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type TerminalRenderer struct {
	screen   tcell.Screen
	emulator *jeebie.Emulator
	running  bool
	logBuffer *LogBuffer
}

func NewTerminalRenderer(emu *jeebie.Emulator) (*TerminalRenderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terminal: %v", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize terminal: %v", err)
	}

	// Create log buffer and set up logging
	logBuffer := NewLogBuffer(100)
	
	// Set up the log handler to capture logs
	handler := NewLogBufferHandler(logBuffer, slog.LevelDebug)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	
	// Add some initial test logs
	slog.Info("Terminal renderer initialized")
	slog.Debug("Split-screen layout ready")

	return &TerminalRenderer{
		screen:    screen,
		emulator:  emu,
		running:   true,
		logBuffer: logBuffer,
	}, nil
}

func (t *TerminalRenderer) Run() error {
	defer func() {
		slog.Info("Finishing terminal")
		t.screen.Fini()
	}()

	t.screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	t.screen.Clear()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	go t.handleInput()

	ticker := time.NewTicker(frameTime)
	defer ticker.Stop()

	for t.running {
		select {
		case <-ticker.C:
			t.emulator.RunUntilFrame()
			t.render()
			t.screen.Show()
			
		case <-signals:
			t.running = false
			slog.Info("Received signal to stop")
			return nil
		}
	}

	return nil
}

func (t *TerminalRenderer) handleInput() {
	for t.running {
		ev := t.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				t.running = false
				return
			case tcell.KeyEnter:
				t.emulator.HandleKeyPress(memory.JoypadStart)
			case tcell.KeyRight:
				t.emulator.HandleKeyPress(memory.JoypadRight)
			case tcell.KeyLeft:
				t.emulator.HandleKeyPress(memory.JoypadLeft)
			case tcell.KeyUp:
				t.emulator.HandleKeyPress(memory.JoypadUp)
			case tcell.KeyDown:
				t.emulator.HandleKeyPress(memory.JoypadDown)
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'a':
					t.emulator.HandleKeyPress(memory.JoypadA)
				case 's':
					t.emulator.HandleKeyPress(memory.JoypadB)
				case 'q':
					t.emulator.HandleKeyPress(memory.JoypadSelect)
				
				// Debugger controls
				case ' ': // Spacebar - pause/resume toggle
					debugState := t.emulator.GetDebuggerState()
					if debugState == 1 { // DebuggerPaused
						t.emulator.DebuggerResume()
					} else {
						t.emulator.DebuggerPause()
					}
				case 'n': // Next instruction (step)
					t.emulator.DebuggerStepInstruction()
				case 'f': // Next frame (step frame)
					t.emulator.DebuggerStepFrame()
				case 'r': // Resume from any state
					t.emulator.DebuggerResume()
				case 'p': // Pause
					t.emulator.DebuggerPause()
				}
			}
		case *tcell.EventResize:
			t.screen.Sync()
		}
	}
}

func (t *TerminalRenderer) render() {
	termWidth, termHeight := t.screen.Size()
	
	// Check minimum terminal size
	if termWidth < minTermWidth || termHeight < minTermHeight {
		t.screen.Clear()
		style := tcell.StyleDefault.Foreground(tcell.ColorRed)
		msg := fmt.Sprintf("Terminal too small! Need at least %dx%d", minTermWidth, minTermHeight)
		for i, ch := range msg {
			t.screen.SetContent(i, termHeight/2, ch, nil, style)
		}
		return
	}

	t.screen.Clear()

	// Draw borders and sections
	t.drawBorders(termWidth, termHeight)
	
	// Draw Game Boy screen (left side)
	t.drawGameBoy()
	
	// Draw CPU registers (top-right)
	t.drawRegisters(termWidth, termHeight)
	
	// Draw disassembly (middle-right)
	t.drawDisassembly(termWidth, termHeight)
	
	// Draw logs (bottom-right)  
	t.drawLogs(termWidth, termHeight)
}

func (t *TerminalRenderer) drawBorders(termWidth, termHeight int) {
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	
	// Adaptive border position - use available space
	borderX := min(gameAreaWidth + 1, termWidth/2)
	if borderX >= termWidth-10 {
		borderX = termWidth - 10 // Leave at least 10 chars for right panel
	}
	
	// Vertical border between game area and right panel
	for y := 0; y < termHeight; y++ {
		if borderX < termWidth {
			t.screen.SetContent(borderX, y, '│', nil, borderStyle)
		}
	}
	
	// Horizontal border between registers and disassembly
	registerEndY := registerHeight + 1
	if registerEndY < termHeight {
		for x := borderX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, registerEndY, '─', nil, borderStyle)
		}
		// Corner piece
		t.screen.SetContent(borderX, registerEndY, '├', nil, borderStyle)
	}
	
	// Horizontal border between disassembly and logs
	disasmEndY := registerEndY + disasmHeight + 1
	if disasmEndY < termHeight {
		for x := borderX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, disasmEndY, '─', nil, borderStyle)
		}
		// Corner piece
		t.screen.SetContent(borderX, disasmEndY, '├', nil, borderStyle)
	}
	
	// Title headers
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
	
	// Game Boy title
	title := " Game Boy "
	for i, ch := range title {
		t.screen.SetContent(1+i, 0, ch, nil, titleStyle)
	}
	
	// CPU Registers title  
	title = " CPU Registers "
	for i, ch := range title {
		t.screen.SetContent(borderX+2+i, 0, ch, nil, titleStyle)
	}
	
	// Disassembly title
	if registerEndY+1 < termHeight {
		title = " Disassembly "
		for i, ch := range title {
			t.screen.SetContent(borderX+2+i, registerEndY+1, ch, nil, titleStyle)
		}
	}
	
	// Logs title
	disasmEndY = registerEndY + disasmHeight + 1
	if disasmEndY+1 < termHeight {
		title = " Logs "
		for i, ch := range title {
			t.screen.SetContent(borderX+2+i, disasmEndY+1, ch, nil, titleStyle)
		}
	}
	
	// Debug help text at bottom
	if termHeight > 10 {
		helpStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		helpText := "Debug: SPACE=pause/resume N=step P=pause R=resume F=step-frame"
		startX := 1
		maxWidth := min(len(helpText), termWidth-2)
		for i, ch := range helpText[:maxWidth] {
			t.screen.SetContent(startX+i, termHeight-1, ch, nil, helpStyle)
		}
	}
}

func (t *TerminalRenderer) drawGameBoy() {
	fb := t.emulator.GetCurrentFrame()
	frame := fb.ToSlice()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := frame[y*width+x]
			
			// Convert Game Boy color to shade
			shade := 0
			switch pixel {
			case 0x000000FF: // BlackColor
				shade = 0
			case 0x4C4C4CFF: // DarkGreyColor
				shade = 1
			case 0x989898FF: // LightGreyColor
				shade = 2
			case 0xFFFFFFFF: // WhiteColor
				shade = 3
			default:
				shade = 0
			}
			
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			char := shadeChars[shade]
			screenX := x * scaleX
			screenY := y * scaleY + 1 // Offset for title
			
			// Only draw within game area bounds
			for sx := 0; sx < scaleX; sx++ {
				if screenX+sx < gameAreaWidth {
					t.screen.SetContent(screenX+sx, screenY, char, nil, style)
				}
			}
		}
	}
}

func (t *TerminalRenderer) drawRegisters(termWidth, termHeight int) {
	cpu := t.emulator.GetCPU()
	startX := gameAreaWidth + 3
	startY := 1
	
	regStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	
	// Get debugger state and format status
	debugState := t.emulator.GetDebuggerState()
	debugStatus := ""
	debugStyle := regStyle
	switch debugState {
	case 0: // DebuggerRunning
		debugStatus = "RUNNING"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	case 1: // DebuggerPaused
		debugStatus = "PAUSED"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow)
	case 2: // DebuggerStep
		debugStatus = "STEP"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorBlue)
	case 3: // DebuggerStepFrame
		debugStatus = "FRAME"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorRed)
	}
	
	// Format and display CPU registers
	registers := []string{
		fmt.Sprintf("Status: %s", debugStatus),
		fmt.Sprintf("A: 0x%02X  F: 0x%02X [%s]", cpu.GetA(), cpu.GetF(), cpu.GetFlagString()),
		fmt.Sprintf("B: 0x%02X  C: 0x%02X", cpu.GetB(), cpu.GetC()),
		fmt.Sprintf("D: 0x%02X  E: 0x%02X", cpu.GetD(), cpu.GetE()),
		fmt.Sprintf("H: 0x%02X  L: 0x%02X", cpu.GetH(), cpu.GetL()),
		fmt.Sprintf("SP: 0x%04X  PC: 0x%04X", cpu.GetSP(), cpu.GetPC()),
		fmt.Sprintf("Frame: %d  Instr: %d", t.emulator.GetFrameCount(), t.emulator.GetInstructionCount()),
	}
	
	for i, reg := range registers {
		if startY+i >= registerHeight+1 || startY+i >= termHeight {
			break
		}
		
		// Use debug style for status line, regular style for others
		style := regStyle
		if i == 0 { // Status line
			style = debugStyle
		}
		
		x := startX
		for _, ch := range reg {
			if x >= termWidth {
				break
			}
			t.screen.SetContent(x, startY+i, ch, nil, style)
			x++
		}
	}
}

func (t *TerminalRenderer) drawDisassembly(termWidth, termHeight int) {
	startX := gameAreaWidth + 3
	startY := registerHeight + 3
	
	// Get current PC and MMU from emulator
	cpu := t.emulator.GetCPU()
	mmu := t.emulator.GetMMU()
	currentPC := cpu.GetPC()
	
	// Get disassembly around current PC (4 before, 4 after)
	lines := disasm.DisassembleAround(currentPC, 4, 4, mmu)
	
	disasmStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	currentPCStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue)
	
	// Display up to disasmHeight lines
	maxLines := min(len(lines), disasmHeight)
	for i := 0; i < maxLines; i++ {
		if startY+i >= termHeight {
			break
		}
		
		line := lines[i]
		isCurrentPC := line.Address == currentPC
		
		// Format the disassembly line
		text := disasm.FormatDisassemblyLine(line, isCurrentPC)
		
		// Choose style based on whether this is the current PC
		style := disasmStyle
		if isCurrentPC {
			style = currentPCStyle
		}
		
		x := startX
		maxWidth := termWidth - startX - 1
		if len(text) > maxWidth {
			text = text[:maxWidth-3] + "..."
		}
		
		for _, ch := range text {
			if x >= termWidth {
				break
			}
			t.screen.SetContent(x, startY+i, ch, nil, style)
			x++
		}
	}
}

func (t *TerminalRenderer) drawLogs(termWidth, termHeight int) {
	startX := gameAreaWidth + 3
	startY := registerHeight + 3 + disasmHeight + 1 // Account for disassembly section
	availableHeight := termHeight - startY
	
	if availableHeight <= 0 {
		return
	}
	
	// Get recent logs
	logs := t.logBuffer.GetRecent(availableHeight)
	
	logStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue)
	warnStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
	errStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
	
	for i, logEntry := range logs {
		if i >= availableHeight {
			break
		}
		
		// Choose style based on log level
		style := logStyle
		switch logEntry.Level {
		case slog.LevelWarn:
			style = warnStyle
		case slog.LevelError:
			style = errStyle
		}
		
		logText := FormatLogEntry(logEntry)
		y := startY + i
		x := startX
		
		// Truncate log line if too long
		maxWidth := termWidth - startX - 1
		if len(logText) > maxWidth {
			logText = logText[:maxWidth-3] + "..."
		}
		
		for _, ch := range logText {
			if x >= termWidth {
				break
			}
			t.screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}
