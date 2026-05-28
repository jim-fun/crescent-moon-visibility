// Package jobspec holds the pure orchestration logic shared by the CLI and
// the web server: year/month parsing, renderer discovery, the cached new-moon
// lookup, and the rule that turns a set of years into concrete render tasks.
//
// Nothing in this package executes the renderer or the blender — see
// internal/runner for that. Keeping planning separate from execution lets the
// CLI (main.go) and the web server (internal/server) build identical task
// lists without duplicating the day-selection rule.
package jobspec

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
)

const (
	// DaysToProcess is the number of consecutive days to generate maps for
	// after each new moon. The day of conjunction itself is skipped since the
	// crescent is too young (<24h) to be realistically visible anywhere.
	DaysToProcess = 3

	// Method selects the visibility criterion used by the renderer.
	Method = "yallop"

	// TimeType selects evening (waxing) or morning (waning) crescent.
	TimeType = "evening"

	// MinIlluminationFraction is the threshold below which a candidate day is
	// skipped — the visibility map would be empty everywhere. 0.2 % ≈ 9.5 h
	// moon age, below the typical aided-naked-eye threshold (~12-13 h).
	MinIlluminationFraction = 0.002
)

// Task represents a single map-generation job — one date, one output path.
type Task struct {
	DateStr    string  // ISO date passed to the renderer
	OutputFile string  // path to the output file (without .bin suffix)
	MoonAge    float64 // days elapsed since the parent new moon, at mid-day UTC
}

// ParseYears parses the -years CSV flag or falls back to the -start/-end range.
// Returns an ordered slice of calendar years; may be empty if neither form is set.
func ParseYears(yearsStr string, startYear, endYear int) []int {
	var years []int
	if yearsStr != "" {
		for _, p := range strings.Split(yearsStr, ",") {
			y, err := strconv.Atoi(strings.TrimSpace(p))
			if err == nil {
				years = append(years, y)
			}
		}
		return years
	}
	if startYear > 0 && endYear >= startYear {
		for y := startYear; y <= endYear; y++ {
			years = append(years, y)
		}
	}
	return years
}

// ParseMonths parses a comma-separated string of month numbers (1-12) into a
// set. Returns nil when the input is empty — callers should treat nil as
// "all months".
func ParseMonths(s string) map[int]bool {
	if s == "" {
		return nil
	}
	m := make(map[int]bool)
	for _, p := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil && n >= 1 && n <= 12 {
			m[n] = true
		}
	}
	return m
}

// RendererCandidates returns possible filenames for a renderer binary,
// accounting for Windows .exe convention and the release artifact naming.
func RendererCandidates(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			"bin/" + base + "-windows-amd64.exe",
			"./" + base + "-windows-amd64.exe",
			"bin/" + base + ".exe",
			"./" + base + ".exe",
			"bin/" + base + ".out",
			"./" + base + ".out",
		}
	}
	return []string{
		"bin/" + base + ".out",
		"./" + base + ".out",
	}
}

// GetNewMoonsForYear returns the list of new moon dates (as "YYYY-MM-DD" strings)
// for the given year. This is the function the web UI should use for the
// "select year → choose new moon" dropdown in the point query feature.
func GetNewMoonsForYear(year int) []string {
	moons := GetNewMoonsCached(year)
	out := make([]string, len(moons))
	for i, t := range moons {
		out[i] = t.Format("2006-01-02")
	}
	return out
}

// GetNewMoonsCached returns the new moon times for a given year, using a simple
// in-memory cache.
var newMoonCache = struct {
	sync.Mutex
	m map[int][]time.Time
}{m: make(map[int][]time.Time)}

func GetNewMoonsCached(year int) []time.Time {
	newMoonCache.Lock()
	defer newMoonCache.Unlock()

	if moons, ok := newMoonCache.m[year]; ok {
		return moons
	}

	moons := astro.NewMoonsInYear(year)
	if len(moons) == 0 {
		// Fallback for web build or when astro returns nothing
		moons = approximateNewMoonsForYear(year)
	}

	newMoonCache.m[year] = moons
	return moons
}

// approximateNewMoonsForYear uses a mean synodic month for web builds.
func approximateNewMoonsForYear(year int) []time.Time {
	epoch := time.Date(2000, 1, 6, 12, 0, 0, 0, time.UTC)
	synodicMonth := time.Duration(29.530588853 * float64(24*time.Hour))

	var moons []time.Time
	t := epoch
	for t.Year() > year-1 {
		t = t.Add(-synodicMonth)
	}
	for {
		if t.Year() > year {
			break
		}
		if t.Year() == year {
			moons = append(moons, t)
		}
		t = t.Add(synodicMonth)
	}
	return moons
}

// BuildTasks turns a set of years + optional month filter into the list of
// concrete (date, output path, moon age) tasks that need to be rendered.
// This is the single source of truth for the "which days do we render?" rule.
func BuildTasks(years []int, months map[int]bool, outDir string) []Task {
	var tasks []Task

	for _, y := range years {
		moons := GetNewMoonsCached(y)

		for _, nm := range moons {
			mapsGenerated := 0
			for d := 0; d <= 4 && mapsGenerated < DaysToProcess; d++ {
				midnight := time.Date(nm.Year(), nm.Month(), nm.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
				latestSunset := midnight.AddDate(0, 0, 1).Add(6 * time.Hour)

				if astro.MoonIlluminationFraction(latestSunset) < MinIlluminationFraction {
					continue
				}
				if months != nil && !months[int(midnight.Month())] {
					continue
				}

				dateStr := midnight.Format("2006-01-02")
				outputFile := filepath.Join(outDir, dateStr)

				midDay := midnight.Add(12 * time.Hour)
				moonAge := midDay.Sub(nm).Hours() / 24.0

				tasks = append(tasks, Task{
					DateStr:    dateStr,
					OutputFile: outputFile,
					MoonAge:    moonAge,
				})
				mapsGenerated++
			}
		}
	}
	return tasks
}
