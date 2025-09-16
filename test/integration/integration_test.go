package integration

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/debug"
)

type IntegrationTestCase struct {
	ROMPath      string
	ExpectedHash string
	MaxFrames    uint64
	MinLoopCount int
	GoldenFile   string
	Name         string
}

func GetIntegrationTests() []IntegrationTestCase {
	baseDir := "../../test-roms/game-boy-test-roms/blargg/cpu_instrs/individual"

	tests := []IntegrationTestCase{
		{
			ROMPath:      filepath.Join(baseDir, "01-special.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "01-special",
		},
		{
			ROMPath:      filepath.Join(baseDir, "02-interrupts.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "02-interrupts",
		},
		{
			ROMPath:      filepath.Join(baseDir, "03-op sp,hl.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "03-op sp,hl",
		},
		{
			ROMPath:      filepath.Join(baseDir, "04-op r,imm.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "04-op r,imm",
		},
		{
			ROMPath:      filepath.Join(baseDir, "05-op rp.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "05-op rp",
		},
		{
			ROMPath:      filepath.Join(baseDir, "06-ld r,r.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "06-ld r,r",
		},
		{
			ROMPath:      filepath.Join(baseDir, "07-jr,jp,call,ret,rst.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "07-jr,jp,call,ret,rst",
		},
		{
			ROMPath:      filepath.Join(baseDir, "08-misc instrs.gb"),
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "08-misc instrs",
		},
		{
			ROMPath:      filepath.Join(baseDir, "09-op r,r.gb"),
			MaxFrames:    1000,
			MinLoopCount: 10,
			Name:         "09-op r,r",
		},
		{
			ROMPath:      filepath.Join(baseDir, "10-bit ops.gb"),
			MaxFrames:    1000,
			MinLoopCount: 10,
			Name:         "10-bit ops",
		},
		{
			ROMPath:      filepath.Join(baseDir, "11-op a,(hl).gb"),
			MaxFrames:    1500,
			MinLoopCount: 10,
			Name:         "11-op a,(hl)",
		},
		{
			ROMPath:   "../../test-roms/game-boy-test-roms/dmg-acid2/dmg-acid2.gb",
			MaxFrames: 10, // run for fixed frames
			Name:      "dmg-acid2",
		},
		{
			ROMPath:      "../../test-roms/game-boy-test-roms/blargg/halt_bug.gb",
			MaxFrames:    500,
			MinLoopCount: 10,
			Name:         "halt_bug",
		},
		{
			ROMPath:      "../../test-roms/game-boy-test-roms/blargg/instr_timing/instr_timing.gb",
			MaxFrames:    1200,
			MinLoopCount: 10,
			Name:         "instr_timing",
		},
		{
			ROMPath:   "../../test-roms/game-boy-test-roms/blargg/mem_timing/individual/01-read_timing.gb",
			MaxFrames: 60,
			Name:      "mem_timing_01-read",
		},
		{
			ROMPath:   "../../test-roms/game-boy-test-roms/blargg/mem_timing/individual/02-write_timing.gb",
			MaxFrames: 60,
			Name:      "mem_timing_02-write",
		},
		{
			ROMPath:   "../../test-roms/game-boy-test-roms/blargg/mem_timing/individual/03-modify_timing.gb",
			MaxFrames: 60,
			Name:      "mem_timing_03-modify",
		},
		{
			ROMPath:   "../../external/gb-test-roms/dmg_sound/rom_singles/01-registers.gb",
			MaxFrames: 60,
			Name:      "dmg_sound_01-registers",
		},
	}

	return tests
}

func runIntegrationTest(t *testing.T, testCase IntegrationTestCase) {
	if _, err := os.Stat(testCase.ROMPath); os.IsNotExist(err) {
		t.Fatalf("Test ROM not found: %s\n\nPlease download the test ROMs first by running:\n    make test-roms-download\n\nOr run the full test suite with:\n    make test-all", testCase.ROMPath)
		return
	}

	t.Logf("Running integration test: %s (%s)", testCase.Name, testCase.ROMPath)
	emu, err := jeebie.NewWithFile(testCase.ROMPath)
	if err != nil {
		t.Fatalf("Failed to create emulator: %v", err)
	}

	emu.ConfigureCompletionDetection(testCase.MaxFrames, testCase.MinLoopCount)

	emu.RunUntilComplete()

	fb := emu.GetCurrentFrame()

	testName := testCase.Name

	screenDataPath := filepath.Join("testdata", fmt.Sprintf("%s.bin", testName))
	snapshotPath := filepath.Join("testdata", "snapshots", fmt.Sprintf("%s.png", testName))

	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join("testdata", "snapshots"), 0755); err != nil {
		t.Fatalf("Failed to create snapshots directory: %v", err)
	}

	binaryData := fb.ToGrayscale()
	hash := fmt.Sprintf("%x", md5.Sum(binaryData))

	generateReference := os.Getenv("BLARGG_GENERATE_GOLDEN") == "true"

	if generateReference {
		t.Logf("Generating reference files for %s", testCase.Name)
		if err := os.WriteFile(screenDataPath, binaryData, 0644); err != nil {
			t.Fatalf("Failed to write screen data file: %v", err)
		}

		if err := debug.SaveFrameGrayPNG(fb, snapshotPath); err != nil {
			t.Fatalf("Failed to write snapshot PNG file: %v", err)
		}

		t.Logf("Reference files generated - hash: %s", hash)
		return
	}

	if _, err := os.Stat(screenDataPath); os.IsNotExist(err) {
		t.Fatalf("Screen data file not found: %s. Run 'make test-integration-golden' to generate reference files first.", screenDataPath)
	}

	expectedData, err := os.ReadFile(screenDataPath)
	if err != nil {
		t.Fatalf("Failed to read screen data file: %v", err)
	}

	expectedHash := fmt.Sprintf("%x", md5.Sum(expectedData))

	if hash != expectedHash {
		actualBinPath := filepath.Join("testdata", fmt.Sprintf("%s_actual.bin", testName))
		actualPngPath := filepath.Join("testdata", "snapshots", fmt.Sprintf("%s_actual.png", testName))

		os.WriteFile(actualBinPath, binaryData, 0644)
		debug.SaveFrameGrayPNG(fb, actualPngPath)

		t.Errorf("Test output differs from expected\n  Expected hash: %s\n  Actual hash:   %s\n  Files saved:   %s, %s",
			expectedHash, hash, actualBinPath, actualPngPath)
	} else {
		t.Logf("Test passed - hash: %s", hash)
	}
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Check if test ROMs are available
	testRomsPath := "../../test-roms/game-boy-test-roms"
	if _, err := os.Stat(testRomsPath); os.IsNotExist(err) {
		t.Fatalf("Test ROMs not found at %s\n\n"+
			"Please download the test ROMs first by running:\n"+
			"    make test-roms-download\n\n"+
			"Or run the full test suite (which downloads automatically):\n"+
			"    make test-all\n", testRomsPath)
	}

	tests := GetIntegrationTests()

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			runIntegrationTest(t, testCase)
		})
	}
}
