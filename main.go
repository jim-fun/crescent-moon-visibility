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
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
)

const (
	// DaysToProcess is the number of consecutive days to generate maps for
	// after each new moon (the new moon day + 2 following days).
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

func main() {
	var yearsStr string
	var startYear, endYear int
	var outDir string
	var maxWorkers int

	flag.StringVar(&yearsStr, "years", "", "Comma-separated list of years (e.g., 2027,2028)")
	flag.IntVar(&startYear, "start", 2027, "Start year (if -years is not set)")
	flag.IntVar(&endYear, "end", 2027, "End year (inclusive)")
	flag.StringVar(&outDir, "out", "output_maps", "Output directory for the generated maps")
	flag.IntVar(&maxWorkers, "workers", 4, "Number of parallel workers for CPU generation")
	flag.Parse()

	years := parseYears(yearsStr, startYear, endYear)
	if len(years) == 0 {
		fmt.Println("No years specified to process.")
		return
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

	fmt.Printf("Processing %d years. Found %d total new moons.\n", len(years), len(allMoons))
	fmt.Printf("Generating %d days per new moon = %d total maps\n", DaysToProcess, len(allMoons)*DaysToProcess)
	fmt.Printf("Output Directory: %s | Workers: %d\n", outDir, maxWorkers)

	// Build the task list: for each new moon, generate maps for N consecutive days.
	var tasks []task
	for _, nm := range allMoons {
		for i := 0; i < DaysToProcess; i++ {
			currentDate := nm.AddDate(0, 0, i)
			dateStr := currentDate.Format("2006-01-02")
			outputFile := filepath.Join(outDir, dateStr+".png")
			moonAge := currentDate.Sub(nm).Hours() / 24.0
			tasks = append(tasks, task{DateStr: dateStr, OutputFile: outputFile, MoonAge: moonAge})
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

	// Phase 1: Parallel CPU rendering via the C++ visibility.out binary.
	// Each worker picks tasks from the channel and shells out to the renderer.
	fmt.Println("\n[1/2] Starting parallel CPU generation (OpenMP calculation)...")
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for t := range taskCh {
				cmd := exec.Command("./visibility.out", t.DateStr, "map", TimeType, Method, t.OutputFile)
				if err := cmd.Run(); err != nil {
					fmt.Printf("✗ [Worker %d] Failed %s: %v\n", workerID, t.DateStr, err)
					continue
				}

				if stat, err := os.Stat(t.OutputFile); err == nil {
					fmt.Printf("✓ [Worker %d] Generated %s (%.2f MB)\n", workerID, t.OutputFile, float64(stat.Size())/(1024*1024))
					mu.Lock()
					mapFiles = append(mapFiles, fmt.Sprintf("%s|%.2f", t.OutputFile, t.MoonAge))
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("=> CPU generation complete in %.2fs.\n", time.Since(cpuStartTime).Seconds())

	// Phase 2: GPU-accelerated batch blending via OpenCV OpenCL T-API.
	// All generated PNGs are passed to gpu_blend.py in a single invocation
	// to amortise GPU context setup and base-map loading costs.
	if len(mapFiles) > 0 {
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
