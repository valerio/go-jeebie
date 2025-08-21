//go:build sdl2
// +build sdl2

package jeebie

import (
	"testing"

	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/backend/sdl2"
	"github.com/valerio/go-jeebie/jeebie/input/action"
)

func BenchmarkSDL2Backend(b *testing.B) {
	testCases := []struct {
		name   string
		path   string
		frames int
	}{
		{"dmg_acid_100", "../test-roms/dmg-acid2.gb", 100},
		{"dmg_acid_1000", "../test-roms/dmg-acid2.gb", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Setup: Create emulator and SDL2 backend once
			emu, err := NewWithFile(tc.path)
			if err != nil {
				b.Fatalf("Failed to create emulator: %v", err)
			}

			sdlBackend := sdl2.New()
			config := backend.BackendConfig{
				Title: "Benchmark",
				Scale: 1, // Minimal scale for benchmarking
			}
			if err := sdlBackend.Init(config); err != nil {
				b.Fatalf("Failed to initialize SDL2 backend: %v", err)
			}
			defer sdlBackend.Cleanup()

			emu.SetFrameLimiter(nil) // No frame limiting for benchmarks

			// Reset timer to exclude initialization
			b.ResetTimer()
			b.ReportAllocs()

			// Benchmark loop with SDL2 rendering
			for i := 0; i < b.N; i++ {
				for frameCount := 0; frameCount < tc.frames; frameCount++ {
					emu.RunUntilFrame()
					frame := emu.GetCurrentFrame()

					// Update SDL2 backend (includes rendering)
					events, err := sdlBackend.Update(frame)
					if err != nil {
						b.Fatalf("SDL2 update failed: %v", err)
					}

					// Check for quit events
					for _, evt := range events {
						if evt.Action == action.EmulatorQuit {
							b.Fatalf("Unexpected quit event during benchmark")
						}
					}
				}
			}
		})
	}
}
