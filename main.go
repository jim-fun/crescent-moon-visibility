// Crescent Moon Visibility Map Generator — Golang Orchestrator
//
// This program orchestrates the generation of crescent moon visibility maps
// across configurable year ranges. It uses the Astronomy Engine (via CGO in
// internal/astro) to compute new moon dates, dispatches parallel CPU workers
// to run the C++ renderer (visibility.out), and then hands off all generated
// images to a GPU-accelerated Python blending script (gpu_blend.py) that
// composites them onto a NASA base map using OpenCV's OpenCL Transparent API.
//
// Supported GPU backends (auto-detected by OpenCL):
//   - macOS: Metal/OpenCL
//   - Linux/NVIDIA: CUDA via OpenCL
//   - Linux/AMD: ROCm via OpenCL
//   - Linux/Intel: Intel GPU Compute Runtime via OpenCL
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
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

	flag.StringVar(&yearsStr, "years", "", "Comma-separated list of years (e.g., 2027,2028)")
	flag.StringVar(&monthsStr, "months", "", "Comma-separated list of months to process (e.g., 1,2 for Jan/Feb; 1-12) — overrides year range when set")
	flag.IntVar(&startYear, "start", 2027, "Start year (if -years is not set)")
	flag.IntVar(&endYear, "end", 2027, "End year (inclusive)")
	flag.StringVar(&outDir, "out", "output_maps", "Output directory for the generated maps")
	flag.IntVar(&maxWorkers, "workers", 4, "Number of parallel workers for CPU generation")
	flag.BoolVar(&useGPU, "gpu", false, "Use GPU renderer (gpu_visibility.out) instead of CPU (visibility.out)")
	flag.BoolVar(&noBlend, "noblend", false, "Skip GPU blending step — useful when map_nasa.png or OpenCV dependencies are unavailable")
	flag.Parse()

	years := parseYears(yearsStr, startYear, endYear)
	if len(years) == 0 {
		fmt.Println("No years specified to process.")
		return
	}

	// Determine renderer: GPU binary if available and -gpu flag, otherwise CPU.
	var rendererBin string
	if useGPU {
		gpuBin := "./gpu_visibility.out"
		if _, err := os.Stat(gpuBin); err == nil {
			rendererBin = gpuBin
		} else {
			fmt.Printf("[warn] -gpu requested but %s not found — falling back to CPU renderer\n", gpuBin)
			rendererBin = "./visibility.out"
		}
	} else {
		rendererBin = "./visibility.out"
		if _, err := os.Stat(rendererBin); err != nil {
			fmt.Printf("[warn] %s not found — run 'make cpu' to build the CPU renderer\n", rendererBin)
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
	// the first DaysToProcess days where the crescent is at least 16 h old at a
	// reference observation moment (18:00 UTC). This way an early-morning
	// conjunction (~02 UTC) still yields a useful day-of-conjunction map (~16 h),
	// and a late-night conjunction (~22 UTC) correctly skips day +1 where the
	// crescent would only be ~20 h old somewhere unhelpful and instead emits
	// days +2..+4.
	const minMoonAgeHours = 16.0
	const referenceHourUTC = 18

	var tasks []task
	for _, nm := range allMoons {
		mapsGenerated := 0
		for d := 0; d <= 4 && mapsGenerated < DaysToProcess; d++ {
			midnight := time.Date(nm.Year(), nm.Month(), nm.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
			referenceEvening := midnight.Add(referenceHourUTC * time.Hour)
			hoursSinceConjunction := referenceEvening.Sub(nm).Hours()
			if hoursSinceConjunction < minMoonAgeHours {
				continue
			}
			dateStr := midnight.Format("2006-01-02")
			outputFile := filepath.Join(outDir, dateStr)
			moonAge := hoursSinceConjunction / 24.0
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

	// Phase 2: GPU-accelerated batch blending via OpenCV OpenCL T-API.
	// All generated PNGs are passed to gpu_blend.py in a single invocation
	// to amortise GPU context setup and base-map loading costs.
	if len(mapFiles) > 0 && !noBlend {
		fmt.Println("\n[2/2] Starting universal GPU batch blending (macOS Metal / AMD ROCm / NVIDIA CUDA)...")

		args := append([]string{"gpu_blend.py"}, mapFiles...)
		cmd := exec.Command("python3", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		gpuStart := time.Now()
		if err := cmd.Run(); err != nil {
			fmt.Printf("✗ GPU blending failed: %v\n", err)
		} else {
			fmt.Printf("=> GPU blending complete in %.2fs.\n", time.Since(gpuStart).Seconds())
		}
	}

	fmt.Printf("\nDone! Total Execution Time: %.2fs\n", time.Since(globalStartTime).Seconds())
}
