// Package blend provides the final compositing step (previously implemented in
// the now-legacy gpu_blend.py).
//
// It takes raw overlay data produced by the CPU or GPU renderers, composites
// them onto the NASA base map using a 60% alpha blend, draws the legend,
// and writes high-quality WEBP output.
//
// The core pipeline is now free of Python dependencies.
package blend

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	drawstd "image/draw"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Target output resolution (4K)
const (
	TargetWidth  = 3840
	TargetHeight = 2160
)

//go:embed fonts/Inter-Regular.ttf
var interRegularTTF []byte

var (
	headerFace font.Face
	labelFace  font.Face
	fontOnce   sync.Once
)

func initFonts() {
	fontOnce.Do(func() {
		f, err := opentype.Parse(interRegularTTF)
		if err != nil {
			panic("failed to parse embedded Inter font: " + err.Error())
		}

		// Larger font for headers (date, moon age, title) — doubled
		headerFace, err = opentype.NewFace(f, &opentype.FaceOptions{
			Size:    44,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			panic("failed to create Inter header face: " + err.Error())
		}

		// Standard size for category labels — doubled
		labelFace, err = opentype.NewFace(f, &opentype.FaceOptions{
			Size:    30,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			panic("failed to create Inter label face: " + err.Error())
		}
	})
}

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

	// --- Draw first-visibility diamonds on the map ---
	// Only draw them in the blend layer if they are not already baked into the overlay
	// (CPU renderer bakes them; GPU does not).
	// We now check a small neighborhood around the detected point because the baked
	// diamond is a filled shape, not a single pixel.
	if nakedX, nakedY, teleX, teleY := findFirstVisibilityDiamonds(overlayResized); nakedX != -1 || teleX != -1 {
		// For CPU renders, the diamonds are already baked into the overlay by the C++ renderer.
		// To avoid double diamonds, we clear any strong red/blue pixels near the detected positions
		// before drawing our own (larger, outlined) versions. This ensures only one set appears.
		if nakedX != -1 && nakedY != -1 {
			clearStrongColorInNeighborhood(overlayResized, nakedX, nakedY, 255, 0, 0, 13)
			drawDiamond(blended, nakedX, nakedY, 12, color.RGBA{0, 0, 0, 255})   // thick black outline
			drawDiamond(blended, nakedX, nakedY, 9, color.RGBA{255, 0, 0, 255})   // red fill
		}
		if teleX != -1 && teleY != -1 {
			clearStrongColorInNeighborhood(overlayResized, teleX, teleY, 0, 0, 255, 13)
			drawDiamond(blended, teleX, teleY, 12, color.RGBA{0, 0, 0, 255})   // thick black outline
			drawDiamond(blended, teleX, teleY, 9, color.RGBA{0, 0, 255, 255})   // blue fill
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
// the easternmost positions for first naked-eye (A/B) and first telescope (C/D) visibility.
// It uses a more robust color-distance approach to handle small variations between
// CPU and GPU (especially FP32+DD) renderers.
func findFirstVisibilityDiamonds(overlay image.Image) (nakedX, nakedY, teleX, teleY int) {
	bounds := overlay.Bounds()
	nakedX, nakedY = -1, -1
	teleX, teleY = -1, -1

	// Target colors (approximate sRGB)
	targetAB := [3]float64{0, 191, 191}   // cyan-ish
	targetCD := [3]float64{255, 255, 38}  // yellow-ish

	bestABX, bestABY := -1, -1
	bestCDX, bestCDY := -1, -1

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Max.X - 1; x >= bounds.Min.X; x-- {
			r8, g8, b8, a8 := overlay.At(x, y).RGBA()
			if a8 < 0x8000 {
				continue
			}

			// Convert to 8-bit
			r := float64(r8 >> 8)
			g := float64(g8 >> 8)
			b := float64(b8 >> 8)

			// Color distance to A/B (cyan)
			distAB := math.Sqrt((r-targetAB[0])*(r-targetAB[0]) +
				(g-targetAB[1])*(g-targetAB[1]) +
				(b-targetAB[2])*(b-targetAB[2]))

			// Color distance to C/D (yellow)
			distCD := math.Sqrt((r-targetCD[0])*(r-targetCD[0]) +
				(g-targetCD[1])*(g-targetCD[1]) +
				(b-targetCD[2])*(b-targetCD[2]))

			// Thresholds (tuned to be tolerant of GPU color shifts)
			if distAB < 55 && (bestABX == -1 || x > bestABX) {
				bestABX, bestABY = x, y
			}
			if distCD < 55 && (bestCDX == -1 || x > bestCDX) {
				bestCDX, bestCDY = x, y
			}
		}
	}

	nakedX, nakedY = bestABX, bestABY
	teleX, teleY = bestCDX, bestCDY
	return
}



// formatDateForLegend converts "2026-01-19" style strings into
// human-friendly dates like "January 19, 2026".
func formatDateForLegend(dateStr string) string {
	// Expect format YYYY-MM-DD
	if len(dateStr) < 10 {
		return dateStr
	}
	year := dateStr[0:4]
	month := dateStr[5:7]
	day := dateStr[8:10]

	monthNames := map[string]string{
		"01": "January", "02": "February", "03": "March",
		"04": "April", "05": "May", "06": "June",
		"07": "July", "08": "August", "09": "September",
		"10": "October", "11": "November", "12": "December",
	}

	monthName := monthNames[month]
	if monthName == "" {
		return dateStr
	}

	// Remove leading zero from day (e.g. "08" → "8")
	dayNum := strings.TrimLeft(day, "0")
	if dayNum == "" {
		dayNum = "0"
	}

	return fmt.Sprintf("%s %s, %s", monthName, dayNum, year)
}

// hasStrongColorInNeighborhood checks a small square around (cx, cy) to see
// if there is a pixel that is very close to the target (r,g,b) color.
// This is used to detect when the CPU renderer has already baked in a
// first-visibility diamond so we don't draw a duplicate one in the blend layer.
func hasStrongColorInNeighborhood(img image.Image, cx, cy int, tr, tg, tb uint8, radius int) bool {
	bounds := img.Bounds()
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			x := cx + dx
			y := cy + dy
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			r8, g8, b8, a8 := img.At(x, y).RGBA()
			if a8 < 0xc000 { // require reasonably opaque
				continue
			}
			r := uint8(r8 >> 8)
			g := uint8(g8 >> 8)
			b := uint8(b8 >> 8)

			// Very strong match to the target color
			if r > 240 && g < 30 && b < 30 && tr == 255 {
				return true
			}
			if r < 30 && g < 30 && b > 240 && tb == 255 {
				return true
			}
		}
	}
	return false
}

// clearStrongColorInNeighborhood zeros out pixels that are strongly red or blue
// in a neighborhood. Used to remove diamonds baked by the CPU renderer before
// we draw our own versions in the blend layer.
//
// This version is intentionally more aggressive to reliably catch the diamonds
// that the C++ CPU renderer bakes into the raw overlay (including after resizing).
func clearStrongColorInNeighborhood(img *image.RGBA, cx, cy int, tr, tg, tb uint8, radius int) {
	bounds := img.Bounds()
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			x := cx + dx
			y := cy + dy
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			r8, g8, b8, a8 := img.At(x, y).RGBA()
			if a8 < 0x6000 { // lower threshold to catch anti-aliased edges
				continue
			}
			r := uint8(r8 >> 8)
			g := uint8(g8 >> 8)
			b := uint8(b8 >> 8)

			if tr == 255 {
				// Any pixel that is predominantly red (catches baked CPU diamonds reliably)
				if r > 160 && g < 140 && b < 140 {
					img.Set(x, y, color.RGBA{0, 0, 0, 0})
				}
			} else if tb == 255 {
				// Any pixel that is predominantly blue
				if r < 140 && g < 140 && b > 160 {
					img.Set(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
		}
	}
}

// --- Legend drawing (improved with proper diamonds) ---

func drawLegend(img *image.RGBA, dateStr string, moonAge float64) *image.RGBA {
	// Legend background rectangle (bottom-right). Geometry constants keep
	// padding consistent for every element below.
	const (
		legendL = 2720
		legendT = 1520
		legendR = 3830
		legendB = 2150
		padL    = 30 // inner padding from left edge
		padR    = 30 // inner padding from right edge
	)
	legendRect := image.Rect(legendL, legendT, legendR, legendB)
	drawstd.Draw(img, legendRect, &image.Uniform{color.White}, image.Point{}, drawstd.Src)

	// Thin dark border for definition (big aesthetic win)
	borderCol := color.RGBA{60, 60, 60, 255}
	for x := legendRect.Min.X; x < legendRect.Max.X; x++ {
		img.Set(x, legendRect.Min.Y, borderCol)
		img.Set(x, legendRect.Max.Y-1, borderCol)
	}
	for y := legendRect.Min.Y; y < legendRect.Max.Y; y++ {
		img.Set(legendRect.Min.X, y, borderCol)
		img.Set(legendRect.Max.X-1, y, borderCol)
	}

	initFonts()
	headerDrawer := &font.Drawer{Dst: img, Src: image.Black, Face: headerFace}
	labelDrawer := &font.Drawer{Dst: img, Src: image.Black, Face: labelFace}

	innerL := legendL + padL
	innerR := legendR - padR

	// === Header row: date on the left, moon age right-aligned ===
	headerY := legendT + 50
	headerDrawer.Dot = fixed.P(innerL, headerY)
	headerDrawer.DrawString(formatDateForLegend(dateStr))

	if moonAge > 0 {
		ageStr := fmt.Sprintf("Moon age: %.2f days", moonAge)
		ageW := headerDrawer.MeasureString(ageStr).Ceil()
		headerDrawer.Dot = fixed.P(innerR-ageW, headerY)
		headerDrawer.DrawString(ageStr)
	}

	// Subtle separator line under header
	sepY := headerY + 30
	for x := innerL; x < innerR; x++ {
		img.Set(x, sepY, color.RGBA{180, 180, 180, 255})
	}

	// Title + criterion row (criterion right-aligned to balance the title)
	titleY := sepY + 50
	headerDrawer.Dot = fixed.P(innerL, titleY)
	headerDrawer.DrawString("Visibility")

	critStr := "Criterion: " + CriterionName
	critW := labelDrawer.MeasureString(critStr).Ceil()
	labelDrawer.Dot = fixed.P(innerR-critW, titleY)
	labelDrawer.DrawString(critStr)

	// === Two-column layout for legend items ===
	// Left column: A–E (5 rows). Right column: 2 diamonds — vertically
	// centered against the left column so the right side doesn't look top-heavy.
	const rowH = 50
	leftEntries := []struct {
		col   color.Color
		label string
	}{
		{color.RGBA{0, 204, 204, 255}, "A: Easily visible (naked eye)"},
		{color.RGBA{0, 179, 179, 255}, "B: Visible, perfect conditions"},
		{color.RGBA{255, 255, 26, 255}, "C: May need optical aid"},
		{color.RGBA{230, 230, 0, 255}, "D: Will need optical aid"},
		{color.RGBA{179, 179, 0, 255}, "E: Telescope only"},
	}
	rightEntries := []struct {
		col   color.Color
		label string
	}{
		{color.RGBA{255, 0, 0, 255}, "First naked-eye visibility"},
		{color.RGBA{0, 0, 255, 255}, "First telescope visibility"},
	}

	itemsTopY := titleY + 40
	leftColX := innerL
	rightColX := legendL + (legendR-legendL)/2 + 20

	// Left column (A–E)
	for i, e := range leftEntries {
		y := itemsTopY + i*rowH
		colRect := image.Rect(leftColX, y, leftColX+28, y+24)
		drawstd.Draw(img, colRect, &image.Uniform{e.col}, image.Point{}, drawstd.Src)

		border := color.RGBA{30, 30, 30, 200}
		drawstd.Draw(img, image.Rect(leftColX, y, leftColX+28, y+1), &image.Uniform{border}, image.Point{}, drawstd.Src)
		drawstd.Draw(img, image.Rect(leftColX, y+23, leftColX+28, y+24), &image.Uniform{border}, image.Point{}, drawstd.Src)
		drawstd.Draw(img, image.Rect(leftColX, y, leftColX+1, y+24), &image.Uniform{border}, image.Point{}, drawstd.Src)
		drawstd.Draw(img, image.Rect(leftColX+27, y, leftColX+28, y+24), &image.Uniform{border}, image.Point{}, drawstd.Src)

		labelDrawer.Dot = fixed.P(leftColX+40, y+20)
		labelDrawer.DrawString(e.label)
	}

	// Right column — vertically center 2 entries against 5 left entries.
	// Left column spans 5 rows; right column has 2 rows → offset by 1.5 rows.
	rightStartY := itemsTopY + (len(leftEntries)-len(rightEntries))*rowH/2
	for i, e := range rightEntries {
		y := rightStartY + i*rowH
		drawDiamond(img, rightColX+14, y+12, 14, color.RGBA{0, 0, 0, 255})
		drawDiamond(img, rightColX+14, y+12, 11, e.col)

		labelDrawer.Dot = fixed.P(rightColX+40, y+20)
		labelDrawer.DrawString(e.label)
	}

	// Git repo reference, right-aligned inside the box using real text metrics.
	repoText := "github.com/jim-fun/crescent-moon-visibility"
	repoW := labelDrawer.MeasureString(repoText).Ceil()
	repoY := legendB - 20
	labelDrawer.Dot = fixed.P(innerR-repoW, repoY)
	labelDrawer.DrawString(repoText)

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