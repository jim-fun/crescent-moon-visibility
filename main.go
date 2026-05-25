// Crescent Moon Visibility Map Generator — Golang Orchestrator
//
// This program orchestrates the generation of crescent moon visibility maps
// across configurable year ranges. It uses the Astronomy Engine (via CGO in
// internal/astro) to compute new moon dates, dispatches parallel CPU workers
// to run the C++ renderer (visibility.out), and then hands off all generated
// images to the pure-Go blending package (internal/blend) that composites
// them onto the NASA base map and writes high-quality WEBP output.
//
//
// Supported GPU backends (auto-detected by OpenCL):
//   - macOS (Apple Silicon): Metal/OpenCL with automatic FP32 + double-double
//     time kernel (visibility_kernel_fp32.cl) when FP64 is unavailable.
//   - macOS (Intel): Metal/OpenCL (FP64 path when available)
//   - Linux/NVIDIA: CUDA via OpenCL (FP64)
//   - Linux/AMD: ROCm via OpenCL (FP64)
//   - Linux/Intel: Intel Compute Runtime via OpenCL (FP64 when present)
// The host (`gpu/gpu_render.c`) selects the appropriate kernel for maximum
// accuracy and compatibility. Both paths target ~97 % per-pixel fidelity.
package main

import (
	"flag"
	"fmt"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
	"github.com/jim-fun/crescent-moon-visibility/internal/blend"
)

// Version and build information are injected at build time via -ldflags.
var (
	version   = "dev"
	buildDate = "unknown"
)

const (
	// DaysToProcess is the number of consecutive days to generate maps for
	// after each new moon. The first map is the day AFTER the new moon (day +1)
	// since on the new moon day itself the crescent is too young (<24h) to be
	// realistically visible anywhere on Earth.
	DaysToProcess = 3

	// Method selects the visibility criterion used by the C++ renderer.
	// Supported values: "yallop", "odeh".
	Method = "yallop"

	// TimeType selects evening (waxing) or morning (waning) crescent.
	TimeType = "evening"
)

// getRendererCandidates returns possible filenames for a renderer binary,
// accounting for Windows .exe convention and common release artifact naming.
func getRendererCandidates(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			"bin/" + base + "-windows-amd64.exe",
			"./" + base + "-windows-amd64.exe",
			"bin/" + base + ".exe",
			"./" + base + ".exe",
			"bin/" + base + ".out", // fallback
			"./" + base + ".out",
		}
	}
	return []string{
		"bin/" + base + ".out",
		"./" + base + ".out",
	}
}

// task represents a single map-generation job.
type task struct {
	DateStr    string  // ISO date passed to visibility.out
	OutputFile string  // path to the output PNG
	MoonAge    float64 // days elapsed since the parent new moon
}

// parseYears parses the -years CSV flag or falls back to the -start/-end range.
// Returns a deduplicated, ordered slice of calendar years to process.
func parseYears(yearsStr string, startYear, endYear int) []int {
	var years []int
	if yearsStr != "" {
		parts := strings.Split(yearsStr, ",")
		for _, p := range parts {
			y, err := strconv.Atoi(strings.TrimSpace(p))
			if err == nil {
				years = append(years, y)
			}
		}
	} else if startYear > 0 && endYear >= startYear {
		for y := startYear; y <= endYear; y++ {
			years = append(years, y)
		}
	}
	return years
}

// newMoonCache stores previously computed new moon dates to avoid redundant
// CGO calls when the same year appears in multiple runs or overlapping ranges.
var newMoonCache = make(map[int][]time.Time)

// getNewMoonsCached returns new moon dates for a year, using a cache to avoid
// recomputing the expensive Astronomy Engine lunar phase search.
func getNewMoonsCached(year int) []time.Time {
	if cached, ok := newMoonCache[year]; ok {
		return cached
	}
	moons := astro.NewMoonsInYear(year)
	newMoonCache[year] = moons
	return moons
}

// parseMonths parses a comma-separated string of month numbers (1-12) into a set.
// Returns nil (zero value) when the input is empty — caller should use all months.
func parseMonths(s string) map[int]bool {
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

func main() {
	var yearsStr string
	var monthsStr string
	var startYear, endYear int
	var outDir string
	var maxWorkers int
	var useGPU bool
	var noBlend bool
	var showVersion bool

	flag.BoolVar(&showVersion, "version", false, "Print version and build date, then exit")
	flag.StringVar(&yearsStr, "years", "", "Comma-separated list of years (e.g., 2027,2028)")
	flag.StringVar(&monthsStr, "months", "", "Comma-separated list of months to process (e.g., 1,2 for Jan/Feb; 1-12) — overrides year range when set")
	flag.IntVar(&startYear, "start", 2027, "Start year (if -years is not set)")
	flag.IntVar(&endYear, "end", 2027, "End year (inclusive)")
	flag.StringVar(&outDir, "out", "output_maps", "Output directory for the generated maps")
	flag.IntVar(&maxWorkers, "workers", 4, "Number of parallel workers for CPU generation")
	flag.BoolVar(&useGPU, "gpu", false, "Use GPU renderer (bin/gpu_visibility.out) instead of CPU (bin/visibility.out)")
	flag.BoolVar(&noBlend, "noblend", false, "Skip GPU blending step — useful when data/map_nasa.png or OpenCV dependencies are unavailable")
	flag.Parse()

	if showVersion {
		fmt.Printf("crescent_maps version %s (built %s)\n", version, buildDate)

		// Try to report versions of the bundled renderers
		fmt.Println("Bundled renderers:")
		for _, base := range []string{"visibility", "gpu_visibility"} {
			names := []string{base + ".out", base + ".exe"}
			if runtime.GOOS == "windows" {
				names = []string{base + "-windows-amd64.exe", base + ".exe", base + ".out"}
			}
			found := false
			for _, name := range names {
				for _, dir := range []string{"bin", "."} {
					path := filepath.Join(dir, name)
					if _, err := os.Stat(path); err == nil {
						cmd := exec.Command(path, "-version")
						out, err := cmd.Output()
						if err == nil {
							fmt.Printf("  %-20s %s", name+":", string(out))
						} else {
							fmt.Printf("  %-20s (version query failed)\n", name+":")
						}
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				fmt.Printf("  %-20s (not found)\n", base+":")
			}
		}
		return
	}

	years := parseYears(yearsStr, startYear, endYear)
	if len(years) == 0 {
		fmt.Println("No years specified to process.")
		return
	}

	// Determine renderer: GPU binary if available and -gpu flag, otherwise CPU.
	var rendererBin string
	if useGPU {
		candidates := getRendererCandidates("gpu_visibility")
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				rendererBin = c
				break
			}
		}
		if rendererBin == "" {
			fmt.Printf("[warn] -gpu requested but no gpu_visibility binary found — falling back to CPU renderer\n")
			candidates = getRendererCandidates("visibility")
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					rendererBin = c
					break
				}
			}
		}
	} else {
		candidates := getRendererCandidates("visibility")
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				rendererBin = c
				break
			}
		}
		if rendererBin == "" {
			fmt.Printf("[warn] CPU renderer not found — run 'make cpu' (or the Windows equivalent) to build it\n")
			return
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Failed to create output directory %s: %v\n", outDir, err)
		return
	}

	// Phase 0: Compute new moon dates for all requested years (cached).
	var allMoons []time.Time
	for _, y := range years {
		moons := getNewMoonsCached(y)
		allMoons = append(allMoons, moons...)
	}

	// Filter by month if -months is set.
	selectedMonths := parseMonths(monthsStr)
	if selectedMonths != nil {
		var filtered []time.Time
		for _, m := range allMoons {
			if selectedMonths[int(m.Month())] {
				filtered = append(filtered, m)
			}
		}
		allMoons = filtered
	}

	fmt.Printf("Processing %d year(s). Found %d total new moons (%d maps).\n",
		len(years), len(allMoons), len(allMoons)*DaysToProcess)
	if selectedMonths != nil {
		var months []int
		for m := range selectedMonths {
			months = append(months, m)
		}
		sort.Ints(months)
		fmt.Printf("  Filtered to month(s): %s\n", strings.Join(func() []string {
			var ss []string
			for _, m := range months {
				ss = append(ss, strconv.Itoa(m))
			}
			return ss
		}(), ","))
	}
	fmt.Printf("Output Directory: %s | Workers: %d | Renderer: %s\n", outDir, maxWorkers, rendererBin)

	// Build the task list: for each new moon, walk forward day-by-day and pick
	// the first DaysToProcess days where the Moon will be at least
	// MinIlluminationFraction illuminated at the *latest* sunset anywhere on
	// UTC day D — observers near the date line on the west side see their
	// sunset at roughly D+1 06:00 UTC, so that's the natural sample point for
	// an "anywhere on Earth" criterion.
	//
	// 0.2 % illumination corresponds to ~5.1° elongation ≈ 9.5 h moon age,
	// below the typical aided-naked-eye threshold (~12-13 h) but in the range
	// where well-equipped observers in clear desert air have claimed sightings.
	// Below this the visibility map would be entirely empty everywhere.
	const MinIlluminationFraction = 0.002

	var tasks []task
	for _, nm := range allMoons {
		mapsGenerated := 0
		for d := 0; d <= 4 && mapsGenerated < DaysToProcess; d++ {
			midnight := time.Date(nm.Year(), nm.Month(), nm.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
			latestSunset := midnight.AddDate(0, 0, 1).Add(6 * time.Hour) // D+1 06:00 UTC
			if astro.MoonIlluminationFraction(latestSunset) < MinIlluminationFraction {
				continue
			}
			dateStr := midnight.Format("2006-01-02")
			outputFile := filepath.Join(outDir, dateStr)
			// Report the mid-day moon age (12:00 UTC) so the legend's value is
			// representative of the whole day rather than its end.
			midDay := midnight.Add(12 * time.Hour)
			moonAge := midDay.Sub(nm).Hours() / 24.0
			tasks = append(tasks, task{DateStr: dateStr, OutputFile: outputFile, MoonAge: moonAge})
			mapsGenerated++
		}
	}

	// Fan-out tasks to a buffered channel consumed by worker goroutines.
	taskCh := make(chan task, len(tasks))
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var wg sync.WaitGroup
	var mapFiles []string
	var mu sync.Mutex

	globalStartTime := time.Now()
	cpuStartTime := time.Now()

	// Phase 1: Parallel rendering via the C++ renderer binary.
	// Each worker picks tasks from the channel and shells out to the renderer.
	fmt.Println("\n[1/2] Starting parallel generation (" + func() string { if useGPU { return "GPU (OpenCL)" } else { return "CPU (OpenMP)" } }() + ")...")
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for t := range taskCh {
				cmd := exec.Command(rendererBin, t.DateStr, "map", TimeType, Method, t.OutputFile)
				// Forward the renderer's stderr so build/runtime errors (e.g. kernel
				// compile failures) are visible instead of being collapsed into a
				// bare "exit status 1". (Apple Silicon now works via the FP32+DD kernel.)
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("✗ [Worker %d] Failed %s: %v\n", workerID, t.DateStr, err)
					continue
				}

				// CPU renderer writes <path>.bin; GPU renderer writes to <path> directly.
				statPath := t.OutputFile
				if _, err := os.Stat(statPath); err != nil {
					if _, err := os.Stat(statPath + ".bin"); err == nil {
						statPath = statPath + ".bin"
					}
				}
				if stat, err := os.Stat(statPath); err == nil {
					fmt.Printf("✓ [Worker %d] Generated %s (%.2f MB)\n", workerID, t.OutputFile, float64(stat.Size())/(1024*1024))
					mu.Lock()
					mapFiles = append(mapFiles, fmt.Sprintf("%s|%.2f", t.OutputFile, t.MoonAge))
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	rendererName := "CPU"
	if useGPU {
		rendererName = "GPU"
	}
	fmt.Printf("=> %s generation complete in %.2fs.\n", rendererName, time.Since(cpuStartTime).Seconds())

	// Phase 2: Pure-Go blending (replacement for the previous gpu_blend.py).
	// Composites overlays onto the NASA base map, draws the legend, and writes
	// high-quality WEBP output.
	if len(mapFiles) > 0 && !noBlend {
		fmt.Println("\n[2/2] Starting Go blending...")

		gpuStart := time.Now()
		opts := blend.DefaultOptions()
		opts.OutputDir = outDir

		if err := blend.ProcessFiles(mapFiles, opts); err != nil {
			fmt.Printf("✗ Blending failed: %v\n", err)
		} else {
			fmt.Printf("=> Blending complete in %.2fs.\n", time.Since(gpuStart).Seconds())
		}
	}

	fmt.Printf("\nDone! Total Execution Time: %.2fs\n", time.Since(globalStartTime).Seconds())
}
