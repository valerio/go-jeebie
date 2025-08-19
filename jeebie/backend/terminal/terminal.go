package terminal

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/disasm"
	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/memory"
	"github.com/valerio/go-jeebie/jeebie/render"
	"github.com/valerio/go-jeebie/jeebie/video"
)

const (
	width     = video.FramebufferWidth
	height    = video.FramebufferHeight
	scaleX    = 1
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

// Backend implements the Backend interface using tcell for terminal rendering
type Backend struct {
	screen    tcell.Screen
	running   bool
	logBuffer *render.LogBuffer
	logLevel  slog.Level
	callbacks backend.BackendCallbacks
	config    backend.BackendConfig

	// For accessing emulator state (will be passed via interface)
	getCPU func() CPUState
	getMMU func() MMUState

	// Test pattern state
	testPatternFrame *video.FrameBuffer
	testPatternType  int
	testFrameCount   int

	// Snapshot state
	currentFrame *video.FrameBuffer // Store current frame for snapshot generation
}

// CPUState represents the CPU state needed for debugging display
type CPUState interface {
	GetA() uint8
	GetF() uint8
	GetB() uint8
	GetC() uint8
	GetD() uint8
	GetE() uint8
	GetH() uint8
	GetL() uint8
	GetSP() uint16
	GetPC() uint16
	GetIME() bool
}

// MMUState represents the MMU state needed for debugging display
type MMUState interface {
	Read(addr uint16) uint8
}

// New creates a new terminal backend
func New() *Backend {
	return &Backend{
		logLevel: slog.LevelInfo,
	}
}

// Init initializes the terminal backend
func (t *Backend) Init(config backend.BackendConfig) error {
	t.config = config
	t.callbacks = config.Callbacks

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to initialize terminal: %v", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize terminal: %v", err)
	}

	t.screen = screen
	t.running = true

	// Create log buffer and set up logging
	t.logBuffer = render.NewLogBuffer(100)

	// Set up the log handler to capture logs
	handler := render.NewLogBufferHandler(t.logBuffer, slog.LevelDebug)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Add some initial test logs
	if config.TestPattern {
		t.testPatternFrame = video.NewFrameBuffer()
		t.generateTestPattern(0) // Start with checkerboard
		slog.Info("Terminal backend initialized in test pattern mode")
	} else {
		slog.Info("Terminal backend initialized")
		if config.ShowDebug {
			slog.Debug("Debug mode enabled")
		}
	}

	t.screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	t.screen.Clear()

	// Set up signal handling for graceful shutdown
	go t.handleSignals()
	go t.handleInput()

	return nil
}

// Update renders a frame and processes events
func (t *Backend) Update(frame *video.FrameBuffer) error {
	if !t.running {
		return nil
	}

	// Use test pattern frame if in test pattern mode
	renderFrame := frame
	if t.config.TestPattern {
		t.testFrameCount++
		// Animate test pattern occasionally
		if t.testFrameCount%display.TestPatternAnimationFrames == 0 {
			t.animateTestPattern()
		}
		renderFrame = t.testPatternFrame
	}

	// Store current frame for snapshots and render
	t.currentFrame = renderFrame
	t.render(renderFrame)
	t.screen.Show()

	return nil
}

// Cleanup cleans up terminal resources
func (t *Backend) Cleanup() error {
	if t.screen != nil {
		slog.Info("Cleaning up terminal backend")
		t.screen.Fini()
	}
	return nil
}

// SetEmulatorState allows the backend to access emulator state for debugging
func (t *Backend) SetEmulatorState(getCPU func() CPUState, getMMU func() MMUState) {
	t.getCPU = getCPU
	t.getMMU = getMMU
}

func (t *Backend) handleSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	<-signals
	t.running = false
	if t.callbacks.OnQuit != nil {
		t.callbacks.OnQuit()
	}
}

func (t *Backend) handleInput() {
	for t.running {
		ev := t.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			t.handleKeyEvent(ev)
		case *tcell.EventResize:
			t.screen.Sync()
		}
	}
}

func (t *Backend) handleKeyEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		t.running = false
		if t.callbacks.OnQuit != nil {
			t.callbacks.OnQuit()
		}
		return
	case tcell.KeyF12:
		debug.TakeSnapshot(t.currentFrame, t.config.TestPattern, t.testPatternType)
	case tcell.KeyEnter:
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadStart)
		}
	case tcell.KeyRight:
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadRight)
		}
	case tcell.KeyLeft:
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadLeft)
		}
	case tcell.KeyUp:
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadUp)
		}
	case tcell.KeyDown:
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadDown)
		}
	case tcell.KeyRune:
		t.handleRuneKey(ev.Rune())
	}
}

func (t *Backend) handleRuneKey(r rune) {
	if t.config.TestPattern {
		// Test pattern specific controls
		switch r {
		case 't': // 't' key - cycle test patterns
			t.testPatternType = (t.testPatternType + 1) % display.TestPatternCount
			t.generateTestPattern(t.testPatternType)
			patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
			slog.Info("Switched to test pattern", "pattern", patternNames[t.testPatternType])
		}
		return
	}

	// Normal emulator controls
	switch r {
	case 'a':
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadA)
		}
	case 's':
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadB)
		}
	case 'q':
		if t.callbacks.OnKeyPress != nil {
			t.callbacks.OnKeyPress(memory.JoypadSelect)
		}

	case ' ': // Spacebar - pause/resume toggle
		if t.callbacks.OnDebugMessage != nil {
			t.callbacks.OnDebugMessage("debug:toggle_pause")
		}
	case 'n': // Next instruction (step)
		if t.callbacks.OnDebugMessage != nil {
			t.callbacks.OnDebugMessage("debug:step_instruction")
		}
	case 'f': // Next frame (step frame)
		if t.callbacks.OnDebugMessage != nil {
			t.callbacks.OnDebugMessage("debug:step_frame")
		}
	case 'r': // Resume
		if t.callbacks.OnDebugMessage != nil {
			t.callbacks.OnDebugMessage("debug:resume")
		}
	case 'p': // Pause
		if t.callbacks.OnDebugMessage != nil {
			t.callbacks.OnDebugMessage("debug:pause")
		}
	case '-', '_': // Decrease log verbosity
		t.changeLogLevel(-1)
	case '+', '=': // Increase log verbosity
		t.changeLogLevel(1)
	}
}

func (t *Backend) changeLogLevel(direction int) {
	oldLevel := t.logLevel
	switch direction {
	case -1: // Decrease verbosity (show fewer logs)
		switch t.logLevel {
		case slog.LevelDebug:
			t.logLevel = slog.LevelInfo
		case slog.LevelInfo:
			t.logLevel = slog.LevelWarn
		case slog.LevelWarn:
			t.logLevel = slog.LevelError
		}
	case 1: // Increase verbosity (show more logs)
		switch t.logLevel {
		case slog.LevelError:
			t.logLevel = slog.LevelWarn
		case slog.LevelWarn:
			t.logLevel = slog.LevelInfo
		case slog.LevelInfo:
			t.logLevel = slog.LevelDebug
		}
	}
	if oldLevel != t.logLevel {
		slog.Info("Log filter changed", "from", oldLevel, "to", t.logLevel)
	}
}

func (t *Backend) render(frame *video.FrameBuffer) {
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

	gbScreenWidth := width
	dividerX := gbScreenWidth + 2
	rightPanelX := dividerX + 1
	rightPanelWidth := termWidth - rightPanelX
	if rightPanelWidth < 0 {
		rightPanelWidth = 0
	}

	t.drawBorders(termWidth, termHeight, dividerX)
	t.drawGameBoy(frame)

	if t.config.ShowDebug && t.getCPU != nil && t.getMMU != nil {
		t.drawRegisters(rightPanelX, 1, rightPanelWidth, termHeight)
		disasmY := registerHeight + 2
		t.drawDisassembly(rightPanelX, disasmY, rightPanelWidth, termHeight)
	}

	logsY := registerHeight + disasmHeight + 3
	if !t.config.ShowDebug {
		logsY = 1
	}
	t.drawLogs(rightPanelX, logsY, rightPanelWidth, termHeight)
}

func (t *Backend) drawBorders(termWidth, termHeight, dividerX int) {
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

	// horizontal line after registers
	if registerEndY < termHeight && t.config.ShowDebug {
		for x := dividerX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, registerEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(dividerX, registerEndY, '├', nil, borderStyle)
	}

	// horizontal line after disassembly
	if disasmEndY < termHeight && t.config.ShowDebug {
		for x := dividerX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, disasmEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(dividerX, disasmEndY, '├', nil, borderStyle)
	}

	var title string
	if t.config.TestPattern {
		patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
		title = fmt.Sprintf(" Test Pattern: %s ", patternNames[t.testPatternType])
	} else {
		title = " Game Boy "
	}
	for i, ch := range title {
		if i+1 < dividerX {
			t.screen.SetContent(1+i, 0, ch, nil, titleStyle)
		}
	}

	if t.config.ShowDebug {
		title = " CPU Registers "
		startX := dividerX + 2
		for i, ch := range title {
			if startX+i < termWidth {
				t.screen.SetContent(startX+i, 0, ch, nil, titleStyle)
			}
		}

		if registerEndY+1 < termHeight {
			title = " Disassembly "
			for i, ch := range title {
				if startX+i < termWidth {
					t.screen.SetContent(startX+i, disasmEndY+1, ch, nil, titleStyle)
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
				if startX+i < termWidth {
					t.screen.SetContent(startX+i, disasmEndY+1, ch, nil, titleStyle)
				}
			}
		}
	}

	helpY := termHeight - 1
	var helpText string
	if t.config.TestPattern {
		helpText = " Test Pattern Mode: T=cycle patterns F12=snapshot ESC=exit "
	} else {
		helpText = " Debug: SPACE=pause/resume N=step F=frame F12=snapshot | Logs: +/- filter "
	}
	for i, ch := range helpText {
		if i < termWidth {
			t.screen.SetContent(i, helpY, ch, nil, borderStyle)
		}
	}
}

func (t *Backend) drawGameBoy(frame *video.FrameBuffer) {
	frameData := frame.ToSlice()

	// process two rows at a time using half-blocks
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			topPixel := frameData[y*width+x]
			bottomPixel := uint32(0xFFFFFFFF) // default to white if out of bounds
			if y+1 < height {
				bottomPixel = frameData[(y+1)*width+x]
			}

			topShade := render.PixelToShade(topPixel)
			bottomShade := render.PixelToShade(bottomPixel)

			char, fg, bg := getHalfBlockChar(topShade, bottomShade)

			style := tcell.StyleDefault.Foreground(fg).Background(bg)
			screenX := x * scaleX
			screenY := y/2 + 1
			t.screen.SetContent(screenX, screenY, char, nil, style)
		}
	}
}

// getHalfBlockChar returns the appropriate half-block character and colors for terminal
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
	char := render.GetHalfBlockChar(topShade, bottomShade)

	if topShade == bottomShade {
		return char, topColor, tcell.ColorDefault
	} else if topShade == 3 && bottomShade != 3 {
		return char, bottomColor, topColor
	} else if topShade != 3 && bottomShade == 3 {
		return char, topColor, bottomColor
	} else {
		return char, topColor, bottomColor
	}
}

func (t *Backend) drawRegisters(startX, startY, width, termHeight int) {
	if t.getCPU == nil || t.getMMU == nil {
		return
	}

	cpu := t.getCPU()
	mmu := t.getMMU()

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
	}

	style := tcell.StyleDefault.Foreground(tcell.ColorBlue)
	for i, line := range lines {
		y := startY + i
		if y >= termHeight || y >= startY+registerHeight {
			break
		}

		if len(line) > width {
			line = line[:width]
		}

		x := startX
		for j, ch := range line {
			if j >= width || x >= startX+width || x >= 300 {
				break
			}
			t.screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}

func (t *Backend) drawDisassembly(startX, startY, width, termHeight int) {
	if t.getCPU == nil || t.getMMU == nil {
		return
	}

	cpu := t.getCPU()
	mmu := t.getMMU()

	if width <= 0 || startY >= termHeight {
		return
	}

	pc := cpu.GetPC()
	halfHeight := disasmHeight / 2

	// cast to *memory.MMU since disasm functions expect that type
	var lines []disasm.DisassemblyLine
	if realMMU, ok := mmu.(*memory.MMU); ok {
		lines = disasm.DisassembleAround(pc, halfHeight, disasmHeight-halfHeight-1, realMMU)
		if len(lines) == 0 {
			lines = disasm.DisassembleRange(pc, disasmHeight, realMMU)
		}
	} else {
		lines = []disasm.DisassemblyLine{
			{Address: pc, Instruction: "???"},
		}
	}

	style := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	currentStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	for i, disasmLine := range lines {
		y := startY + i
		if y >= termHeight || y >= startY+disasmHeight || i >= disasmHeight {
			break
		}

		line := fmt.Sprintf(" 0x%04X: %s", disasmLine.Address, disasmLine.Instruction)

		if disasmLine.Address == pc {
			line = "→" + line[1:]
		}

		if len(line) > width {
			line = line[:width]
		}

		useStyle := style
		if disasmLine.Address == pc {
			useStyle = currentStyle
		}

		x := startX
		for j, ch := range line {
			if j >= width || x >= startX+width {
				break
			}
			t.screen.SetContent(x, y, ch, nil, useStyle)
			x++
		}
	}
}

func (t *Backend) drawLogs(startX, startY, width, termHeight int) {
	if width <= 0 || startY >= termHeight {
		return
	}

	availableHeight := termHeight - startY - 1
	if availableHeight <= 0 {
		return
	}

	allLogs := t.logBuffer.GetRecent(availableHeight * 2)
	logs := make([]render.LogEntry, 0, availableHeight)
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

		logText := render.FormatLogEntry(logEntry)
		y := startY + i

		if y >= termHeight-1 {
			break
		}

		if len(logText) > width {
			if width > 3 {
				logText = logText[:width-3] + "..."
			} else if width > 0 {
				logText = logText[:width]
			}
		}

		x := startX
		for j, ch := range logText {
			if j >= width || x >= startX+width {
				break
			}
			t.screen.SetContent(x, y, ch, nil, style)
			x++
		}
	}
}

// generateTestPattern creates different test patterns
func (t *Backend) generateTestPattern(patternType int) {
	switch patternType {
	case 0: // Checkerboard
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x/display.TestPatternTileSize)+(y/display.TestPatternTileSize))%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.BlackColor
				}
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 1: // Gradient
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				gray := uint32(x * display.GrayscaleWhite / video.FramebufferWidth)
				color := video.GBColor((gray << display.RGBARShift) | (gray << display.RGBAGShift) | (gray << display.RGBABShift) | display.FullAlpha)
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 2: // Vertical stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if (x/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Diagonal lines
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}

// animateTestPattern provides simple animation for test patterns
func (t *Backend) animateTestPattern() {
	frame := t.testFrameCount / display.TestPatternAnimationFrames
	switch t.testPatternType {
	case 2: // Animate stripes
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+frame*display.TestPatternStripeSpeed)/display.TestPatternStripeWidth)%2 == 0 {
					color = video.WhiteColor
				} else {
					color = video.DarkGreyColor
				}
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 3: // Animate diagonal
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				var color video.GBColor
				if ((x+y+frame*display.TestPatternDiagonalSpeed)/display.TestPatternTileSize)%2 == 0 {
					color = video.LightGreyColor
				} else {
					color = video.DarkGreyColor
				}
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	}
}
