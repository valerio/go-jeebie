package backend

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/valerio/go-jeebie/jeebie/render"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// HeadlessBackend implements the Backend interface for automated testing and batch processing
type HeadlessBackend struct {
	config         BackendConfig
	callbacks      BackendCallbacks
	frameCount     int
	maxFrames      int
	snapshotConfig SnapshotConfig
}

// SnapshotConfig holds configuration for frame snapshots
type SnapshotConfig struct {
	Enabled   bool
	Interval  int    // Save snapshot every N frames
	Directory string // Directory to save snapshots
	ROMName   string // ROM name for snapshot filenames
}

func NewHeadlessBackend(maxFrames int, snapshotConfig SnapshotConfig) *HeadlessBackend {
	return &HeadlessBackend{
		maxFrames:      maxFrames,
		snapshotConfig: snapshotConfig,
	}
}

func (h *HeadlessBackend) Init(config BackendConfig) error {
	h.config = config
	h.callbacks = config.Callbacks

	if config.TestPattern {
		slog.Info("Headless test pattern mode - test patterns verified, exiting")
		// Signal completion immediately for test pattern mode
		if h.callbacks.OnQuit != nil {
			h.callbacks.OnQuit()
		}
		return nil
	}

	slog.Info("Running headless mode",
		"frames", h.maxFrames,
		"snapshot_interval", h.snapshotConfig.Interval,
		"snapshot_dir", h.snapshotConfig.Directory)

	// Set up debug logging for headless mode
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

// Update processes a frame and handles snapshots
func (h *HeadlessBackend) Update(frame *video.FrameBuffer) error {
	h.frameCount++

	// Save snapshot if needed
	if h.snapshotConfig.Enabled && h.frameCount%h.snapshotConfig.Interval == 0 {
		snapshotPath := filepath.Join(h.snapshotConfig.Directory,
			fmt.Sprintf("%s_frame_%d.txt", h.snapshotConfig.ROMName, h.frameCount))

		if err := h.saveFrameSnapshot(frame, snapshotPath); err != nil {
			slog.Error("Failed to save snapshot", "frame", h.frameCount, "path", snapshotPath, "error", err)
		} else {
			slog.Info("Saved frame snapshot", "frame", h.frameCount, "path", snapshotPath)
		}
	}

	// Log progress periodically
	if h.frameCount%10 == 0 {
		slog.Info("Frame progress", "completed", h.frameCount, "total", h.maxFrames)
	}

	// Check if we've reached the target frame count
	if h.frameCount >= h.maxFrames {
		if h.snapshotConfig.Enabled {
			slog.Info("Headless execution completed", "frames", h.maxFrames, "snapshots_saved_to", h.snapshotConfig.Directory)
		} else {
			slog.Info("Headless execution completed", "frames", h.maxFrames)
		}

		// Signal completion
		if h.callbacks.OnQuit != nil {
			h.callbacks.OnQuit()
		}
	}

	return nil
}

func (h *HeadlessBackend) Cleanup() error {
	return nil
}

// saveFrameSnapshot saves the current frame as a text representation using half-blocks
func (h *HeadlessBackend) saveFrameSnapshot(frame *video.FrameBuffer, filename string) error {
	frameData := frame.ToSlice()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write header with metadata
	fmt.Fprintf(file, "# Game Boy Frame Snapshot (Half-Block Rendering)\n")
	fmt.Fprintf(file, "# Frame: %d\n", h.frameCount)
	fmt.Fprintf(file, "# Resolution: 160x144 pixels -> 160x72 text rows\n")
	fmt.Fprintf(file, "# Characters: ▀ ▄ █ (upper half, lower half, full block)\n")
	fmt.Fprintf(file, "#\n")

	// Use shared rendering utility to convert to half-blocks
	lines := render.RenderFrameToHalfBlocks(frameData, 160, 144)

	for _, line := range lines {
		fmt.Fprintf(file, "%s\n", line)
	}

	return nil
}

// CreateSnapshotConfig creates a snapshot configuration from CLI parameters
func CreateSnapshotConfig(interval int, directory, romPath string) (SnapshotConfig, error) {
	config := SnapshotConfig{
		Enabled:  interval > 0,
		Interval: interval,
	}

	if !config.Enabled {
		return config, nil
	}

	// Set up snapshot directory
	if directory == "" {
		tempDir, err := os.MkdirTemp("", "jeebie-snapshots-*")
		if err != nil {
			return config, fmt.Errorf("failed to create snapshot directory: %v", err)
		}
		config.Directory = tempDir
	} else {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return config, fmt.Errorf("failed to create snapshot directory: %v", err)
		}
		config.Directory = directory
	}

	// Extract ROM name for snapshot filenames
	config.ROMName = filepath.Base(romPath)
	config.ROMName = strings.TrimSuffix(config.ROMName, filepath.Ext(config.ROMName))

	return config, nil
}
