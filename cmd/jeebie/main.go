package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli"
	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/backend/headless"
	"github.com/valerio/go-jeebie/jeebie/backend/sdl2"
	"github.com/valerio/go-jeebie/jeebie/backend/terminal"
	"github.com/valerio/go-jeebie/jeebie/input"
	"github.com/valerio/go-jeebie/jeebie/input/action"
	"github.com/valerio/go-jeebie/jeebie/input/event"
)

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
		cli.BoolFlag{
			Name:  "headless",
			Usage: "Run the emulator without a graphical interface",
		},
		cli.IntFlag{
			Name:  "frames",
			Usage: "Number of frames to run in headless mode (required for headless)",
			Value: 0,
		},
		cli.BoolFlag{
			Name:  "test-pattern",
			Usage: "Display a test pattern instead of emulation (for debugging display)",
		},
		cli.IntFlag{
			Name:  "snapshot-interval",
			Usage: "Save frame snapshots every N frames in headless mode (0 = disabled)",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "snapshot-dir",
			Usage: "Directory to save frame snapshots (default: temp directory)",
		},
		cli.StringFlag{
			Name:  "backend",
			Usage: "Backend to use for rendering (terminal, sdl2)",
			Value: "terminal",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug information display",
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
	// Set log level based on debug flag
	if c.Bool("debug") {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	testPattern := c.Bool("test-pattern")

	var romPath string
	var emu jeebie.Emulator
	var err error

	if testPattern {
		emu = jeebie.NewTestPatternEmulator()
	} else {
		romPath = c.String("rom")
		if romPath == "" {
			if c.NArg() > 0 {
				romPath = c.Args().Get(0)
			} else {
				cli.ShowAppHelp(c)
				return errors.New("no ROM path provided")
			}
		}

		emu, err = jeebie.NewWithFile(romPath)
		if err != nil {
			return err
		}
	}

	emulatorBackend, err := createBackend(c, romPath)
	if err != nil {
		return err
	}

	running := true

	config := backend.BackendConfig{
		Title:         "Jeebie",
		Scale:         2,
		ShowDebug:     c.Bool("debug"),
		TestPattern:   testPattern,
		DebugProvider: emu,
	}

	if err := emulatorBackend.Init(config); err != nil {
		return fmt.Errorf("failed to initialize backend: %v", err)
	}
	defer emulatorBackend.Cleanup()

	// provide access to emulator state for debugging displays
	if terminalBackend, ok := emulatorBackend.(*terminal.Backend); ok {
		terminalBackend.SetDebugProvider(emu)
	}

	// Create input handler for debouncing
	inputHandler := input.NewHandler()

	for running {
		emu.RunUntilFrame()
		frame := emu.GetCurrentFrame()

		events, err := emulatorBackend.Update(frame)
		if err != nil {
			return fmt.Errorf("backend update failed: %v", err)
		}

		// Process events from backend with debouncing
		for _, evt := range events {
			if inputHandler.ProcessEvent(evt) {
				handleEvent(emu, emulatorBackend, evt, &running)
			}
		}
	}

	return nil
}

func createBackend(c *cli.Context, romPath string) (backend.Backend, error) {
	if c.Bool("headless") {
		frames := c.Int("frames")
		// Test pattern mode doesn't need frames since it exits immediately
		if frames <= 0 && !c.Bool("test-pattern") {
			return nil, errors.New("headless mode requires --frames option with a positive value")
		}

		snapshotConfig, err := headless.CreateSnapshotConfig(
			c.Int("snapshot-interval"),
			c.String("snapshot-dir"),
			romPath,
		)
		if err != nil {
			return nil, err
		}

		return headless.New(frames, snapshotConfig), nil
	}

	backendName := c.String("backend")
	switch backendName {
	case "terminal":
		return terminal.New(), nil
	case "sdl2":
		return sdl2.New(), nil
	case "headless":
		return nil, errors.New("use --headless flag instead of --backend=headless")
	default:
		return nil, fmt.Errorf("unsupported backend: %s (available: terminal, sdl2)", backendName)
	}
}

func handleEvent(emu jeebie.Emulator, b backend.Backend, evt backend.InputEvent, running *bool) {
	switch evt.Action {
	case action.EmulatorQuit:
		if evt.Type == event.Press {
			*running = false
		}
	case action.EmulatorPauseToggle:
		if evt.Type == event.Press {
			// HandleAction will toggle pause state internally
			emu.HandleAction(action.EmulatorPauseToggle, true)
		}
	// Backend-specific actions that need special handling
	case action.EmulatorSnapshot, action.EmulatorTestPatternCycle,
		action.EmulatorDebugToggle, action.EmulatorDebugUpdate:
		if evt.Type == event.Press {
			// Let SDL2 backend handle its specific actions
			if sdlBackend, ok := b.(*sdl2.Backend); ok {
				sdlBackend.HandleBackendAction(evt.Action)
			}
		}
	default:
		emu.HandleAction(evt.Action, evt.Type == event.Press)
	}
}
