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

	// Calculate layout dynamically based on half-block rendering
	// Game Boy screen now uses 72 rows (144/2) plus 1 for title
	// gbScreenHeight := height/2 + 1 // 73 rows total (unused for now)
	gbScreenWidth := width // 160 columns

	// Position of vertical divider (after game screen + small margin)
	dividerX := gbScreenWidth + 2

	// Right panel starts after divider
	rightPanelX := dividerX + 1
	// Ensure we don't exceed terminal boundaries
	rightPanelWidth := termWidth - rightPanelX
	if rightPanelWidth < 0 {
		rightPanelWidth = 0
	}

	// Draw borders and sections with calculated positions
	t.drawBorders(termWidth, termHeight, dividerX)

	// Draw Game Boy screen (left side)
	t.drawGameBoy()

	// Draw CPU registers (top-right)
	t.drawRegisters(rightPanelX, 1, rightPanelWidth, termHeight)

	// Draw disassembly (middle-right)
	disasmY := registerHeight + 2
	t.drawDisassembly(rightPanelX, disasmY, rightPanelWidth, termHeight)

	// Draw logs (bottom-right)
	logsY := disasmY + disasmHeight + 1
	t.drawLogs(rightPanelX, logsY, rightPanelWidth, termHeight)
}

func (t *TerminalRenderer) drawBorders(termWidth, termHeight, dividerX int) {
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	// Draw vertical divider
	for y := 0; y < termHeight; y++ {
		if dividerX < termWidth {
			t.screen.SetContent(dividerX, y, '│', nil, borderStyle)
		}
	}

	// Draw horizontal dividers for right panel sections
	registerEndY := registerHeight + 1
	disasmEndY := registerEndY + disasmHeight + 1

	// Horizontal line after registers
	if registerEndY < termHeight {
		for x := dividerX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, registerEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(dividerX, registerEndY, '├', nil, borderStyle)
	}

	// Horizontal line after disassembly
	if disasmEndY < termHeight {
		for x := dividerX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, disasmEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(dividerX, disasmEndY, '├', nil, borderStyle)
	}

	// Draw section titles
	// Game Boy title
	title := " Game Boy "
	for i, ch := range title {
		if i+1 < dividerX {
			t.screen.SetContent(1+i, 0, ch, nil, titleStyle)
		}
	}

	// CPU Registers title
	title = " CPU Registers "
	startX := dividerX + 2
	for i, ch := range title {
		if startX+i < termWidth {
			t.screen.SetContent(startX+i, 0, ch, nil, titleStyle)
		}
	}

	// Disassembly title
	if registerEndY+1 < termHeight {
		title = " Disassembly "
		for i, ch := range title {
			if startX+i < termWidth {
				t.screen.SetContent(startX+i, registerEndY+1, ch, nil, titleStyle)
			}
		}
	}

	// Logs title with filter level
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
			if startX+i < termWidth {
				t.screen.SetContent(startX+i, disasmEndY+1, ch, nil, titleStyle)
			}
		}
	}

	// Draw bottom help text
	helpY := termHeight - 1
	helpText := " Debug: SPACE=pause/resume N=step F=frame | Logs: +/- filter "
	for i, ch := range helpText {
		if i < termWidth {
			t.screen.SetContent(i, helpY, ch, nil, borderStyle)
		}
	}
}

func (t *TerminalRenderer) drawGameBoy() {
	// Use half-block rendering for better vertical fit
	t.drawGameBoyHalfBlock()
}

// drawGameBoyHalfBlock renders using Unicode half-blocks (▀ and ▄)
// to display two pixel rows per terminal row
func (t *TerminalRenderer) drawGameBoyHalfBlock() {
	fb := t.emulator.GetCurrentFrame()
	frame := fb.ToSlice()

	// Process two rows at a time
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			// Get top and bottom pixels
			topPixel := frame[y*width+x]
			bottomPixel := uint32(0xFFFFFFFF) // Default to white if out of bounds
			if y+1 < height {
				bottomPixel = frame[(y+1)*width+x]
			}

			// Convert pixels to shade values
			topShade := pixelToShade(topPixel)
			bottomShade := pixelToShade(bottomPixel)

			// Determine character and colors based on shade combination
			char, fg, bg := getHalfBlockChar(topShade, bottomShade)

			style := tcell.StyleDefault.Foreground(fg).Background(bg)
			screenX := x * scaleX
			screenY := y/2 + 1 // Half the vertical space

			t.screen.SetContent(screenX, screenY, char, nil, style)
		}
	}
}

// pixelToShade converts a pixel value to a shade level (0-3)
func pixelToShade(pixel uint32) int {
	switch pixel {
	case 0x000000FF:
		return 0 // Black
	case 0x4C4C4CFF:
		return 1 // Dark gray
	case 0x989898FF:
		return 2 // Light gray
	case 0xFFFFFFFF:
		return 3 // White
	default:
		return 0
	}
}

// getHalfBlockChar returns the appropriate half-block character and colors
func getHalfBlockChar(topShade, bottomShade int) (rune, tcell.Color, tcell.Color) {
	// Map Game Boy shades to terminal colors
	shadeColors := []tcell.Color{
		tcell.ColorBlack,
		tcell.ColorGray,
		tcell.ColorSilver,
		tcell.ColorWhite,
	}

	topColor := shadeColors[topShade]
	bottomColor := shadeColors[bottomShade]

	if topShade == bottomShade {
		// Both pixels same shade - use full block
		return '█', topColor, tcell.ColorDefault
	} else if topShade == 3 && bottomShade != 3 {
		// Top white, bottom not - use lower half block
		return '▄', bottomColor, topColor
	} else if topShade != 3 && bottomShade == 3 {
		// Top not white, bottom white - use upper half block
		return '▀', topColor, bottomColor
	} else {
		// Mixed shades - use upper half block with appropriate colors
		return '▀', topColor, bottomColor
	}
}

func (t *TerminalRenderer) drawRegisters(startX, startY, width, termHeight int) {
	cpu := t.emulator.GetCPU()
	mmu := t.emulator.GetMMU()

	if width <= 0 || startY >= termHeight {
		return
	}

	lines := []string{
		"Status: RUNNING",
		fmt.Sprintf("A: 0x%02X  F: 0x%02X", cpu.GetA(), cpu.GetF()),
		fmt.Sprintf("B: 0x%02X  C: 0x%02X", cpu.GetB(), cpu.GetC()),
		fmt.Sprintf("D: 0x%02X  E: 0x%02X", cpu.GetD(), cpu.GetE()),
		fmt.Sprintf("H: 0x%02X  L: 0x%02X", cpu.GetH(), cpu.GetL()),
		fmt.Sprintf("SP: 0x%04X  PC: 0x%04X", cpu.GetSP(), cpu.GetPC()),
		fmt.Sprintf("IME: %s  IE: 0x%02X  IF: 0x%02X",
			map[bool]string{true: "ON", false: "OFF"}[cpu.GetIME()],
			mmu.Read(0xFFFF), mmu.Read(0xFF0F)),
		"Pending: none",
		fmt.Sprintf("Joypad: 0x%02X", mmu.Read(0xFF00)),
		fmt.Sprintf("Frame: %d", t.emulator.GetFrameCount()),
	}

	style := tcell.StyleDefault.Foreground(tcell.ColorBlue)
	for i, line := range lines {
		y := startY + i
		if y >= termHeight || y >= startY+registerHeight {
			break
		}

		// Truncate line if too long
		if len(line) > width {
			line = line[:width]
		}

		x := startX
		for j, ch := range line {
			// Double check we don't exceed width OR terminal bounds
			if j >= width || x >= startX+width || x >= 300 { // 300 is a safety max
				break
			}
			t.screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}

func (t *TerminalRenderer) drawDisassembly(startX, startY, width, termHeight int) {
	cpu := t.emulator.GetCPU()
	mmu := t.emulator.GetMMU()

	if width <= 0 || startY >= termHeight {
		return
	}

	pc := cpu.GetPC()

	// Calculate how many instructions to show before and after current PC
	// Aim to center the current instruction
	halfHeight := disasmHeight / 2

	// Use DisassembleAround to get context before and after PC
	// This function handles finding the right starting point
	lines := disasm.DisassembleAround(pc, halfHeight, disasmHeight-halfHeight-1, mmu)

	// If we couldn't get enough context (e.g., near address boundaries),
	// fall back to simple forward disassembly
	if len(lines) == 0 {
		lines = disasm.DisassembleRange(pc, disasmHeight, mmu)
	}

	style := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	currentStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	for i, disasmLine := range lines {
		y := startY + i
		if y >= termHeight || y >= startY+disasmHeight || i >= disasmHeight {
			break
		}

		line := fmt.Sprintf(" 0x%04X: %s", disasmLine.Address, disasmLine.Instruction)

		// Add arrow for current PC
		if disasmLine.Address == pc {
			line = "→" + line[1:]
		}

		// Truncate if too long
		if len(line) > width {
			line = line[:width]
		}

		useStyle := style
		if disasmLine.Address == pc {
			useStyle = currentStyle
		}

		x := startX
		for j, ch := range line {
			// Strict boundary checking
			if j >= width || x >= startX+width {
				break
			}
			t.screen.SetContent(x, y, ch, nil, useStyle)
			x++
		}
	}
}

func (t *TerminalRenderer) drawLogs(startX, startY, width, termHeight int) {
	if width <= 0 || startY >= termHeight {
		return
	}

	availableHeight := termHeight - startY - 1 // Leave room for bottom help text
	if availableHeight <= 0 {
		return
	}

	// Get recent logs and filter by level
	allLogs := t.logBuffer.GetRecent(availableHeight * 2)
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

		// Ensure y is within bounds
		if y >= termHeight-1 { // Leave room for help text
			break
		}

		// Truncate the text if needed
		if len(logText) > width {
			if width > 3 {
				logText = logText[:width-3] + "..."
			} else if width > 0 {
				logText = logText[:width]
			}
		}

		x := startX
		for j, ch := range logText {
			// Strict boundary checking to prevent overflow
			if j >= width || x >= startX+width {
				break
			}
			t.screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}
