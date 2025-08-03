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

	if c.Bool("headless") {
		for {
			emu.RunUntilFrame()
		}
	} else {
		renderer, err := render.NewTerminalRenderer(emu)
		if err != nil {
			return err
		}
		return renderer.Run()
	}
}
