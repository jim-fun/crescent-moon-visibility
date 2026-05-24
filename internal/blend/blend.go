// Package blend provides the final compositing step (previously implemented in
// the now-legacy gpu_blend.py).
//
// It takes raw overlay data produced by the CPU or GPU renderers, composites
// them onto the NASA base map using a 60% alpha blend, draws the Yallop legend,
// and writes high-quality WEBP output.
//
// The core pipeline is now free of Python dependencies.
package blend

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	drawstd "image/draw"
	xdraw "golang.org/x/image/draw"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Target output resolution (4K)
const (
	TargetWidth  = 3840
	TargetHeight = 2160
)

// BlendOptions controls the blending behavior.
type BlendOptions struct {
	// OutputDir is where .webp files will be written.
	OutputDir string

	// BaseMapPath is the path to the NASA base map (usually data/map_nasa.png).
	BaseMapPath string

	// Alpha is the blend strength for the overlay (0.0 = base only, 1.0 = full overlay).
	// The classic value used by the project is 0.6.
	Alpha float64
}

// DefaultOptions returns sensible defaults matching historical behavior.
func DefaultOptions() BlendOptions {
	return BlendOptions{
		OutputDir:   ".",
		BaseMapPath: "data/map_nasa.png",
		Alpha:       0.6,
	}
}

// ProcessFiles is the main entry point that replaces the previous call to
// `python3 gpu_blend.py ...`.
//
// Each entry in files should be in the form "path|moon_age" (moon_age is
// currently ignored, matching the previous dead-code behavior).
func ProcessFiles(files []string, opts BlendOptions) error {
	if opts.BaseMapPath == "" {
		opts.BaseMapPath = DefaultOptions().BaseMapPath
	}
	if opts.Alpha == 0 {
		opts.Alpha = DefaultOptions().Alpha
	}

	baseImg, err := loadBaseMap(opts.BaseMapPath)
	if err != nil {
		return fmt.Errorf("failed to load base map: %w", err)
	}

	// Precompute once
	targetSize := image.Rect(0, 0, TargetWidth, TargetHeight)
	baseResized := resizeToTarget(baseImg, targetSize)

	for _, entry := range files {
		parts := strings.Split(entry, "|")
		overlayPath := parts[0]

		if err := processOne(overlayPath, baseResized, opts); err != nil {
			fmt.Printf("✗ blend failed for %s: %v\n", overlayPath, err)
			// Continue with other files (matching old Python behavior)
		}
	}

	return nil
}

func processOne(overlayPath string, baseImg image.Image, opts BlendOptions) error {
	overlay, cleanup, err := loadOverlay(overlayPath)
	if err != nil {
		return err
	}
	defer cleanup()

	// Early skip logic (identical to Python): count A-E pixels
	if !hasSufficientVisibility(overlay) {
		fmt.Printf("[skip] %s: no visibility zones — overlay too young to render\n", filepath.Base(overlayPath))
		cleanupTempFiles(overlayPath)
		return nil
	}

	// Resize overlay to target using high-quality CatmullRom
	overlayResized := resizeToTarget(overlay, baseImg.Bounds())

	// Perform 60% alpha blend
	blended := blendImages(baseImg, overlayResized, opts.Alpha)

	// Draw legend
	finalImg := drawLegend(blended, filepath.Base(strings.TrimSuffix(overlayPath, filepath.Ext(overlayPath))))

	// Write WEBP (quality 98 equivalent)
	outPath := strings.TrimSuffix(overlayPath, filepath.Ext(overlayPath)) + ".webp"
	if opts.OutputDir != "" && opts.OutputDir != "." {
		outPath = filepath.Join(opts.OutputDir, filepath.Base(outPath))
	}

	if err := encodeWEBP(finalImg, outPath, 98); err != nil {
		return fmt.Errorf("failed to write WEBP: %w", err)
	}

	fmt.Printf("✓ blended %s -> %s\n", filepath.Base(overlayPath), filepath.Base(outPath))

	cleanupTempFiles(overlayPath)
	return nil
}

// --- Image loading & helpers ---

func loadBaseMap(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

// loadOverlay supports both raw .bin (from CPU renderer) and PNG (from GPU renderer).
func loadOverlay(path string) (img image.Image, cleanup func(), err error) {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	binPath := base + ".bin"

	if _, err := os.Stat(binPath); err == nil {
		data, err := os.ReadFile(binPath)
		if err != nil {
			return nil, nil, err
		}
		// Try common resolutions used by the project
		for _, dims := range [][2]int{{3600, 2160}, {3840, 2160}, {1440, 720}} {
			w, h := dims[0], dims[1]
			if len(data) == w*h*4 {
				rgba := &image.RGBA{
					Pix:    data,
					Stride: w * 4,
					Rect:   image.Rect(0, 0, w, h),
				}
				return rgba, func() { os.Remove(binPath) }, nil
			}
		}
		return nil, nil, fmt.Errorf("unexpected .bin size: %d bytes", len(data))
	}

	// Fallback to PNG
	f, err := os.Open(path)
	if err != nil {
		// Try without extension
		if f2, err2 := os.Open(base); err2 == nil {
			f = f2
		} else {
			return nil, nil, err
		}
	}
	defer f.Close()

	img, _, err = image.Decode(f)
	if err != nil {
		return nil, nil, err
	}
	return img, func() {}, nil
}

func hasSufficientVisibility(img image.Image) bool {
	// These RGB tuples match the ones used in gpu_blend.py (A-E colors)
	targets := [][3]uint8{
		{0, 204, 204},   // A
		{0, 179, 179},   // B
		{255, 255, 26},  // C
		{230, 230, 0},   // D
		{179, 179, 0},   // E
	}

	bounds := img.Bounds()
	count := 0

	// Fast path using RGBA if possible
	if rgba, ok := img.(*image.RGBA); ok {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				i := (y-bounds.Min.Y)*rgba.Stride + (x-bounds.Min.X)*4
				r, g, b, a := rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2], rgba.Pix[i+3]
				if a < 250 {
					continue
				}
				for _, t := range targets {
					if r == t[0] && g == t[1] && b == t[2] {
						count++
						if count >= 100 {
							return true
						}
					}
				}
			}
		}
	} else {
		// Generic path
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				if a>>8 < 250 {
					continue
				}
				r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
				for _, t := range targets {
					if r8 == t[0] && g8 == t[1] && b8 == t[2] {
						count++
						if count >= 100 {
							return true
						}
					}
				}
			}
		}
	}
	return count >= 100
}

func cleanupTempFiles(overlayPath string) {
	base := strings.TrimSuffix(overlayPath, filepath.Ext(overlayPath))
	binPath := base + ".bin"
	if _, err := os.Stat(binPath); err == nil {
		os.Remove(binPath)
	}
	if _, err := os.Stat(overlayPath); err == nil && !strings.HasSuffix(overlayPath, ".webp") {
		os.Remove(overlayPath)
	}
}

// resizeToTarget uses high-quality CatmullRom scaling (much better than nearest).
func resizeToTarget(src image.Image, target image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(target)
	xdraw.BiLinear.Scale(dst, target, src, src.Bounds(), drawstd.Src, nil)
	return dst
}

// 60% alpha blend (matching historical behavior)
func blendImages(base, overlay image.Image, alpha float64) *image.RGBA {
	bounds := base.Bounds()
	dst := image.NewRGBA(bounds)

	alphaScale := uint32(alpha * 65535)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			bR, bG, bB, _ := base.At(x, y).RGBA()
			oR, oG, oB, oA := overlay.At(x, y).RGBA()

			// Apply per-pixel alpha from overlay, then the global 60% factor
			effAlpha := uint32(oA) * alphaScale / 65535

			outR := (bR*(65535-effAlpha) + oR*effAlpha) / 65535
			outG := (bG*(65535-effAlpha) + oG*effAlpha) / 65535
			outB := (bB*(65535-effAlpha) + oB*effAlpha) / 65535

			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(outR >> 8),
				G: uint8(outG >> 8),
				B: uint8(outB >> 8),
				A: 255,
			})
		}
	}
	return dst
}

// --- Legend drawing (simplified but functional version) ---

func drawLegend(img *image.RGBA, dateStr string) *image.RGBA {
	// Legend background rectangle (bottom-right, matching Python layout)
	legendRect := image.Rect(3080, 1780, 3820, 2150)
	draw.Draw(img, legendRect, &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Basic font (we can improve with embedded TTF later)
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
	}

	// Date
	d.Dot = fixed.P(3100, 1805)
	d.DrawString(dateStr)

	// Title
	d.Dot = fixed.P(3100, 1835)
	d.DrawString("Yallop visibility (Q):")

	entries := []struct {
		col   color.Color
		label string
	}{
		{color.RGBA{0, 204, 204, 255}, "A: Easily visible (naked eye)"},
		{color.RGBA{0, 179, 179, 255}, "B: Visible, perfect conditions"},
		{color.RGBA{255, 255, 26, 255}, "C: May need optical aid"},
		{color.RGBA{230, 230, 0, 255}, "D: Will need optical aid"},
		{color.RGBA{179, 179, 0, 255}, "E: Telescope only"},
		{color.RGBA{255, 0, 0, 255}, "First naked-eye visibility"},
		{color.RGBA{0, 0, 255, 255}, "First telescope visibility"},
	}

	y := 1870
	for _, e := range entries {
		// Draw colored square/ellipse
		colRect := image.Rect(3100, y+2, 3118, y+18)
		draw.Draw(img, colRect, &image.Uniform{e.col}, image.Point{}, draw.Src)

		// Label
		d.Dot = fixed.P(3135, y+15)
		d.DrawString(e.label)
		y += 26
	}

	return img
}

// --- WEBP output ---

func encodeWEBP(img image.Image, path string, quality int) error {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: float32(quality)}); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}