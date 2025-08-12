package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/urfave/cli"
	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/render"
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
	}
	app.Action = runEmulator

	err := app.Run(os.Args)
	if err != nil {
		slog.Error("Error running emulator", "error", err)
		os.Exit(1)
	}
}

func runEmulator(c *cli.Context) error {
	// Test pattern mode - no ROM needed
	if c.Bool("test-pattern") {
		slog.Info("Running in test pattern mode")
		return render.RunTestPattern()
	}

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

	if c.Bool("headless") {
		frames := c.Int("frames")
		if frames <= 0 {
			return errors.New("headless mode requires --frames option with a positive value")
		}
		
		// Set up debug logging for headless mode
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger := slog.New(handler)
		slog.SetDefault(logger)
		
		slog.Info("Running headless mode", "frames", frames)
		for i := 0; i < frames; i++ {
			emu.RunUntilFrame()
			if i%10 == 0 {  // Log progress every 10 frames
				slog.Info("Frame progress", "completed", i+1, "total", frames)
			}
		}
		slog.Info("Headless execution completed", "frames", frames)
		return nil
	} else {
		renderer, err := render.NewTerminalRenderer(emu)
		if err != nil {
			return err
		}
		return renderer.Run()
	}
}
