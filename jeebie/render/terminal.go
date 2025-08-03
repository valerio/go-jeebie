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
	"github.com/valerio/go-jeebie/jeebie/memory"
)

const (
	width     = 160
	height    = 144
	scaleX    = 2
	scaleY    = 1
	frameTime = time.Second / 60
)

var shadeChars = []rune{'█', '▓', '▒', '░'}

type TerminalRenderer struct {
	screen   tcell.Screen
	emulator *jeebie.Emulator
	running  bool
}

func NewTerminalRenderer(emu *jeebie.Emulator) (*TerminalRenderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terminal: %v", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize terminal: %v", err)
	}

	return &TerminalRenderer{
		screen:   screen,
		emulator: emu,
		running:  true,
	}, nil
}

func (t *TerminalRenderer) Run() error {
	defer func() {
		slog.Info("Finishing terminal")
		t.screen.Fini()
	}()

	t.screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	t.screen.Clear()

	go t.handleInput()

	ticker := time.NewTicker(frameTime)
	defer ticker.Stop()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

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
				}
			}
		case *tcell.EventResize:
			t.screen.Sync()
		}
	}
}

func (t *TerminalRenderer) render() {
	fb := t.emulator.GetCurrentFrame()
	frame := fb.ToSlice()

	t.screen.Clear()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := frame[x*height+y]
			shade := 3 - (pixel>>24)/64
			if shade > 3 {
				shade = 3
			}
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			char := shadeChars[shade]
			screenX := x * scaleX
			screenY := y * scaleY
			for sx := 0; sx < scaleX; sx++ {
				t.screen.SetContent(screenX+sx, screenY, char, nil, style)
			}
		}
	}
}
