package main

/*
#cgo CFLAGS: -I${SRCDIR}/thirdparty -O3
#cgo LDFLAGS: -lm
#include "astronomy.c"
*/
import "C"
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
)

const (
	DaysToProcess = 3
	Method        = "yallop"
	TimeType      = "evening"
)

func getNewMoonsInYear(year int) []time.Time {
	var moons []time.Time
	startTime := C.Astronomy_MakeTime(C.int(year), 1, 1, 0, 0, 0)
	for {
		result := C.Astronomy_SearchMoonPhase(0, startTime, 40)
		if result.status != C.ASTRO_SUCCESS {
			break
		}
		utc := C.Astronomy_UtcFromTime(result.time)
		if int(utc.year) != year {
			break
		}
		moonDate := time.Date(int(utc.year), time.Month(utc.month), int(utc.day), 0, 0, 0, 0, time.UTC)
		moons = append(moons, moonDate)
		startTime = C.Astronomy_AddDays(result.time, 20)
	}
	return moons
}

type task struct {
	DateStr    string
	OutputFile string
}

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

	var allMoons []time.Time
	for _, y := range years {
		moons := getNewMoonsInYear(y)
		allMoons = append(allMoons, moons...)
	}

	fmt.Printf("Processing %d years. Found %d total new moons.\n", len(years), len(allMoons))
	fmt.Printf("Generating %d days per new moon = %d total maps\n", DaysToProcess, len(allMoons)*DaysToProcess)
	fmt.Printf("Output Directory: %s | Workers: %d\n", outDir, maxWorkers)

	var tasks []task
	for _, nm := range allMoons {
		for i := 0; i < DaysToProcess; i++ {
			currentDate := nm.AddDate(0, 0, i)
			dateStr := currentDate.Format("2006-01-02")
			outputFile := filepath.Join(outDir, dateStr+".png")
			tasks = append(tasks, task{DateStr: dateStr, OutputFile: outputFile})
		}
	}

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
					mapFiles = append(mapFiles, t.OutputFile)
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("=> CPU generation complete in %.2fs.\n", time.Since(cpuStartTime).Seconds())

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
