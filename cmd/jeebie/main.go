package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
		cli.IntFlag{
			Name:  "snapshot-interval",
			Usage: "Save frame snapshots every N frames in headless mode (0 = disabled)",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "snapshot-dir",
			Usage: "Directory to save frame snapshots (default: temp directory)",
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
		
		snapshotInterval := c.Int("snapshot-interval")
		snapshotDir := c.String("snapshot-dir")
		
		// Set up snapshot directory
		if snapshotInterval > 0 {
			if snapshotDir == "" {
				// Create temp directory
				tempDir, err := os.MkdirTemp("", "jeebie-snapshots-*")
				if err != nil {
					return fmt.Errorf("failed to create snapshot directory: %v", err)
				}
				snapshotDir = tempDir
			} else {
				// Create specified directory if it doesn't exist
				if err := os.MkdirAll(snapshotDir, 0755); err != nil {
					return fmt.Errorf("failed to create snapshot directory: %v", err)
				}
			}
		}
		
		// Set up debug logging for headless mode
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger := slog.New(handler)
		slog.SetDefault(logger)
		
		// Extract ROM name for snapshot filenames
		romName := filepath.Base(romPath)
		romName = strings.TrimSuffix(romName, filepath.Ext(romName))
		
		slog.Info("Running headless mode", "frames", frames, "snapshot_interval", snapshotInterval, "snapshot_dir", snapshotDir)
		
		for i := 0; i < frames; i++ {
			emu.RunUntilFrame()
			
			// Save snapshot if needed
			if snapshotInterval > 0 && (i+1)%snapshotInterval == 0 {
				snapshotPath := filepath.Join(snapshotDir, fmt.Sprintf("%s_frame_%d.txt", romName, i+1))
				if err := saveFrameSnapshot(emu, snapshotPath); err != nil {
					slog.Error("Failed to save snapshot", "frame", i+1, "path", snapshotPath, "error", err)
				} else {
					slog.Info("Saved frame snapshot", "frame", i+1, "path", snapshotPath)
				}
			}
			
			if i%10 == 0 {  // Log progress every 10 frames
				slog.Info("Frame progress", "completed", i+1, "total", frames)
			}
		}
		
		if snapshotInterval > 0 {
			slog.Info("Headless execution completed", "frames", frames, "snapshots_saved_to", snapshotDir)
		} else {
			slog.Info("Headless execution completed", "frames", frames)
		}
		return nil
	} else {
		renderer, err := render.NewTerminalRenderer(emu)
		if err != nil {
			return err
		}
		return renderer.Run()
	}
}

// saveFrameSnapshot saves the current frame as a text representation
func saveFrameSnapshot(emu *jeebie.Emulator, filename string) error {
	fb := emu.GetCurrentFrame()
	frame := fb.ToSlice()
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write header with metadata
	fmt.Fprintf(file, "# Game Boy Frame Snapshot\n")
	fmt.Fprintf(file, "# Frame: %d, Instructions: %d\n", emu.GetFrameCount(), emu.GetInstructionCount())
	fmt.Fprintf(file, "# Resolution: 160x144 pixels\n")
	fmt.Fprintf(file, "# Legend: █=black ▓=dark ▒=light ░=white\n")
	fmt.Fprintf(file, "#\n")
	
	// Character mapping for different shades
	shadeChars := []rune{'█', '▓', '▒', '░'}
	
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			pixel := frame[y*160+x]
			
			shade := 0
			switch pixel {
			case 0x000000FF: // Black
				shade = 0
			case 0x4C4C4CFF: // Dark gray  
				shade = 1
			case 0x989898FF: // Light gray
				shade = 2
			case 0xFFFFFFFF: // White
				shade = 3
			default:
				shade = 3 // Default to white for unknown colors
			}
			
			fmt.Fprintf(file, "%c", shadeChars[shade])
		}
		fmt.Fprintf(file, "\n")
	}
	
	return nil
}
