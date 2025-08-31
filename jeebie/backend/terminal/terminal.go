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
	"github.com/valerio/go-jeebie/jeebie/backend/terminal/render"
	"github.com/valerio/go-jeebie/jeebie/debug"
	"github.com/valerio/go-jeebie/jeebie/disasm"
	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
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

// Backend implements the Backend interface using tcell for terminal rendering
type Backend struct {
	screen     tcell.Screen
	running    bool
	logBuffer  *render.LogBuffer
	logLevel   slog.Level
	config     backend.BackendConfig
	eventQueue []backend.InputEvent // Collect events to return

	keyStates  map[action.Action]time.Time // Last time each key was pressed
	activeKeys map[action.Action]bool      // Keys active in previous frame

	// For accessing emulator state
	debugProvider backend.DebugDataProvider

	// Test pattern state
	testPatternFrame *video.FrameBuffer
	testPatternType  int
	testFrameCount   int

	// Snapshot state
	currentFrame *video.FrameBuffer // Store current frame for snapshot generation
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
	t.debugProvider = config.DebugProvider
	t.eventQueue = make([]backend.InputEvent, 0)
	t.keyStates = make(map[action.Action]time.Time)
	t.activeKeys = make(map[action.Action]bool)

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

	return nil
}

// Key expiry timeout - slightly longer than typical key repeat interval
const keyTimeout = 100 * time.Millisecond

// Update renders a frame and processes events
func (t *Backend) Update(frame *video.FrameBuffer) ([]backend.InputEvent, error) {
	var events []backend.InputEvent
	now := time.Now()

	// Poll for input events synchronously
	for t.screen.HasPendingEvent() {
		ev := t.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			t.processKeyEvent(ev, now)
		case *tcell.EventResize:
			t.screen.Sync()
		}
	}

	// Track which keys are currently active this frame
	currentlyActive := make(map[action.Action]bool)

	// Check all tracked keys and generate appropriate events
	for act, lastPressed := range t.keyStates {
		info := action.GetInfo(act)

		// Skip non-game inputs (they're handled via eventQueue)
		if info.Category != action.CategoryGameInput {
			continue
		}

		if now.Sub(lastPressed) < keyTimeout {
			// Key is still active
			currentlyActive[act] = true

			if !t.activeKeys[act] {
				// Was not active last frame - send Press
				slog.Debug("Key press", "action", info.Description)
				events = append(events, backend.InputEvent{Action: act, Type: event.Press})
			} else {
				// Was active last frame - send Hold
				events = append(events, backend.InputEvent{Action: act, Type: event.Hold})
			}
		} else {
			// Key has expired - remove it
			delete(t.keyStates, act)
		}
	}

	// Check for released keys (were active last frame but not this frame)
	for act := range t.activeKeys {
		if !currentlyActive[act] {
			info := action.GetInfo(act)
			slog.Debug("Key release", "action", info.Description)
			events = append(events, backend.InputEvent{Action: act, Type: event.Release})
		}
	}

	// Update active keys for next frame
	t.activeKeys = currentlyActive

	// Add non-game input events (pause, debug, etc)
	if len(t.eventQueue) > 0 {
		for _, evt := range t.eventQueue {
			info := action.GetInfo(evt.Action)
			slog.Debug("UI event", "action", info.Description, "type", evt.Type)
		}
		events = append(events, t.eventQueue...)
	}
	t.eventQueue = nil

	if !t.running {
		return events, nil
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

	return events, nil
}

// Cleanup cleans up terminal resources
func (t *Backend) Cleanup() error {
	if t.screen != nil {
		slog.Info("Cleaning up terminal backend")
		t.screen.Fini()
	}
	return nil
}

// HandleAction processes backend-specific actions
func (t *Backend) HandleAction(act action.Action) {
	switch act {
	case action.EmulatorSnapshot:
		debug.TakeSnapshot(t.currentFrame, t.config.TestPattern, t.testPatternType)
	case action.EmulatorTestPatternCycle:
		if t.config.TestPattern {
			t.testPatternType = (t.testPatternType + 1) % display.TestPatternCount
			t.generateTestPattern(t.testPatternType)
			patternNames := []string{"Checkerboard", "Gradient", "Stripes", "Diagonal"}
			slog.Info("Switched to test pattern", "pattern", patternNames[t.testPatternType])
		}
	case action.EmulatorDebugToggle:
		t.config.ShowDebug = !t.config.ShowDebug
		if t.config.ShowDebug {
			slog.Info("Debug display enabled")
		} else {
			slog.Info("Debug display disabled")
		}
	case action.EmulatorDebugUpdate:
		// Force a screen refresh/update
		t.screen.Sync()
	case action.DebugLogLevelIncrease:
		t.changeLogLevel(1)
	case action.DebugLogLevelDecrease:
		t.changeLogLevel(-1)
	// Terminal doesn't support audio actions currently, but logs them
	case action.AudioToggleChannel1, action.AudioToggleChannel2,
		action.AudioToggleChannel3, action.AudioToggleChannel4,
		action.AudioSoloChannel1, action.AudioSoloChannel2,
		action.AudioSoloChannel3, action.AudioSoloChannel4,
		action.AudioShowStatus:
		slog.Debug("Audio action not supported in terminal backend", "action", act)
	}
}

func (t *Backend) handleSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	<-signals
	t.running = false
	// Signal quit via event queue
	t.eventQueue = append(t.eventQueue, backend.InputEvent{Action: action.EmulatorQuit, Type: event.Press})
}

func (t *Backend) processKeyEvent(ev *tcell.EventKey, now time.Time) {
	if act, exists := keyMapping[ev.Key()]; exists {
		if act == action.EmulatorQuit {
			t.running = false
		}
		info := action.GetInfo(act)
		if info.Category == action.CategoryGameInput {
			// For game inputs, clear other directional inputs if this is a d-pad action
			if act == action.GBDPadUp || act == action.GBDPadDown ||
				act == action.GBDPadLeft || act == action.GBDPadRight {
				// Clear all d-pad directions to simulate exclusive directions
				delete(t.keyStates, action.GBDPadUp)
				delete(t.keyStates, action.GBDPadDown)
				delete(t.keyStates, action.GBDPadLeft)
				delete(t.keyStates, action.GBDPadRight)
			}
			t.keyStates[act] = now
		} else {
			t.eventQueue = append(t.eventQueue, backend.InputEvent{Action: act, Type: event.Press})
		}
		return
	}

	if ev.Key() == tcell.KeyRune {
		t.processRuneKey(ev.Rune(), now)
	}
}

// tcellKeyNameMap converts tcell keys to key names used in default mappings
var tcellKeyNameMap = map[tcell.Key]string{
	tcell.KeyEnter:  "Enter",
	tcell.KeyUp:     "Up",
	tcell.KeyDown:   "Down",
	tcell.KeyLeft:   "Left",
	tcell.KeyRight:  "Right",
	tcell.KeyEscape: "Escape",
	tcell.KeyF1:     "F1",
	tcell.KeyF2:     "F2",
	tcell.KeyF3:     "F3",
	tcell.KeyF4:     "F4",
	tcell.KeyF5:     "F5",
	tcell.KeyF9:     "F9",
	tcell.KeyF10:    "F10",
	tcell.KeyF11:    "F11",
	tcell.KeyF12:    "F12",
}

// tcellRuneNameMap converts runes to key names used in default mappings
var tcellRuneNameMap = map[rune]string{
	'z': "z",
	'x': "x",
	'w': "w",
	's': "s",
	'a': "a",
	'd': "d",
	'p': "p",
	'r': "r",
	'o': "o",
	'f': "f",
	'i': "i",
	'n': "n",
	'q': "q",
	' ': "Space",
	't': "t",
	'1': "1",
	'2': "2",
	'3': "3",
	'4': "4",
	'+': "+",
	'=': "=",
	'-': "-",
	'_': "_",
}

// buildKeyMapping creates the key mapping from default mappings
func buildKeyMapping() map[tcell.Key]action.Action {
	mapping := make(map[tcell.Key]action.Action)

	// Apply default mappings for special keys
	for key, keyName := range tcellKeyNameMap {
		if act, ok := input.GetDefaultMapping(keyName); ok {
			mapping[key] = act
		}
	}

	mapping[tcell.KeyCtrlC] = action.EmulatorQuit

	return mapping
}

// buildRuneMapping creates the rune mapping from default mappings
func buildRuneMapping() map[rune]action.Action {
	mapping := make(map[rune]action.Action)

	for r, keyName := range tcellRuneNameMap {
		if act, ok := input.GetDefaultMapping(keyName); ok {
			mapping[r] = act
		}
	}

	return mapping
}

// keyMapping maps tcell keys to actions
var keyMapping = buildKeyMapping()

// runeMapping maps runes to actions
var runeMapping = buildRuneMapping()

func (t *Backend) processRuneKey(r rune, now time.Time) {
	// Handle mapped runes
	if act, exists := runeMapping[r]; exists {
		// Check if this is a game input that needs state tracking
		info := action.GetInfo(act)
		slog.Debug("Key event (rune)", "rune", string(r), "action", info.Description, "category", info.Category)

		if info.Category == action.CategoryGameInput {
			// For game inputs using WASD, clear other directional inputs
			if act == action.GBDPadUp || act == action.GBDPadDown ||
				act == action.GBDPadLeft || act == action.GBDPadRight {
				// Clear all d-pad directions to simulate exclusive directions
				delete(t.keyStates, action.GBDPadUp)
				delete(t.keyStates, action.GBDPadDown)
				delete(t.keyStates, action.GBDPadLeft)
				delete(t.keyStates, action.GBDPadRight)
			}
			t.keyStates[act] = now
		} else {
			t.eventQueue = append(t.eventQueue, backend.InputEvent{Action: act, Type: event.Press})
		}
		return
	}
}

func (t *Backend) changeLogLevel(direction int) {
	oldLevel := t.logLevel
	switch direction {
	case -1:
		switch t.logLevel {
		case slog.LevelDebug:
			t.logLevel = slog.LevelInfo
		case slog.LevelInfo:
			t.logLevel = slog.LevelWarn
		case slog.LevelWarn:
			t.logLevel = slog.LevelError
		}
	case 1:
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

	if t.config.ShowDebug && t.debugProvider != nil {
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

	for y := 0; y < termHeight; y++ {
		if dividerX < termWidth {
			t.screen.SetContent(dividerX, y, '│', nil, borderStyle)
		}
	}

	registerEndY := registerHeight + 1
	disasmEndY := registerEndY + disasmHeight + 1

	if registerEndY < termHeight && t.config.ShowDebug {
		for x := dividerX + 1; x < termWidth; x++ {
			t.screen.SetContent(x, registerEndY, '─', nil, borderStyle)
		}
		t.screen.SetContent(dividerX, registerEndY, '├', nil, borderStyle)
	}

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
		helpText = " Debug: F10=toggle debug view SPACE=pause/resume N=step F=frame F12=snapshot | Logs: +/- filter "
	}
	for i, ch := range helpText {
		if i < termWidth {
			t.screen.SetContent(i, helpY, ch, nil, borderStyle)
		}
	}
}

func (t *Backend) drawGameBoy(frame *video.FrameBuffer) {
	frameData := frame.ToSlice()
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			topPixel := frameData[y*width+x]
			bottomPixel := uint32(0xFFFFFFFF)
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

func getHalfBlockChar(topShade, bottomShade int) (rune, tcell.Color, tcell.Color) {
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
	if t.debugProvider == nil {
		return
	}

	debugData := t.debugProvider.ExtractDebugData()
	if debugData == nil || debugData.CPU == nil {
		return
	}

	cpu := debugData.CPU

	if width <= 0 || startY >= termHeight {
		return
	}

	statusStr := "RUNNING"
	switch debugData.DebuggerState {
	case debug.DebuggerPaused:
		statusStr = "PAUSED"
	case debug.DebuggerStepInstruction:
		statusStr = "STEP"
	case debug.DebuggerStepFrame:
		statusStr = "FRAME"
	}

	lines := []string{
		fmt.Sprintf("Status: %s", statusStr),
		fmt.Sprintf("A: 0x%02X  F: 0x%02X", cpu.A, cpu.F),
		fmt.Sprintf("B: 0x%02X  C: 0x%02X", cpu.B, cpu.C),
		fmt.Sprintf("D: 0x%02X  E: 0x%02X", cpu.D, cpu.E),
		fmt.Sprintf("H: 0x%02X  L: 0x%02X", cpu.H, cpu.L),
		fmt.Sprintf("SP: 0x%04X  PC: 0x%04X", cpu.SP, cpu.PC),
		fmt.Sprintf("IME: %s  IE: 0x%02X  IF: 0x%02X",
			map[bool]string{true: "ON", false: "OFF"}[cpu.IME],
			debugData.InterruptEnable, debugData.InterruptFlags),
		"Pending: none",
		fmt.Sprintf("Cycles: %d", cpu.Cycles),
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
	if t.debugProvider == nil {
		return
	}

	debugData := t.debugProvider.ExtractDebugData()
	if debugData == nil || debugData.CPU == nil || debugData.Memory == nil {
		return
	}

	if width <= 0 || startY >= termHeight {
		return
	}

	pc := debugData.CPU.PC

	lines := t.createSimpleDisassembly(debugData.Memory, pc)

	style := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	currentStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	displayedLines := 0
	for _, disasmLine := range lines {
		if displayedLines >= disasmHeight {
			break
		}

		y := startY + displayedLines
		if y >= termHeight || y >= startY+disasmHeight {
			break
		}

		line := fmt.Sprintf(" 0x%04X: %s", disasmLine.address, disasmLine.instruction)

		if disasmLine.address == pc {
			line = "→" + line[1:]
		}

		if len(line) > width {
			line = line[:width]
		}

		useStyle := style
		if disasmLine.address == pc {
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

		displayedLines++
	}
}

type simpleDisasmLine struct {
	address     uint16
	instruction string
}

func (t *Backend) createSimpleDisassembly(snapshot *debug.MemorySnapshot, pc uint16) []simpleDisasmLine {
	pcOffset := -1
	if pc >= snapshot.StartAddr && pc < snapshot.StartAddr+uint16(len(snapshot.Bytes)) {
		pcOffset = int(pc - snapshot.StartAddr)
	}

	if pcOffset < 0 {
		lines := []simpleDisasmLine{}
		for i := 0; i < len(snapshot.Bytes) && len(lines) < disasmHeight; {
			addr := snapshot.StartAddr + uint16(i)
			instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)
			lines = append(lines, simpleDisasmLine{
				address:     addr,
				instruction: instruction,
			})
			i += length
		}
		return lines
	}

	allLines := []simpleDisasmLine{}

	backwardBytes := 30
	startOffset := pcOffset - backwardBytes
	if startOffset < 0 {
		startOffset = 0
	}

	for i := startOffset; i < len(snapshot.Bytes); {
		addr := snapshot.StartAddr + uint16(i)
		instruction, length := disasm.DisassembleBytes(snapshot.Bytes, i)

		allLines = append(allLines, simpleDisasmLine{
			address:     addr,
			instruction: instruction,
		})

		i += length
		if addr > pc && len(allLines) > disasmHeight*2 {
			break
		}
	}

	pcIndex := -1
	for i, line := range allLines {
		if line.address == pc {
			pcIndex = i
			break
		}
	}

	if pcIndex >= 0 {
		halfHeight := disasmHeight / 2
		startIdx := pcIndex - halfHeight
		endIdx := pcIndex + halfHeight + 1

		if startIdx < 0 {
			startIdx = 0
			endIdx = disasmHeight
		}
		if endIdx > len(allLines) {
			endIdx = len(allLines)
			startIdx = endIdx - disasmHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}

		return allLines[startIdx:endIdx]
	}

	if len(allLines) > disasmHeight {
		return allLines[:disasmHeight]
	}
	return allLines
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

func (t *Backend) generateTestPattern(patternType int) {
	switch patternType {
	case 0:
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
	case 1:
		for y := 0; y < video.FramebufferHeight; y++ {
			for x := 0; x < video.FramebufferWidth; x++ {
				gray := uint32(x * display.GrayscaleWhite / video.FramebufferWidth)
				color := video.GBColor((gray << display.RGBARShift) | (gray << display.RGBAGShift) | (gray << display.RGBABShift) | display.FullAlpha)
				t.testPatternFrame.SetPixel(uint(x), uint(y), color)
			}
		}
	case 2:
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
	case 3:
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

func (t *Backend) animateTestPattern() {
	frame := t.testFrameCount / display.TestPatternAnimationFrames
	switch t.testPatternType {
	case 2:
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
	case 3:
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
