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
	"github.com/valerio/go-jeebie/jeebie/events"
	"github.com/valerio/go-jeebie/jeebie/memory"
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
		cli.BoolFlag{
			Name:  "event-driven",
			Usage: "Use event-driven emulation for cycle-accurate timing (experimental)",
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

		eventDriven := c.Bool("event-driven")

		slog.Info("Running headless mode", "frames", frames, "snapshot_interval", snapshotInterval, "snapshot_dir", snapshotDir, "event_driven", eventDriven)

		if eventDriven {
			// Use event-driven emulation
			return runEventDrivenHeadless(romPath, frames, snapshotInterval, snapshotDir, romName)
		} else {
			// Use traditional emulation
			emu, err := jeebie.NewWithFile(romPath)
			if err != nil {
				return err
			}

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

				if i%10 == 0 { // Log progress every 10 frames
					slog.Info("Frame progress", "completed", i+1, "total", frames)
				}
			}
		}

		if snapshotInterval > 0 {
			slog.Info("Headless execution completed", "frames", frames, "snapshots_saved_to", snapshotDir)
		} else {
			slog.Info("Headless execution completed", "frames", frames)
		}
		return nil
	} else {
		// Interactive mode (not event-driven yet)
		emu, err := jeebie.NewWithFile(romPath)
		if err != nil {
			return err
		}

		renderer, err := render.NewTerminalRenderer(emu)
		if err != nil {
			return err
		}
		return renderer.Run()
	}
}

// runEventDrivenHeadless runs the event-driven emulator in headless mode
func runEventDrivenHeadless(romPath string, frames, snapshotInterval int, snapshotDir, romName string) error {
	// Load ROM data
	data, err := os.ReadFile(romPath)
	if err != nil {
		return err
	}

	// Create memory management unit with ROM data
	cart := memory.NewCartridgeWithData(data)
	mmu := memory.NewWithCartridge(cart)

	// Create event-driven emulator
	emu := events.NewEventDrivenEmulator(mmu)

	slog.Info("Starting event-driven emulator", "rom", romPath)

	// Track snapshots saved
	snapshotsToSave := make(map[int]string)
	if snapshotInterval > 0 {
		for i := snapshotInterval; i <= frames; i += snapshotInterval {
			snapshotPath := filepath.Join(snapshotDir, fmt.Sprintf("%s_frame_%d.txt", romName, i))
			snapshotsToSave[i] = snapshotPath
		}
	}

	// Run emulation with periodic snapshot saves
	go func() {
		// Monitor frame progress and save snapshots
		lastFrameCount := uint64(0)

		for {
			currentFrameCount := emu.GetFrameCount()

			if currentFrameCount != lastFrameCount {
				// Frame completed
				frameNum := int(currentFrameCount)

				// Save snapshot if needed
				if snapshotPath, shouldSave := snapshotsToSave[frameNum]; shouldSave {
					if err := saveFrameSnapshotEventDriven(emu, snapshotPath); err != nil {
						slog.Error("Failed to save snapshot", "frame", frameNum, "path", snapshotPath, "error", err)
					} else {
						slog.Info("Saved frame snapshot", "frame", frameNum, "path", snapshotPath)
					}
				}

				// Log progress
				if frameNum%10 == 0 {
					slog.Info("Frame progress", "completed", frameNum, "total", frames)
				}

				lastFrameCount = currentFrameCount
			}

			// Check if emulation is complete
			if currentFrameCount >= uint64(frames) {
				emu.Stop()
				break
			}

			// Brief pause to avoid busy waiting
			// time.Sleep(time.Millisecond) // Uncomment if needed
		}
	}()

	// Run the event loop (this will block until completion)
	emu.RunEventLoop(frames)

	slog.Info("Event-driven emulation completed",
		"frames", emu.GetFrameCount(),
		"instructions", emu.GetInstructionCount(),
		"events", emu.GetEventCount())

	return nil
}

// saveFrameSnapshotEventDriven saves a frame snapshot from event-driven emulator using half-blocks
func saveFrameSnapshotEventDriven(emu *events.EventDrivenEmulator, filename string) error {
	fb := emu.GetCurrentFrame()
	frame := fb.ToSlice()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# Game Boy Frame Snapshot (Half-Block Rendering)\n")
	fmt.Fprintf(file, "# Frame: %d, Instructions: %d\n", emu.GetFrameCount(), emu.GetInstructionCount())
	fmt.Fprintf(file, "# Resolution: 160x144 pixels -> 160x72 text rows\n")
	fmt.Fprintf(file, "# Characters: ▀ ▄ █ (upper half, lower half, full block)\n")
	fmt.Fprintf(file, "#\n")

	// Use shared rendering utility to convert to half-blocks
	lines := render.RenderFrameToHalfBlocks(frame, 160, 144)

	// Write the rendered lines
	for _, line := range lines {
		fmt.Fprintf(file, "%s\n", line)
	}

	return nil
}

// saveFrameSnapshot saves the current frame as a text representation using half-blocks
func saveFrameSnapshot(emu *jeebie.Emulator, filename string) error {
	fb := emu.GetCurrentFrame()
	frame := fb.ToSlice()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header with metadata
	fmt.Fprintf(file, "# Game Boy Frame Snapshot (Half-Block Rendering)\n")
	fmt.Fprintf(file, "# Frame: %d, Instructions: %d\n", emu.GetFrameCount(), emu.GetInstructionCount())
	fmt.Fprintf(file, "# Resolution: 160x144 pixels -> 160x72 text rows\n")
	fmt.Fprintf(file, "# Characters: ▀ ▄ █ (upper half, lower half, full block)\n")
	fmt.Fprintf(file, "#\n")

	// Use shared rendering utility to convert to half-blocks
	lines := render.RenderFrameToHalfBlocks(frame, 160, 144)

	// Write the rendered lines
	for _, line := range lines {
		fmt.Fprintf(file, "%s\n", line)
	}

	return nil
}
