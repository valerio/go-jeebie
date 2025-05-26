package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/urfave/cli"
	"github.com/valerio/go-jeebie/jeebie"
)

const (
	// Game Boy screen dimensions
	width  = 160
	height = 144

	// Since terminal characters are taller than wide, we'll scale the width more
	// to maintain approximate aspect ratio
	scaleX = 2 // Each pixel becomes 2 characters wide
	scaleY = 1 // Each pixel becomes 1 character tall

	// Frame timing (Game Boy runs at ~59.7 FPS)
	frameTime = time.Second / 60
)

// Characters to represent different shades of gray
// From darkest to lightest.
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

	// Set up screen
	t.screen.SetStyle(tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite))
	t.screen.Clear()

	// Handle input in a separate goroutine
	go t.handleInput()

	// Main render loop
	ticker := time.NewTicker(frameTime)
	defer ticker.Stop()

	// catch SIGINT and SIGTERM
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
			case tcell.KeyEscape:
				t.running = false
				return
			}
		case *tcell.EventResize:
			t.screen.Sync()
		}
	}
}

func (t *TerminalRenderer) render() {
	fb := t.emulator.GetCurrentFrame()
	frame := fb.ToSlice()

	// Clear screen with background color
	t.screen.Clear()

	// Render each pixel
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get pixel value (assuming it's a 32-bit color where higher values = lighter)
			pixel := frame[x*height+y]
			// Convert to shade index (4 shades, so divide by 64 to get 0-3)
			shade := 3 - (pixel>>24)/64 // Invert so higher values = darker
			if shade > 3 {
				shade = 3
			}

			// Draw scaled pixel
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			char := shadeChars[shade]

			// Draw the character repeated scaleX times
			screenX := x * scaleX
			screenY := y * scaleY
			for sx := 0; sx < scaleX; sx++ {
				t.screen.SetContent(screenX+sx, screenY, char, nil, style)
			}
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "Jeebie"
	app.Description = "A simple gameboy emulator"
	app.Usage = "jeebie [options] <ROM file>"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "rom",
			Usage: "Path to the ROM file",
		},
	}
	app.Action = runEmulator

	err := app.Run(os.Args)
	if err != nil {
		slog.Error("Error running emulator", "error", err)
		os.Exit(1)
	}
}

func runEmulator(c *cli.Context) error {
	romPath := c.String("rom")
	if romPath == "" {
		if c.NArg() > 0 {
			romPath = c.Args().Get(0)
		} else {
			cli.ShowAppHelp(c)
			return errors.New("no ROM path provided")
		}
	}

	emu, err := jeebie.NewWithFile(romPath)
	if err != nil {
		return err
	}

	renderer, err := NewTerminalRenderer(emu)
	if err != nil {
		return err
	}

	return renderer.Run()
}
