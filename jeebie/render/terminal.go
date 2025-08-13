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
	scaleX    = 1 // Reduce from 2 to 1 for more compact display
	scaleY    = 1
	frameTime = time.Second / 60

	gameAreaWidth  = width * scaleX
	gameAreaHeight = height * scaleY
	registerHeight = 12
	disasmHeight   = 9
	minTermWidth   = 80
	minTermHeight  = 24
)

var shadeChars = []rune{'█', '▓', '▒', '░'}

type TerminalRenderer struct {
	screen    tcell.Screen
	emulator  *jeebie.Emulator
	running   bool
	logBuffer *LogBuffer
	logLevel  slog.Level
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
		logLevel:  slog.LevelInfo,
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
						slog.Info("Debugger: Resuming execution")
						t.emulator.DebuggerResume()
					} else {
						slog.Info("Debugger: Pausing execution")
						t.emulator.DebuggerPause()
					}
				case 'n': // Next instruction (step)
					slog.Info("Debugger: Step instruction")
					t.emulator.DebuggerStepInstruction()
				case 'f': // Next frame (step frame)
					slog.Info("Debugger: Step frame")
					t.emulator.DebuggerStepFrame()
				case 'r': // Resume from any state
					slog.Info("Debugger: Resume")
					t.emulator.DebuggerResume()
				case 'p': // Pause
					slog.Info("Debugger: Pause")
					t.emulator.DebuggerPause()
				case '-', '_': // Decrease log verbosity (show fewer logs)
					oldLevel := t.logLevel
					switch t.logLevel {
					case slog.LevelDebug:
						t.logLevel = slog.LevelInfo
					case slog.LevelInfo:
						t.logLevel = slog.LevelWarn
					case slog.LevelWarn:
						t.logLevel = slog.LevelError
					}
					if oldLevel != t.logLevel {
						slog.Info("Log filter changed", "from", oldLevel, "to", t.logLevel)
					}
				case '+', '=': // Increase log verbosity (show more logs)
					oldLevel := t.logLevel
					switch t.logLevel {
					case slog.LevelError:
						t.logLevel = slog.LevelWarn
					case slog.LevelWarn:
						t.logLevel = slog.LevelInfo
					case slog.LevelInfo:
						t.logLevel = slog.LevelDebug
					}
					if oldLevel != t.logLevel {
						slog.Info("Log filter changed", "from", oldLevel, "to", t.logLevel)
					}
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

	borderX := termWidth / 2
	if termWidth > 100 {
		borderX = min(gameAreaWidth+1, termWidth*2/3)
	}

	// Vertical border between game area and right panel
	for y := 0; y < termHeight; y++ {
		if borderX < termWidth {
			t.screen.SetContent(borderX, y, '│', nil, borderStyle)
		}
	}

	registerEndY := min(registerHeight+1, termHeight/3)
	disasmEndY := min(registerEndY+disasmHeight+1, termHeight*2/3)

	if registerEndY < termHeight {
		for x := borderX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, registerEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(borderX, registerEndY, '├', nil, borderStyle)
	}

	if disasmEndY < termHeight-1 {
		for x := borderX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, disasmEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(borderX, disasmEndY, '├', nil, borderStyle)
	}

	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	title := " Game Boy "
	for i, ch := range title {
		t.screen.SetContent(1+i, 0, ch, nil, titleStyle)
	}

	title = " CPU Registers "
	for i, ch := range title {
		if borderX+2+i < termWidth {
			t.screen.SetContent(borderX+2+i, 0, ch, nil, titleStyle)
		}
	}

	if registerEndY+1 < termHeight {
		title = " Disassembly "
		for i, ch := range title {
			if borderX+2+i < termWidth {
				t.screen.SetContent(borderX+2+i, registerEndY+1, ch, nil, titleStyle)
			}
		}
	}

	if disasmEndY+1 < termHeight {
		levelStr := "INFO"
		switch t.logLevel {
		case slog.LevelDebug:
			levelStr = "DEBUG"
		case slog.LevelWarn:
			levelStr = "WARN"
		case slog.LevelError:
			levelStr = "ERROR"
		}
		title = fmt.Sprintf(" Logs [%s] (-/+ filter) ", levelStr)
		for i, ch := range title {
			if borderX+2+i < termWidth {
				t.screen.SetContent(borderX+2+i, disasmEndY+1, ch, nil, titleStyle)
			}
		}
	}

	if termHeight > 10 {
		helpStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		helpText := "Debug: SPACE=pause/resume N=step F=frame | Logs: +/- filter"
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

			shade := 0
			switch pixel {
			case 0x000000FF:
				shade = 0
			case 0x4C4C4CFF:
				shade = 1
			case 0x989898FF:
				shade = 2
			case 0xFFFFFFFF:
				shade = 3
			default:
				shade = 0
			}

			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			char := shadeChars[shade]
			screenX := x * scaleX
			screenY := y*scaleY + 1

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
	mmu := t.emulator.GetMMU()
	borderX := termWidth / 2
	if termWidth > 100 {
		borderX = min(gameAreaWidth+1, termWidth*2/3)
	}
	startX := borderX + 2
	startY := 1

	regStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)

	debugState := t.emulator.GetDebuggerState()
	debugStatus := ""
	debugStyle := regStyle
	switch debugState {
	case 0:
		debugStatus = "RUNNING"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	case 1:
		debugStatus = "PAUSED"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow)
	case 2:
		debugStatus = "STEP"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorBlue)
	case 3:
		debugStatus = "FRAME"
		debugStyle = tcell.StyleDefault.Foreground(tcell.ColorRed)
	}

	// Format interrupt information
	ime := "OFF"
	if cpu.GetIME() {
		ime = "ON"
	}
	halted := ""
	if cpu.IsHalted() {
		halted = " HALT"
	}

	// Decode pending interrupts
	pending := cpu.GetPendingInterrupts()
	pendingStr := ""
	if pending&0x01 != 0 {
		pendingStr += "VBL "
	}
	if pending&0x02 != 0 {
		pendingStr += "LCD "
	}
	if pending&0x04 != 0 {
		pendingStr += "TMR "
	}
	if pending&0x08 != 0 {
		pendingStr += "SER "
	}
	if pending&0x10 != 0 {
		pendingStr += "JOY "
	}
	if pendingStr == "" {
		pendingStr = "none"
	}

	registers := []string{
		fmt.Sprintf("Status: %s%s", debugStatus, halted),
		fmt.Sprintf("A: 0x%02X  F: 0x%02X [%s]", cpu.GetA(), cpu.GetF(), cpu.GetFlagString()),
		fmt.Sprintf("B: 0x%02X  C: 0x%02X", cpu.GetB(), cpu.GetC()),
		fmt.Sprintf("D: 0x%02X  E: 0x%02X", cpu.GetD(), cpu.GetE()),
		fmt.Sprintf("H: 0x%02X  L: 0x%02X", cpu.GetH(), cpu.GetL()),
		fmt.Sprintf("SP: 0x%04X  PC: 0x%04X", cpu.GetSP(), cpu.GetPC()),
		fmt.Sprintf("IME: %s  IE: 0x%02X  IF: 0x%02X", ime, cpu.GetIE(), cpu.GetIF()),
		fmt.Sprintf("Pending: %s", pendingStr),
		fmt.Sprintf("Joypad: 0x%02X", mmu.Read(0xFF00)),
	}

	// Decode joypad state (0 = pressed, 1 = not pressed)
	buttons, directions := mmu.GetJoypadState()
	joypadStr := ""
	if buttons&0x01 == 0 {
		joypadStr += "A "
	}
	if buttons&0x02 == 0 {
		joypadStr += "B "
	}
	if buttons&0x04 == 0 {
		joypadStr += "SEL "
	}
	if buttons&0x08 == 0 {
		joypadStr += "START "
	}
	if directions&0x01 == 0 {
		joypadStr += "RIGHT "
	}
	if directions&0x02 == 0 {
		joypadStr += "LEFT "
	}
	if directions&0x04 == 0 {
		joypadStr += "UP "
	}
	if directions&0x08 == 0 {
		joypadStr += "DOWN "
	}
	if joypadStr == "" {
		joypadStr = "none"
	}

	// Add joypad and frame info to registers
	registers = append(registers,
		fmt.Sprintf("Joypad: %s", joypadStr),
		fmt.Sprintf("Frame: %d  Instr: %d", t.emulator.GetFrameCount(), t.emulator.GetInstructionCount()))

	registerEndY := min(registerHeight+1, termHeight/3)

	for i, reg := range registers {
		if startY+i >= registerEndY || startY+i >= termHeight {
			break
		}

		style := regStyle
		if i == 0 {
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
	borderX := termWidth / 2
	if termWidth > 100 {
		borderX = min(gameAreaWidth+1, termWidth*2/3)
	}
	startX := borderX + 2

	registerEndY := min(registerHeight+1, termHeight/3)
	startY := registerEndY + 2

	cpu := t.emulator.GetCPU()
	mmu := t.emulator.GetMMU()
	currentPC := cpu.GetPC()

	lines := disasm.DisassembleAround(currentPC, 3, 3, mmu)

	disasmStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	currentPCStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue)

	disasmEndY := min(registerEndY+disasmHeight+1, termHeight*2/3)
	availableLines := disasmEndY - startY
	if availableLines <= 0 {
		return
	}

	maxLines := min(len(lines), availableLines)

	// Find the current PC index to center it
	currentIndex := -1
	for i, line := range lines {
		if line.Address == currentPC {
			currentIndex = i
			break
		}
	}

	// Center the current instruction if possible
	displayOffset := 0
	if currentIndex >= 0 && availableLines > 0 {
		desiredCenter := availableLines / 2
		if currentIndex > desiredCenter {
			displayOffset = currentIndex - desiredCenter
			if displayOffset+availableLines > len(lines) {
				displayOffset = max(0, len(lines)-availableLines)
			}
		}
	}

	for i := 0; i < maxLines && displayOffset+i < len(lines); i++ {
		if startY+i >= disasmEndY-1 || startY+i >= termHeight {
			break
		}

		line := lines[displayOffset+i]
		isCurrentPC := line.Address == currentPC

		text := disasm.FormatDisassemblyLine(line, isCurrentPC)

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
	borderX := termWidth / 2
	if termWidth > 100 {
		borderX = min(gameAreaWidth+1, termWidth*2/3)
	}
	startX := borderX + 2

	registerEndY := min(registerHeight+1, termHeight/3)
	disasmEndY := min(registerEndY+disasmHeight+1, termHeight*2/3)
	startY := disasmEndY + 2
	availableHeight := termHeight - startY - 1

	if availableHeight <= 0 {
		return
	}

	// Get recent logs and filter by level
	allLogs := t.logBuffer.GetRecent(availableHeight * 2) // Get more logs to filter
	logs := make([]LogEntry, 0, availableHeight)
	for _, entry := range allLogs {
		if entry.Level >= t.logLevel {
			logs = append(logs, entry)
			if len(logs) >= availableHeight {
				break
			}
		}
	}

	debugStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	infoStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue)
	warnStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
	errStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)

	for i, logEntry := range logs {
		if i >= availableHeight {
			break
		}

		style := infoStyle
		switch logEntry.Level {
		case slog.LevelDebug:
			style = debugStyle
		case slog.LevelWarn:
			style = warnStyle
		case slog.LevelError:
			style = errStyle
		}

		logText := FormatLogEntry(logEntry)
		y := startY + i
		x := startX

		maxWidth := termWidth - startX - 1
		if len(logText) > maxWidth && maxWidth > 3 {
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
