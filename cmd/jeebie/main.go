package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli"
	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/video"
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
			Usage: "Backend to use for rendering (terminal, sdl2, web)",
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
	testPattern := c.Bool("test-pattern")

	var romPath string
	var emu *jeebie.Emulator
	var err error

	if !testPattern {
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
	callbacks := backend.BackendCallbacks{
		OnQuit: func() { running = false },
	}

	// only set up emulator callbacks if we have an emulator
	if emu != nil {
		callbacks.OnKeyPress = emu.HandleKeyPress
		callbacks.OnKeyRelease = emu.HandleKeyRelease
		callbacks.OnDebugMessage = func(message string) {
			handleDebugCommand(emu, message)
		}
	}

	config := backend.BackendConfig{
		Title:       "Jeebie Game Boy Emulator",
		Scale:       2,
		ShowDebug:   c.Bool("debug"),
		TestPattern: testPattern,
		Callbacks:   callbacks,
	}

	if err := emulatorBackend.Init(config); err != nil {
		return fmt.Errorf("failed to initialize backend: %v", err)
	}
	defer emulatorBackend.Cleanup()

	// provide access to emulator state for debugging displays
	if emu != nil {
		if terminalBackend, ok := emulatorBackend.(*backend.TerminalBackend); ok {
			terminalBackend.SetEmulatorState(
				func() backend.CPUState { return emu.GetCPU() },
				func() backend.MMUState { return emu.GetMMU() },
			)
		}
	}

	for running {
		var frame *video.FrameBuffer
		if emu != nil {
			emu.RunUntilFrame()
			frame = emu.GetCurrentFrame()
		}
		// frame can be nil for test pattern mode - backends handle this

		if err := emulatorBackend.Update(frame); err != nil {
			return fmt.Errorf("backend update failed: %v", err)
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

		snapshotConfig, err := backend.CreateSnapshotConfig(
			c.Int("snapshot-interval"),
			c.String("snapshot-dir"),
			romPath,
		)
		if err != nil {
			return nil, err
		}

		return backend.NewHeadlessBackend(frames, snapshotConfig), nil
	}

	backendName := c.String("backend")
	switch backendName {
	case "terminal":
		return backend.NewTerminalBackend(), nil
	case "headless":
		return nil, errors.New("use --headless flag instead of --backend=headless")
	default:
		return nil, fmt.Errorf("unsupported backend: %s (available: terminal)", backendName)
	}
}

// handleDebugCommand processes debug commands from backends
func handleDebugCommand(emu *jeebie.Emulator, command string) {
	switch command {
	case "debug:toggle_pause":
		if emu.GetDebuggerState() == 1 { // DebuggerPaused
			slog.Info("Debugger: Resuming execution")
			emu.DebuggerResume()
		} else {
			slog.Info("Debugger: Pausing execution")
			emu.DebuggerPause()
		}
	case "debug:step_instruction":
		slog.Info("Debugger: Step instruction")
		emu.DebuggerStepInstruction()
	case "debug:step_frame":
		slog.Info("Debugger: Step frame")
		emu.DebuggerStepFrame()
	case "debug:resume":
		slog.Info("Debugger: Resume")
		emu.DebuggerResume()
	case "debug:pause":
		slog.Info("Debugger: Pause")
		emu.DebuggerPause()
	default:
		slog.Debug("Unknown debug command", "command", command)
	}
}
