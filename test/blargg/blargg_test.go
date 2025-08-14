package blargg

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/valerio/go-jeebie/jeebie"
	"github.com/valerio/go-jeebie/jeebie/video"
)

const (
	BlackPixel     = 0x000000FF
	DarkGrayPixel  = 0x4C4C4CFF
	LightGrayPixel = 0x989898FF
	WhitePixel     = 0xFFFFFFFF
)

type BlarggTestCase struct {
	ROMPath      string
	ExpectedHash string
	MaxFrames    uint64
	MinLoopCount int
	GoldenFile   string
	Name         string
}

func GetBlarggTests() []BlarggTestCase {
	baseDir := "../../test-roms"

	return []BlarggTestCase{
		{
			ROMPath:      filepath.Join(baseDir, "01-special.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "01-special",
		},
		{
			ROMPath:      filepath.Join(baseDir, "02-interrupts.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "02-interrupts",
		},
		{
			ROMPath:      filepath.Join(baseDir, "03-op sp,hl.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "03-op sp,hl",
		},
		{
			ROMPath:      filepath.Join(baseDir, "04-op r,imm.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "04-op r,imm",
		},
		{
			ROMPath:      filepath.Join(baseDir, "05-op rp.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "05-op rp",
		},
		{
			ROMPath:      filepath.Join(baseDir, "06-ld r,r.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "06-ld r,r",
		},
		{
			ROMPath:      filepath.Join(baseDir, "07-jr,jp,call,ret,rst.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "07-jr,jp,call,ret,rst",
		},
		{
			ROMPath:      filepath.Join(baseDir, "08-misc instrs.gb"),
			MaxFrames:    500,
			MinLoopCount: 50,
			Name:         "08-misc instrs",
		},
		{
			ROMPath:      filepath.Join(baseDir, "09-op r,r.gb"),
			MaxFrames:    1000,
			MinLoopCount: 50,
			Name:         "09-op r,r",
		},
		{
			ROMPath:      filepath.Join(baseDir, "10-bit ops.gb"),
			MaxFrames:    1000,
			MinLoopCount: 50,
			Name:         "10-bit ops",
		},
		{
			ROMPath:      filepath.Join(baseDir, "11-op a,(hl).gb"),
			MaxFrames:    1500,
			MinLoopCount: 50,
			Name:         "11-op a,(hl)",
		},
	}
}

func runBlarggTest(t *testing.T, testCase BlarggTestCase) {
	if _, err := os.Stat(testCase.ROMPath); os.IsNotExist(err) {
		t.Skipf("ROM file not found: %s", testCase.ROMPath)
		return
	}

	t.Logf("Running Blargg test: %s (%s)", testCase.Name, testCase.ROMPath)
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

		if err := savePNG(fb, snapshotPath); err != nil {
			t.Fatalf("Failed to write snapshot PNG file: %v", err)
		}

		t.Logf("Reference files generated - hash: %s", hash)
		return
	}

	if _, err := os.Stat(screenDataPath); os.IsNotExist(err) {
		t.Fatalf("Screen data file not found: %s. Run 'make test-blargg-golden' to generate reference files first.", screenDataPath)
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
		savePNG(fb, actualPngPath)

		t.Errorf("Test output differs from expected\n  Expected hash: %s\n  Actual hash:   %s\n  Files saved:   %s, %s",
			expectedHash, hash, actualBinPath, actualPngPath)
	} else {
		t.Logf("Test passed - hash: %s", hash)
	}
}

func savePNG(fb *video.FrameBuffer, filename string) error {
	img := image.NewGray(image.Rect(0, 0, 160, 144))

	frameData := fb.ToSlice()
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			pixel := frameData[y*160+x]

			var gray uint8
			switch pixel {
			case BlackPixel:
				gray = 0
			case DarkGrayPixel:
				gray = 85
			case LightGrayPixel:
				gray = 170
			case WhitePixel:
				gray = 255
			default:
				gray = 0
			}

			img.SetGray(x, y, color.Gray{gray})
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func TestBlarggSuite(t *testing.T) {
	tests := GetBlarggTests()
	
	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			runBlarggTest(t, testCase)
		})
	}
}
