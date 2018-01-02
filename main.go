package main

import (
	"github.com/valep27/go-jeebie/jeebie"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()

	app.Name = "Jeebie"
	app.Description = "A simple gameboy emulator"
	app.Action = runEmulator

	app.Run(os.Args)
}

func runEmulator(c *cli.Context) error {
	path := ""

	if c.NArg() > 0 {
		path = c.Args().First()
	}

	if path != "" {
		emu, err := jeebie.NewWithFile(path)
		if err != nil {
			return err
		}

		emu.Run()
		return nil
	}

	// no file specified

	emu := jeebie.New()
	emu.Run()
	return nil
}
