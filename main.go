package main

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten"
	"github.com/urfave/cli"
	"github.com/valerio/go-jeebie/jeebie"
)

const (
	renderScale = 3
	width       = 160
	height      = 144
)

// Game implements ebiten.Game interface.
type Game struct {
	picture  []uint8
	emulator jeebie.Emulator
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return fmt.Errorf("quit")
	}

	g.emulator.RunUntilFrame()
	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Clear()
	fb := g.emulator.GetCurrentFrame()
	frame := fb.ToSlice()

	// Write your game's rendering.
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			pixel := frame[i*height+j]
			c := color.RGBA{
				uint8(pixel >> 24),
				uint8(pixel >> 16),
				uint8(pixel >> 8),
				255,
			}
			screen.Set(i, j, c)
		}
	}
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return width, height
}

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

	game := &Game{emulator: *jeebie.New()}

	ebiten.SetWindowSize(width*renderScale, height*renderScale)
	ebiten.SetWindowTitle(c.App.Name)
	ebiten.SetWindowResizable(true)

	if path != "" {
		emu, err := jeebie.NewWithFile(path)
		if err != nil {
			return err
		}

		game.emulator = *emu
	}

	return ebiten.RunGame(game)
}
