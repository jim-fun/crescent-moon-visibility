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

// CriterionName is surfaced in the legend. This is the first step toward
// making the visibility criterion configurable (per agentic review 2026-05).
// Default remains "Yallop 1997" for backward compatibility and proven accuracy.
var CriterionName = "Yallop 1997"

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
		moonAge := 0.0
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%f", &moonAge)
		}

		if err := processOne(overlayPath, baseResized, opts, moonAge); err != nil {
			fmt.Printf("✗ blend failed for %s: %v\n", overlayPath, err)
			// Continue with other files (matching old Python behavior)
		}
	}

	return nil
}

func processOne(overlayPath string, baseImg image.Image, opts BlendOptions, moonAge float64) error {
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

	// --- New: Draw first-visibility diamonds on the map (works for both CPU and GPU) ---
	if nakedX, nakedY, teleX, teleY := findFirstVisibilityDiamonds(overlayResized); nakedX != -1 || teleX != -1 {
		if nakedX != -1 && nakedY != -1 {
			// More visible diamonds on the map
			drawDiamond(blended, nakedX, nakedY, 10, color.RGBA{0, 0, 0, 255})   // thick black outline
			drawDiamond(blended, nakedX, nakedY, 7, color.RGBA{255, 0, 0, 255})   // red fill
		}
		if teleX != -1 && teleY != -1 {
			drawDiamond(blended, teleX, teleY, 10, color.RGBA{0, 0, 0, 255})   // thick black outline
			drawDiamond(blended, teleX, teleY, 7, color.RGBA{0, 0, 255, 255})   // blue fill
		}
	}

	// Draw legend with moon age
	finalImg := drawLegend(blended, filepath.Base(strings.TrimSuffix(overlayPath, filepath.Ext(overlayPath))), moonAge)

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

// drawDiamond draws a small filled diamond (rotated square) centered at (cx, cy)
// with the given size (half-diagonal) and color.
func drawDiamond(img *image.RGBA, cx, cy, size int, col color.Color) {
	for dy := -size; dy <= size; dy++ {
		absDY := dy
		if absDY < 0 {
			absDY = -absDY
		}
		for dx := -(size - absDY); dx <= (size - absDY); dx++ {
			x := cx + dx
			y := cy + dy
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x, y, col)
			}
		}
	}
}

// findFirstVisibilityDiamonds scans the classification overlay (resized) and returns
// approximate positions for the first naked-eye (A/B) and first telescope (C/D)
// visibility markers. It uses the easternmost (rightmost) qualifying pixel as a
// practical proxy. Returns -1,-1 if none found.
func findFirstVisibilityDiamonds(overlay image.Image) (nakedX, nakedY, teleX, teleY int) {
	bounds := overlay.Bounds()
	nakedX, nakedY = -1, -1
	teleX, teleY = -1, -1

	// Colors from the renderers (approximate; we check for non-transparent cyan/yellowish)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Max.X - 1; x >= bounds.Min.X; x-- { // scan right to left to find easternmost first
			r, g, b, a := overlay.At(x, y).RGBA()
			if a < 0x8000 {
				continue // transparent / no visibility
			}

			// Improved heuristic for both CPU and GPU renderers
			// A/B tend to be cyan (low red, high green+blue)
			// C/D tend to be yellowish (high red+green, lower blue)
			isAB := (r < 0x6000 && g > 0x7000 && b > 0x7000)
			isCD := (r > 0x7000 && g > 0x7000 && b < 0x6000)

			if isAB && nakedX == -1 {
				nakedX, nakedY = x, y
			}
			if isCD && teleX == -1 {
				teleX, teleY = x, y
			}

			if nakedX != -1 && teleX != -1 {
				return
			}
		}
	}
	return
}

// --- Legend drawing (improved with proper diamonds) ---

func drawLegend(img *image.RGBA, dateStr string, moonAge float64) *image.RGBA {
	// Legend background rectangle (bottom-right)
	legendRect := image.Rect(3080, 1780, 3820, 2150)
	draw.Draw(img, legendRect, &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Thin dark border for definition (big aesthetic win)
	borderCol := color.RGBA{60, 60, 60, 255}
	// Top
	for x := legendRect.Min.X; x < legendRect.Max.X; x++ {
		img.Set(x, legendRect.Min.Y, borderCol)
	}
	// Bottom
	for x := legendRect.Min.X; x < legendRect.Max.X; x++ {
		img.Set(x, legendRect.Max.Y-1, borderCol)
	}
	// Left
	for y := legendRect.Min.Y; y < legendRect.Max.Y; y++ {
		img.Set(legendRect.Min.X, y, borderCol)
	}
	// Right
	for y := legendRect.Min.Y; y < legendRect.Max.Y; y++ {
		img.Set(legendRect.Max.X-1, y, borderCol)
	}

	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
	}

	// === Header: Date + Moon Age ===
	headerY := 1798
	d.Dot = fixed.P(3100, headerY)
	d.DrawString(dateStr)

	if moonAge > 0 {
		ageStr := fmt.Sprintf("  |  Moon age: %.2f days", moonAge)
		d.Dot = fixed.P(3100 + 95, headerY)
		d.DrawString(ageStr)
	}

	// Subtle separator line under header
	sepY := 1820
	for x := 3100; x < 3810; x++ {
		img.Set(x, sepY, color.RGBA{180, 180, 180, 255})
	}

	// Title
	d.Dot = fixed.P(3100, 1838)
	d.DrawString("Yallop visibility (Q):")

	// Criterion label (early step toward pluggable criteria per agentic review)
	d.Dot = fixed.P(3100, 1852)
	d.DrawString("(" + CriterionName + ")")

	entries := []struct {
		col       color.Color
		label     string
		isDiamond bool
	}{
		{color.RGBA{0, 204, 204, 255}, "A: Easily visible (naked eye)", false},
		{color.RGBA{0, 179, 179, 255}, "B: Visible, perfect conditions", false},
		{color.RGBA{255, 255, 26, 255}, "C: May need optical aid", false},
		{color.RGBA{230, 230, 0, 255}, "D: Will need optical aid", false},
		{color.RGBA{179, 179, 0, 255}, "E: Telescope only", false},
		{color.RGBA{255, 0, 0, 255}, "First naked-eye visibility", true},
		{color.RGBA{0, 0, 255, 255}, "First telescope visibility", true},
	}

	y := 1878
	for _, e := range entries {
		swatchX := 3105
		swatchY := y + 2

		if e.isDiamond {
			// More prominent diamond for first-visibility markers
			drawDiamond(img, swatchX+10, swatchY+9, 10, color.RGBA{0, 0, 0, 255}) // black outline
			drawDiamond(img, swatchX+10, swatchY+9, 8, e.col)                    // colored fill
		} else {
			// Consistent, slightly larger swatches with clean border
			colRect := image.Rect(swatchX, swatchY, swatchX+20, swatchY+18)
			draw.Draw(img, colRect, &image.Uniform{e.col}, image.Point{}, draw.Src)

			border := color.RGBA{30, 30, 30, 200}
			// Top
			draw.Draw(img, image.Rect(swatchX, swatchY, swatchX+20, swatchY+1), &image.Uniform{border}, image.Point{}, draw.Src)
			// Bottom
			draw.Draw(img, image.Rect(swatchX, swatchY+17, swatchX+20, swatchY+18), &image.Uniform{border}, image.Point{}, draw.Src)
			// Left
			draw.Draw(img, image.Rect(swatchX, swatchY, swatchX+1, swatchY+18), &image.Uniform{border}, image.Point{}, draw.Src)
			// Right
			draw.Draw(img, image.Rect(swatchX+19, swatchY, swatchX+20, swatchY+18), &image.Uniform{border}, image.Point{}, draw.Src)
		}

		// Label
		d.Dot = fixed.P(3140, y+15)
		d.DrawString(e.label)
		y += 30
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