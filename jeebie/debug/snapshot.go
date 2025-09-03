package debug

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/valerio/go-jeebie/jeebie/display"
	"github.com/valerio/go-jeebie/jeebie/video"
)

// TakeSnapshot handles F12 snapshot logic for backends
func TakeSnapshot(frame *video.FrameBuffer, isTestPattern bool, testPatternType int) {
	if frame == nil {
		slog.Warn("No frame data available for snapshot")
		return
	}

	var baseName string
	if isTestPattern {
		patternNames := []string{"checkerboard", "gradient", "stripes", "diagonal"}
		baseName = fmt.Sprintf("jeebie_snapshot_%s", patternNames[testPatternType])
	} else {
		baseName = "jeebie_snapshot"
	}

	if err := SaveFramePNGToDir(frame, baseName, ""); err != nil {
		slog.Error("Failed to save snapshot", "error", err)
	}
}

// SaveFramePNGToDir saves a framebuffer as PNG with timestamp to a specific directory
func SaveFramePNGToDir(frame *video.FrameBuffer, baseName, directory string) error {
	frameData := frame.ToSlice()

	// Convert framebuffer to RGBA
	pixels := make([]byte, video.FramebufferWidth*video.FramebufferHeight*display.RGBABytesPerPixel)
	for i, gbPixel := range frameData {
		idx := i * display.RGBABytesPerPixel
		r, g, b, a := gbPixelToRGBA(gbPixel)
		pixels[idx] = byte(r)
		pixels[idx+1] = byte(g)
		pixels[idx+2] = byte(b)
		pixels[idx+3] = byte(a)
	}

	img := image.NewRGBA(image.Rect(0, 0, video.FramebufferWidth, video.FramebufferHeight))
	copy(img.Pix, pixels)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.png", baseName, timestamp)

	// Determine output directory
	var outputDir string
	if directory != "" {
		outputDir = directory
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
		outputDir = cwd
	}

	filePath := filepath.Join(outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %v", err)
	}

	slog.Info("Snapshot saved", "path", filePath, "size", fmt.Sprintf("%dx%d", video.FramebufferWidth, video.FramebufferHeight), "format", "PNG")
	return nil
}

// SaveFrameGrayPNG saves a framebuffer as a grayscale PNG (used in integration tests)
func SaveFrameGrayPNG(frame *video.FrameBuffer, filepath string) error {
	img := image.NewGray(image.Rect(0, 0, video.FramebufferWidth, video.FramebufferHeight))

	frameData := frame.ToSlice()
	for y := range video.FramebufferHeight {
		for x := range video.FramebufferWidth {
			pixel := frameData[y*video.FramebufferWidth+x]

			var gray uint8
			switch pixel {
			case uint32(video.BlackColor):
				gray = 0
			case uint32(video.DarkGreyColor):
				gray = 85
			case uint32(video.LightGreyColor):
				gray = 170
			case uint32(video.WhiteColor):
				gray = 255
			default:
				gray = 0
			}

			img.SetGray(x, y, color.Gray{gray})
		}
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// gbPixelToRGBA converts Game Boy pixel to RGBA values
func gbPixelToRGBA(gbPixel uint32) (r, g, b, a uint32) {
	switch gbPixel {
	case uint32(video.WhiteColor):
		return display.GrayscaleWhite, display.GrayscaleWhite, display.GrayscaleWhite, display.FullAlpha
	case uint32(video.LightGreyColor):
		return display.GrayscaleLightGray, display.GrayscaleLightGray, display.GrayscaleLightGray, display.FullAlpha
	case uint32(video.DarkGreyColor):
		return display.GrayscaleDarkGray, display.GrayscaleDarkGray, display.GrayscaleDarkGray, display.FullAlpha
	case uint32(video.BlackColor):
		return display.GrayscaleBlack, display.GrayscaleBlack, display.GrayscaleBlack, display.FullAlpha
	default:
		// For any non-standard colors, extract the red channel and convert to grayscale
		red := uint32((gbPixel >> display.RGBARShift) & display.RGBAColorMask)
		return red, red, red, display.FullAlpha
	}
}
