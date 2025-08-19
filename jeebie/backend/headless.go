package backend

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/valerio/go-jeebie/jeebie/debug"
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
		h.saveSnapshot(frame)
	}

	// Log progress periodically
	if h.frameCount%10 == 0 {
		slog.Info("Frame progress", "completed", h.frameCount, "total", h.maxFrames)
	}

	// Check if we've reached the target frame count
	if h.frameCount >= h.maxFrames {
		// Save final snapshot if enabled and we haven't just saved one
		if h.snapshotConfig.Enabled && h.frameCount%h.snapshotConfig.Interval != 0 {
			h.saveSnapshot(frame)
		}

		if h.snapshotConfig.Enabled {
			slog.Info("Headless execution completed", "frames", h.maxFrames, "png_snapshots_saved_to", h.snapshotConfig.Directory)
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

// saveSnapshot saves a PNG snapshot for the current frame
func (h *HeadlessBackend) saveSnapshot(frame *video.FrameBuffer) {
	pngBaseName := fmt.Sprintf("%s_frame_%d", h.snapshotConfig.ROMName, h.frameCount)

	if err := debug.SaveFramePNGToDir(frame, pngBaseName, h.snapshotConfig.Directory); err != nil {
		slog.Error("Failed to save PNG snapshot", "frame", h.frameCount, "error", err)
	}
}
