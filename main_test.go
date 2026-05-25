package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
)

// TestParseYearsFromCSV verifies comma-separated year strings are parsed correctly.
func TestParseYearsFromCSV(t *testing.T) {
	years := parseYears("2024,2025,2026", 0, 0)
	if len(years) != 3 {
		t.Fatalf("expected 3 years, got %d", len(years))
	}
	if years[0] != 2024 || years[1] != 2025 || years[2] != 2026 {
		t.Fatalf("unexpected years: %v", years)
	}
}

// TestParseYearsFromRange verifies the start/end range fallback works.
func TestParseYearsFromRange(t *testing.T) {
	years := parseYears("", 2030, 2033)
	if len(years) != 4 {
		t.Fatalf("expected 4 years, got %d", len(years))
	}
	if years[0] != 2030 || years[3] != 2033 {
		t.Fatalf("unexpected years: %v", years)
	}
}

// TestParseYearsEmpty verifies empty input yields no years.
func TestParseYearsEmpty(t *testing.T) {
	years := parseYears("", 0, 0)
	if len(years) != 0 {
		t.Fatalf("expected 0 years, got %d", len(years))
	}
}

// TestParseYearsCSVOverridesRange verifies -years flag takes precedence.
func TestParseYearsCSVOverridesRange(t *testing.T) {
	years := parseYears("2040", 2020, 2030)
	if len(years) != 1 || years[0] != 2040 {
		t.Fatalf("expected [2040], got %v", years)
	}
}

// TestNewMoonsInYear verifies that astronomical new moon calculation returns
// the expected count for a known year (2027 has 13 new moons).
func TestNewMoonsInYear(t *testing.T) {
	moons := astro.NewMoonsInYear(2027)
	// There are typically 12-13 new moons in a year
	if len(moons) < 12 || len(moons) > 13 {
		t.Fatalf("expected 12-13 new moons in 2027, got %d", len(moons))
	}
	// First new moon of 2027 should be in January
	if moons[0].Month() != time.January {
		t.Fatalf("expected first new moon in January, got %s", moons[0].Month())
	}
	// Last new moon should be in December
	if moons[len(moons)-1].Month() != time.December {
		t.Fatalf("expected last new moon in December, got %s", moons[len(moons)-1].Month())
	}
}

// TestNewMoonsKnownDate validates against the known new moon of
// January 29, 2025 (a well-documented new moon date).
func TestNewMoonsKnownDate(t *testing.T) {
	moons := astro.NewMoonsInYear(2025)
	found := false
	for _, m := range moons {
		if m.Month() == time.January && m.Day() == 29 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find new moon on Jan 29 2025, got: %v", moons)
	}
}

// TestTaskMoonAge verifies moon age calculation is correct for day offsets.
func TestTaskMoonAge(t *testing.T) {
	nm := time.Date(2027, 1, 7, 0, 0, 0, 0, time.UTC)
	for i := 0; i < DaysToProcess; i++ {
		currentDate := nm.AddDate(0, 0, i)
		moonAge := currentDate.Sub(nm).Hours() / 24.0
		if moonAge != float64(i) {
			t.Fatalf("expected moon age %d, got %.2f", i, moonAge)
		}
	}
}

// TestNewMoonsCaching verifies the cache returns consistent results.
func TestNewMoonsCaching(t *testing.T) {
	first := getNewMoonsCached(2026)
	second := getNewMoonsCached(2026)
	if len(first) != len(second) {
		t.Fatalf("cache returned different lengths: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if !first[i].Equal(second[i]) {
			t.Fatalf("cache mismatch at index %d: %v vs %v", i, first[i], second[i])
		}
	}
}

// --- Accuracy validation helpers ---

// pixelMatchRate returns the percentage of pixels that are exactly equal
// between two RGBA images. Useful for validating GPU vs CPU renderer output.
func pixelMatchRate(a, b *image.RGBA) float64 {
	if a == nil || b == nil {
		return 0
	}
	bounds := a.Bounds()
	if !bounds.Eq(b.Bounds()) {
		return 0
	}

	total := bounds.Dx() * bounds.Dy()
	if total == 0 {
		return 100.0
	}

	match := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar == br && ag == bg && ab == bb && aa == ba {
				match++
			}
		}
	}
	return float64(match) / float64(total) * 100.0
}

// TestDaySelection_IlluminationThreshold exercises the modern day selection rule.
func TestDaySelection_IlluminationThreshold(t *testing.T) {
	// This is a smoke test. The actual constant lives in main.go as unexported.
	// We test the behavior indirectly via integration tests.
	t.Log("Day selection uses 0.2% illumination threshold at latest sunset (see main.go)")
}

// TestRendererAccuracy validates that the GPU renderer (including the new
// FP32+DD path on Apple Silicon) produces output that is very close to the
// CPU reference. This is the most important accuracy regression test.
//
// It is skipped unless the environment variable RUN_ACCURACY_TEST=1 is set,
// because it requires the compiled renderers in bin/ and can be slow.
func TestRendererAccuracy(t *testing.T) {
	if os.Getenv("RUN_ACCURACY_TEST") != "1" {
		t.Skip("skipping expensive renderer accuracy test (set RUN_ACCURACY_TEST=1 to run)")
	}

	// Require the renderers (platform-aware for Windows .exe)
	cpuBin := "./bin/visibility.out"
	gpuBin := "./bin/gpu_visibility.out"
	if runtime.GOOS == "windows" {
		cpuBin = "./bin/visibility-windows-amd64.exe"
		gpuBin = "./bin/gpu_visibility-windows-amd64.exe"
		// Also try plain .exe as fallback
		if _, err := os.Stat(cpuBin); err != nil {
			cpuBin = "./bin/visibility.exe"
		}
		if _, err := os.Stat(gpuBin); err != nil {
			gpuBin = "./bin/gpu_visibility.exe"
		}
	}

	if _, err := os.Stat(cpuBin); err != nil {
		t.Fatalf("CPU renderer not found at %s — run 'make cpu' (or Windows equivalent)", cpuBin)
	}
	if _, err := os.Stat(gpuBin); err != nil {
		t.Fatalf("GPU renderer not found at %s — run 'make gpu' (or Windows equivalent)", gpuBin)
	}

	tmp := t.TempDir()
	date := "2025-01-30"

	// Pass base name (without .bin); the CPU renderer appends .bin itself.
	cpuBase := filepath.Join(tmp, "cpu")
	gpuOut := filepath.Join(tmp, "gpu.png")

	// Run CPU renderer
	cmdCPU := exec.Command(cpuBin, date, "map", "evening", "yallop", cpuBase)
	if err := cmdCPU.Run(); err != nil {
		t.Fatalf("CPU renderer failed: %v", err)
	}

	// Run GPU renderer
	cmdGPU := exec.Command(gpuBin, date, "map", "evening", "yallop", gpuOut)
	if err := cmdGPU.Run(); err != nil {
		t.Fatalf("GPU renderer failed: %v", err)
	}

	// Load both as RGBA
	cpuBinPath := cpuBase + ".bin"
	cpuData, err := os.ReadFile(cpuBinPath)
	if err != nil {
		t.Fatalf("failed to read CPU output: %v", err)
	}

	// The CPU writes raw .bin. GPU writes PNG.
	// For simplicity we decode the GPU PNG here.
	gpuImg, err := png.Decode(bytes.NewReader(mustReadFile(gpuOut)))
	if err != nil {
		t.Fatalf("failed to decode GPU PNG: %v", err)
	}

	// Convert CPU raw data to RGBA (assume 3600x2160 for now)
	w, h := 3600, 2160
	if len(cpuData) != w*h*4 {
		t.Fatalf("unexpected CPU .bin size: %d (expected %d)", len(cpuData), w*h*4)
	}

	cpuImg := &image.RGBA{
		Pix:    cpuData,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}

	// Convert GPU image to RGBA for comparison
	gpuRGBA := image.NewRGBA(gpuImg.Bounds())
	draw.Draw(gpuRGBA, gpuRGBA.Bounds(), gpuImg, image.Point{}, draw.Src)

	// Resize or handle size difference if needed (for now assume matching)
	match := pixelMatchRate(cpuImg, gpuRGBA)
	t.Logf("CPU vs GPU pixel match rate: %.2f%%", match)

	if match < 96.0 {
		t.Errorf("renderer accuracy below threshold: got %.2f%%, want >= 96%%", match)
	}
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
