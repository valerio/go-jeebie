package jeebie

import (
	"testing"

	"github.com/valerio/go-jeebie/jeebie/backend"
	"github.com/valerio/go-jeebie/jeebie/backend/headless"
)

func BenchmarkEmulatorHeadless(b *testing.B) {
	testROMs := []struct {
		name   string
		path   string
		frames int
	}{
		{"dmg_acid_100", "../test-roms/dmg-acid2.gb", 100},
		{"dmg_acid_1000", "../test-roms/dmg-acid2.gb", 1000},
	}

	for _, tc := range testROMs {
		b.Run(tc.name, func(b *testing.B) {
			// Setup once outside the benchmark loop
			emu, err := NewWithFile(tc.path)
			if err != nil {
				b.Fatalf("Failed to create emulator: %v", err)
			}

			// Use large frame count to avoid quit condition allocations
			hBackend := headless.New(tc.frames*(b.N+1), headless.SnapshotConfig{})
			config := backend.BackendConfig{
				Title: "Benchmark",
			}
			if err := hBackend.Init(config); err != nil {
				b.Fatalf("Failed to initialize backend: %v", err)
			}
			defer hBackend.Cleanup()

			emu.SetFrameLimiter(nil)

			// Reset timer to exclude initialization
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for frameCount := 0; frameCount < tc.frames; frameCount++ {
					emu.RunUntilFrame()
					frame := emu.GetCurrentFrame()
					if _, err := hBackend.Update(frame); err != nil {
						b.Fatalf("Backend update failed: %v", err)
					}
				}
			}
		})
	}
}
