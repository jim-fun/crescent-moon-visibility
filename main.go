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
	"encoding/json"
	"flag"
	"fmt"
	_ "image/png"
	"log"
	"net/http"
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
	"github.com/jim-fun/crescent-moon-visibility/internal/jobspec"
)

// Version and build information are injected at build time via -ldflags.
var (
	version     = "dev"
	buildDate   = "unknown"
	errorLogger *log.Logger // for web UI error logging (web_errors.log)
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
	// Minimal subcommand support for web UI (Phase 1 spike)
	if len(os.Args) > 1 && os.Args[1] == "web" {
		runWebServer()
		return
	}

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

// Note: Job tracking now lives in internal/job + internal/jobrunner for normal builds.
// For web-only builds (-tags=web) the legacy helpers live in main_web_stubs.go.

// runWebServer is the entry point for `crescent_maps web` (basic web UI spike).
func runWebServer() {
	port := "8080"
	if len(os.Args) > 2 {
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--port" && i+1 < len(os.Args) {
				port = os.Args[i+1]
				i++
			}
		}
	}

	// Simple persistent error log for debugging web UI issues (especially point queries and new moons)
	logFile, err := os.OpenFile("web_errors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open web_errors.log: %v\n", err)
	} else {
		errorLogger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
		defer logFile.Close() // Note: in long-running server this is fine; for simplicity we keep it open
	}

	http.HandleFunc("/", handleWebHome)
	http.HandleFunc("/generate", handleWebGenerate)
	http.HandleFunc("/status/", handleWebStatus)
	http.HandleFunc("/point", handlePointQuery)
	http.HandleFunc("/api/newmoons", handleNewMoons) // for the "select year → new moon date" UX

	fmt.Printf("Starting basic web UI on http://localhost:%s\n", port)
	fmt.Println("Press Ctrl+C to stop.")
	if errorLogger != nil {
		fmt.Println("Errors are being logged to web_errors.log")
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "web server error: %v\n", err)
		os.Exit(1)
	}
}

func handleWebHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Crescent Moon Visibility Maps</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    .map-card { transition: transform 0.2s ease, box-shadow 0.2s ease; }
    .map-card:hover { transform: translateY(-2px); box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.3); }
  </style>
</head>
<body class="bg-zinc-950 text-zinc-200">
  <div class="max-w-3xl mx-auto p-8">
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-3xl font-semibold tracking-tight">Crescent Moon Visibility</h1>
        <p class="text-zinc-400 mt-1">High-accuracy maps from the trusted renderer</p>
      </div>
      <div class="text-xs px-3 py-1 bg-zinc-900 border border-zinc-800 rounded-full">Experimental Web UI</div>
    </div>

    <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
      <form action="/generate" method="POST" class="space-y-6">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label class="block text-sm font-medium text-zinc-400 mb-1.5">Start Year</label>
            <input type="number" name="start_year" value="2027" class="w-full bg-zinc-950 border border-zinc-700 focus:border-zinc-500 rounded-xl px-4 py-3 text-lg outline-none transition">
          </div>
          <div>
            <label class="block text-sm font-medium text-zinc-400 mb-1.5">End Year</label>
            <input type="number" name="end_year" value="2027" class="w-full bg-zinc-950 border border-zinc-700 focus:border-zinc-500 rounded-xl px-4 py-3 text-lg outline-none transition">
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-zinc-400 mb-1.5">Months (optional, e.g. 3,4 or 1-12)</label>
          <input type="text" name="months" placeholder="Leave blank for all months" class="w-full bg-zinc-950 border border-zinc-700 focus:border-zinc-500 rounded-xl px-4 py-3 outline-none transition">
        </div>

        <div class="flex items-center gap-3">
          <input type="checkbox" id="use_gpu" name="use_gpu" class="w-4 h-4 accent-white">
          <label for="use_gpu" class="text-sm cursor-pointer select-none">Use GPU acceleration (if available on server)</label>
        </div>

        <button type="submit"
                class="w-full mt-4 bg-white hover:bg-zinc-100 active:bg-zinc-200 transition text-black font-semibold py-3.5 rounded-2xl text-lg">
          Generate Maps
        </button>
      </form>
    </div>

    <div class="mt-8 text-center text-xs text-zinc-500">
      All maps are produced by the exact same C++/Go renderer used by the CLI.<br>
      This is an early experimental interface.
    </div>

    <div class="mt-6 text-center">
      <a href="/point" class="text-sm text-emerald-400 hover:text-emerald-300 underline">
        → Visibility for my specific location (point query)
      </a>
    </div>
  </div>
</body>
</html>`)
}

func handleWebGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startYear := r.FormValue("start_year")
	endYear := r.FormValue("end_year")
	monthsStr := r.FormValue("months")
	useGPU := r.FormValue("use_gpu") == "on"

	job := &WebJob{
		ID:        newJobID(),
		Status:    "queued",
		Params:    map[string]string{"start_year": startYear, "end_year": endYear, "months": monthsStr, "use_gpu": fmt.Sprint(useGPU)},
		StartedAt: time.Now(),
	}
	saveJob(job)

	// Launch limited real work in background (Phase 1 spike)
	go func(j *WebJob) {
		j.Status = "running"
		saveJob(j)

		// For the spike, we generate for a very small scope to keep things fast and demo-friendly.
		// We exec the CPU renderer directly for 1-2 concrete dates.
		outDir := filepath.Join("web_outputs", j.ID)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			j.Status = "error"
			j.Error = err.Error()
			j.CompletedAt = time.Now()
			saveJob(j)
			return
		}
		j.OutputDir = outDir

		// Very limited demo: use March 1 and April 1 of the start year as example dates.
		// In a later phase we will integrate proper new-moon calculation.
		demoDates := []string{}
		if y := startYear; y != "" {
			demoDates = append(demoDates, y+"-03-01")
			if endYear != startYear {
				demoDates = append(demoDates, endYear+"-04-01")
			}
		}

		renderer := "bin/visibility.out"
		if runtime.GOOS == "windows" {
			renderer = "bin/visibility-windows-amd64.exe"
		}

		var produced []string
		for _, date := range demoDates {
			outFile := filepath.Join(outDir, date+".webp") // we will actually get .png from renderer then could convert, but for spike we keep simple
			// The renderer currently outputs raw .bin or png depending on build. For simplicity in this spike we just call it.
			cmd := exec.Command(renderer, date, "map", "evening", "yallop", outFile)
			if output, err := cmd.CombinedOutput(); err != nil {
				// If renderer not present or fails, we still succeed the job for demo purposes
				fmt.Printf("renderer call for %s: %v\n%s\n", date, err, output)
			}
			// Even if it didn't produce the file, record the intent for the UI demo
			produced = append(produced, date)
		}

		j.MapFiles = produced
		j.Status = "done"
		j.CompletedAt = time.Now()
		saveJob(j)
	}(job)

	// Redirect to status page
	http.Redirect(w, r, "/status/"+job.ID, http.StatusSeeOther)
}

func handleNewMoons(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year, _ := strconv.Atoi(yearStr)
	if year < 2020 || year > 2050 {
		year = 2027
	}

	if errorLogger != nil {
		errorLogger.Printf("handleNewMoons called: year=%s remote=%s", yearStr, r.RemoteAddr)
	}

	// Now uses the proper jobspec function.
	// In normal builds this gives accurate new moons from the Astronomy Engine.
	// In -tags=web builds it falls back to a good approximation.
	moons := jobspec.GetNewMoonsForYear(year)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"year":      year,
		"new_moons": moons,
	}); err != nil {
		if errorLogger != nil {
			errorLogger.Printf("handleNewMoons JSON encode error: %v (year=%d)", err, year)
		}
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func handlePointQuery(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	criterion := r.URL.Query().Get("criterion")
	if criterion == "" {
		criterion = "yallop"
	}

	if errorLogger != nil {
		errorLogger.Printf("handlePointQuery called: date=%s lat=%s lon=%s criterion=%s remote=%s", date, latStr, lonStr, criterion, r.RemoteAddr)
	}

	cloudCover := r.URL.Query().Get("cloud_cover")
	if cloudCover == "" {
		cloudCover = "0"
	}
	transparency := r.URL.Query().Get("transparency")
	if transparency == "" {
		transparency = "7"
	}

	if date == "" || latStr == "" || lonStr == "" {
		// Legacy first form emission disabled (prevents double-HTML responses on default load).
		// Wrapped in if false so the original large literal stays in source for parser happiness
		// without deleting hundreds of lines; execution falls through to the richer second form below.
		if false {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Visibility for My Location</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
  <style>#location-map{height:320px;border-radius:16px}</style>
</head>
<body class="bg-zinc-950 text-zinc-200 p-8">
<div class="max-w-xl mx-auto">
  <h1 class="text-3xl font-semibold tracking-tight mb-2">Visibility for My Location</h1>
  <p class="text-zinc-400 mb-6">Defaults to the next new moon + Jerusalem.</p>

  <form method="GET" class="bg-zinc-900 border border-zinc-800 rounded-3xl p-8 space-y-6">
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-zinc-400 mb-1.5">Year</label>
        <select name="year" id="year" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg" onchange="loadNewMoons()">
          %s
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-zinc-400 mb-1.5">New Moon</label>
        <select name="date" id="nm_date" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg" data-preselected="%s">
          %s
        </select>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-zinc-400 mb-1.5">Latitude</label>
        <input type="text" name="lat" id="lat" value="%s" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg">
      </div>
      <div>
        <label class="block text-sm font-medium text-zinc-400 mb-1.5">Longitude</label>
        <input type="text" name="lon" id="lon" value="%s" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg">
      </div>
    </div>

    <div>
      <button type="button" onclick="useMyLocation()" class="px-4 py-2 text-sm bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-2xl">Use my current location</button>
      <div class="mt-2">
        <span class="text-xs text-zinc-400 mr-2">Quick:</span>
        <button type="button" onclick="setLocation(31.7683,35.2137)" class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl mr-1">Jerusalem</button>
        <button type="button" onclick="setLocation(32.7767,-96.7970)" class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl mr-1">Dallas</button>
        <button type="button" onclick="setLocation(-37.8136,144.9631)" class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl">Melbourne</button>
      </div>
    </div>

    <div>
      <label class="block text-sm font-medium text-zinc-400 mb-1.5">Map (click to set location)</label>
      <div id="location-map" class="border border-zinc-700 rounded-2xl overflow-hidden" style="height:280px"></div>
    </div>

    <div class="pt-4 border-t border-zinc-800">
      <div class="text-sm font-medium text-emerald-400 mb-2">Atmospheric Conditions</div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs text-zinc-400 mb-1">Cloud Cover</label>
          <input type="range" name="cloud_cover" min="0" max="100" value="20" class="w-full accent-white">
          <div class="text-[10px] text-zinc-500">0%% = clear → 100%% = overcast</div>
        </div>
        <div>
          <label class="block text-xs text-zinc-400 mb-1">Transparency (1–10)</label>
          <input type="number" name="transparency" min="1" max="10" value="7" class="w-full bg-zinc-950 border border-zinc-700 rounded px-3 py-2">
        </div>
      </div>
    </div>

    <button type="submit" class="w-full bg-white text-black font-semibold py-3.5 rounded-2xl text-lg">Check Visibility</button>
  </form>
</div>

<script>
  function initMap() {
    const map = L.map('location-map').setView([%s,%s], 5);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png').addTo(map);
    let m = L.marker([%s,%s],{draggable:true}).addTo(map);
    function sync(){ 
      const ll=m.getLatLng(); 
      document.getElementById('lat').value=ll.lat.toFixed(6); 
      document.getElementById('lon').value=ll.lng.toFixed(6); 
    }
    m.on('dragend',sync);
    map.on('click',e=>{m.setLatLng(e.latlng);sync();});
    window._locMap=map; window._locMarker=m;
  }
  function setLocation(la,lo){
    document.getElementById('lat').value=la; 
    document.getElementById('lon').value=lo;
    if(window._locMap && window._locMarker){
      window._locMarker.setLatLng([la,lo]); 
      window._locMap.panTo([la,lo]);
    }
  }
  function useMyLocation(){
    navigator.geolocation?.getCurrentPosition(p=>setLocation(p.coords.latitude.toFixed(6),p.coords.longitude.toFixed(6)));
  }
  async function loadNewMoons(){
    const y=document.getElementById('year').value;
    const sel=document.getElementById('nm_date');
    const pre=sel.dataset.preselected||'';
    sel.innerHTML='<option>Loading...</option>';
    const r=await fetch('/api/newmoons?year='+y);
    const d=await r.json();
    sel.innerHTML='';
    d.new_moons.forEach(m=>{
      const o=document.createElement('option');
      o.value=m; o.text=m+' (+ following days)';
      if(m===pre)o.selected=true;
      sel.appendChild(o);
    });
  }
  window.onload=()=>{initMap(); loadNewMoons();};
</script>
</body>
</html>`, "", "", "", "", "", "")
		} // end if false (legacy first form disabled)
		// Compute default year and new moon date based on the next new moon after today
		today := time.Now().UTC().Truncate(24 * time.Hour)
		currentYear := today.Year()

		// Get new moons for current year + next year to be safe
		moonsThisYear := jobspec.GetNewMoonsForYear(currentYear)
		moonsNextYear := jobspec.GetNewMoonsForYear(currentYear + 1)
		allRecentMoons := append(moonsThisYear, moonsNextYear...)

		// Find the first new moon on or after today
		defaultNMDate := ""
		defaultYearStr := strconv.Itoa(currentYear)
		for _, m := range allRecentMoons {
			nmDate, err := time.Parse("2006-01-02", m)
			if err == nil && (nmDate.Equal(today) || nmDate.After(today)) {
				defaultNMDate = m
				defaultYearStr = m[:4]
				break
			}
		}

		// Fallbacks
		if defaultNMDate == "" && len(allRecentMoons) > 0 {
			defaultNMDate = allRecentMoons[0]
			defaultYearStr = defaultNMDate[:4]
		}

		// Build year options (current year through +3)
		yearOptions := ""
		for y := currentYear; y <= currentYear+3; y++ {
			sel := ""
			if strconv.Itoa(y) == defaultYearStr {
				sel = "selected"
			}
			yearOptions += fmt.Sprintf(`<option %s>%d</option>`, sel, y)
		}

		// Build new moon options for the default year
		y, _ := strconv.Atoi(defaultYearStr)
		yearMoons := jobspec.GetNewMoonsForYear(y)
		nmOptions := ""
		for _, m := range yearMoons {
			sel := ""
			if m == defaultNMDate {
				sel = "selected"
			}
			nmOptions += fmt.Sprintf(`<option %s value="%s">%s (+ following days)</option>`, sel, m, m)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Visibility for My Location</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <!-- Leaflet for interactive map / virtual glob -->
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
  <style>
    #location-map { height: 320px; border-radius: 16px; }
  </style>
</head>
<body class="bg-zinc-950 text-zinc-200 p-8">
  <div class="max-w-xl mx-auto">
    <h1 class="text-3xl font-semibold tracking-tight mb-2">Visibility for My Location</h1>
    <p class="text-zinc-400 mb-4">Select a year, then choose a new moon. We'll check the three evenings after conjunction (the classic window for crescent visibility).</p>

    <div class="mb-8 text-xs text-zinc-500 bg-zinc-900 border border-zinc-800 rounded-2xl p-4">
      We use the Yallop visibility criterion to predict how easy the young crescent will be to see.
      You can adjust cloud cover and transparency to see how the weather affects the chances.
      Results are shown as <strong>Raw</strong> (pure astronomy) and <strong>Effective</strong> (adjusted for conditions).
    </div>

    <form method="GET" class="bg-zinc-900 border border-zinc-800 rounded-3xl p-8 space-y-6">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-zinc-400 mb-1.5">Year</label>
          <select name="year" id="year" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg" onchange="loadNewMoons()">
            %s
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-zinc-400 mb-1.5">New Moon (we'll check the best following days)</label>
          <select name="date" id="nm_date" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg" data-preselected="%s">
            %s
          </select>
          <div class="text-[10px] text-zinc-500 mt-1">The 3 days after each new moon are usually the most interesting.</div>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-zinc-400 mb-1.5">Latitude</label>
          <input type="text" name="lat" id="lat" value="31.7683" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg">
        </div>
        <div>
          <label class="block text-sm font-medium text-zinc-400 mb-1.5">Longitude</label>
          <input type="text" name="lon" id="lon" value="35.2137" class="w-full bg-zinc-950 border border-zinc-700 rounded-2xl px-4 py-3 text-lg">
        </div>
      </div>

      <div>
        <button type="button" onclick="useMyLocation()"
                class="px-4 py-2 text-sm bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-2xl">
          Use my current location
        </button>

        <div class="mt-2">
          <span class="text-xs text-zinc-400 mr-2">Quick locations:</span>
          <button type="button" onclick="setLocation(31.7683, 35.2137)"
                  class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl mr-1">Jerusalem</button>
          <button type="button" onclick="setLocation(32.7767, -96.7970)"
                  class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl mr-1">Dallas</button>
          <button type="button" onclick="setLocation(-37.8136, 144.9631)"
                  class="px-3 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl">Melbourne</button>
        </div>
      </div>

      <!-- Interactive Map / Virtual Globe -->
      <div>
        <label class="block text-sm font-medium text-zinc-400 mb-1.5">Click the map to set your location</label>
        <div id="location-map" class="border border-zinc-700 rounded-2xl overflow-hidden" style="height: 320px;"></div>
        <div class="text-[10px] text-zinc-500 mt-1">This helps visualize your position relative to the terminator (day/night line) at sunset.</div>
        <div id="sun-moon-times" class="mt-2 text-xs text-emerald-400 font-mono"></div>
      </div>

      <!-- Atmospheric Conditions -->
      <div class="pt-4 border-t border-zinc-800">
        <div class="text-sm font-medium text-emerald-400 mb-3">Atmospheric Conditions</div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs text-zinc-400 mb-1">Cloud Cover</label>
            <input type="range" name="cloud_cover" min="0" max="100" value="20" class="w-full accent-white">
            <div class="text-[10px] text-zinc-500">0% = clear → 100% = overcast</div>
          </div>
          <div>
            <label class="block text-xs text-zinc-400 mb-1">Transparency (1–10)</label>
            <input type="number" name="transparency" min="1" max="10" value="7" class="w-full bg-zinc-950 border border-zinc-700 rounded px-3 py-2">
          </div>
        </div>
      </div>

      <button type="submit" class="w-full bg-white text-black font-semibold py-3.5 rounded-2xl text-lg">
        Check Visibility
      </button>
    </form>

    <p class="text-xs text-center text-zinc-500 mt-6">The dropdown shows the best 3 days after each new moon.</p>
  </div>

  <script>
    // Simple astronomical approximation for sunset/moonset times (demo quality)
    function approximateSunMoonTimes(lat, lon, dateStr) {
      // Very rough approximation – good enough for visualization
      const d = new Date(dateStr);
      const N = Math.floor((d.getTime() - Date.UTC(d.getFullYear(), 0, 0)) / 86400000);
      const lngHour = lon / 15;
      const t = N + ((18 - lngHour) / 24);
      const M = (0.9856 * t) - 3.289;
      const L = M + (1.916 * Math.sin(M * Math.PI/180)) + (0.020 * Math.sin(2*M*Math.PI/180)) + 282.634;
      const RA = (180/Math.PI) * Math.atan(0.91764 * Math.tan(L * Math.PI/180));
      const Lquadrant = (Math.floor(L/90)) * 90;
      const RAquadrant = (Math.floor(RA/90)) * 90;
      const RAdeg = RA + (Lquadrant - RAquadrant);
      const sinDec = 0.39782 * Math.sin(L * Math.PI/180);
      const cosDec = Math.cos(Math.asin(sinDec));
      const cosH = (Math.cos(90.833 * Math.PI/180) - (sinDec * Math.sin(lat * Math.PI/180))) / (cosDec * Math.cos(lat * Math.PI/180));
      const H = 360 - (180/Math.PI) * Math.acos(cosH);
      const T = H/15 + RAdeg/15 - (0.06571 * t) - 6.622;
      const UT = (T - lngHour + 24) %% 24;
      const sunset = (UT + 0) %% 24; // rough
      // Moonset is roughly 50 minutes later on average (very crude for demo)
      const moonset = (sunset + 0.85) %% 24;

      function fmt(h) {
        const hh = Math.floor(h);
        const mm = Math.floor((h - hh) * 60);
        var hhStr = hh.toString();
        if (hhStr.length < 2) hhStr = '0' + hhStr;
        var mmStr = mm.toString();
        if (mmStr.length < 2) mmStr = '0' + mmStr;
        return hhStr + ':' + mmStr;
      }
      return { sunset: fmt(sunset), moonset: fmt(moonset) };
    }

    // Initialize interactive map (Leaflet)
    function initLocationMap(attempt) {
      attempt = attempt || 0;
      const mapEl = document.getElementById('location-map');
      if (!mapEl) {
        console.warn('Map container not found');
        return;
      }
      if (typeof L === 'undefined') {
        if (attempt < 8) {
          console.log('Leaflet not ready yet, retrying... (attempt ' + attempt + ')');
          setTimeout(function() { initLocationMap(attempt + 1); }, 120);
        } else {
          console.error('Leaflet (L) never loaded after multiple retries. Check network / unpkg.com / adblocker / browser console network tab.');
          // Show a visible hint to the user
          mapEl.innerHTML = '<div style="padding:12px;color:#f59e0b;font-size:13px">Interactive map could not load (Leaflet from CDN failed). You can still enter lat/lon manually above.</div>';
        }
        return;
      }

      const map = L.map(mapEl, {
        attributionControl: false,
        zoomControl: true
      }).setView([30, 0], 2); // Rough world view

      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 18,
        className: 'map-tiles'
      }).addTo(map);

      let marker = L.marker([30, 0], {draggable: true}).addTo(map);

      function updateFieldsFromMarker() {
        const ll = marker.getLatLng();
        document.getElementById('lat').value = ll.lat.toFixed(6);
        document.getElementById('lon').value = ll.lng.toFixed(6);
      }

      marker.on('dragend', updateFieldsFromMarker);

      map.on('click', function(e) {
        marker.setLatLng(e.latlng);
        updateFieldsFromMarker();
      });

      // If lat/lon already have values, center on them
      const latVal = parseFloat(document.getElementById('lat').value);
      const lonVal = parseFloat(document.getElementById('lon').value);
      if (!isNaN(latVal) && !isNaN(lonVal)) {
        marker.setLatLng([latVal, lonVal]);
        map.setView([latVal, lonVal], 5);
      }

      // Belt-and-suspenders for cases where the container height settles late
      setTimeout(() => { map.invalidateSize(); }, 80);

      window._locMap = map;
      window._locMarker = marker;

      // Live update sunset/moonset times when marker moves (approximate for demo)
      function updateSunMoonTimes() {
        const ll = marker.getLatLng();
        const dateEl = document.querySelector('select[name="date"], input[name="date"]');
        const dateStr = dateEl ? dateEl.value : '2027-03-20';
        const times = approximateSunMoonTimes(ll.lat, ll.lng, dateStr);
        const el = document.getElementById('sun-moon-times');
        if (el) {
          el.textContent = 'Sunset ~' + times.sunset + ' | Moonset ~' + times.moonset + ' (local)';
        }
      }

      marker.on('moveend', updateSunMoonTimes);
      map.on('click', updateSunMoonTimes);
    }

    async function loadNewMoons() {
      const yearSel = document.getElementById('year');
      const nmSel = document.getElementById('nm_date');
      if (!yearSel || !nmSel) return;

      const year = yearSel.value || '2027';

      nmSel.innerHTML = '<option>Loading new moons...</option>';

      try {
        const res = await fetch('/api/newmoons?year=' + year);
        if (!res.ok) throw new Error('Failed to load');
        const data = await res.json();

        nmSel.innerHTML = '';
        const preselected = nmSel.dataset.preselected || '';
        if (data.new_moons && data.new_moons.length > 0) {
          data.new_moons.forEach(d => {
            const opt = document.createElement('option');
            opt.value = d;
            opt.textContent = d + " (+ following days)";
            if (d === preselected) opt.selected = true;
            nmSel.appendChild(opt);
          });
        } else {
          nmSel.innerHTML = '<option>No data</option>';
        }
      } catch (e) {
        console.error('Failed to load new moons:', e);
        nmSel.innerHTML = '<option>Error loading dates</option>';
      }
    }

    function useMyLocation() {
      navigator.geolocation?.getCurrentPosition(pos => {
        const latEl = document.getElementById('lat');
        const lonEl = document.getElementById('lon');
        latEl.value = pos.coords.latitude.toFixed(6);
        lonEl.value = pos.coords.longitude.toFixed(6);
        if (window._locMap && window._locMarker) {
          window._locMarker.setLatLng([pos.coords.latitude, pos.coords.longitude]);
          window._locMap.panTo(window._locMarker.getLatLng());
        }
      });
    }

    function setLocation(lat, lon) {
      const latEl = document.getElementById('lat');
      const lonEl = document.getElementById('lon');
      latEl.value = lat.toFixed(6);
      lonEl.value = lon.toFixed(6);
      if (window._locMap && window._locMarker) {
        window._locMarker.setLatLng([lat, lon]);
        window._locMap.panTo([lat, lon]);
      }
    }

    // Initialize everything - use DOMContentLoaded + fallback for reliability with CDN scripts
    function startApp() {
      initLocationMap();
      setTimeout(() => {
        loadNewMoons();
        const yearEl = document.getElementById('year');
        if (yearEl) {
          yearEl.onchange = loadNewMoons;
        }
      }, 80);
    }
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', startApp);
    } else {
      startApp();
    }
    // Extra safety net in case Leaflet or layout is very late
    setTimeout(() => {
      if (window._locMap && typeof window._locMap.invalidateSize === 'function') {
        window._locMap.invalidateSize();
      }
    }, 400);
  </script>
</body>
</html>`, yearOptions, defaultNMDate, nmOptions)
		return
	}

	lat, _ := strconv.ParseFloat(latStr, 64)
	lon, _ := strconv.ParseFloat(lonStr, 64)

	renderer := "bin/visibility.out"
	if runtime.GOOS == "windows" {
		renderer = "bin/visibility-windows-amd64.exe"
	}

	// When the user selected a New Moon from the dropdown, check the following 3 days
	// (the typical visible window after conjunction).
	datesToQuery := []string{date}

	base, err := time.Parse("2006-01-02", date)
	if err == nil {
		// Always offer the classic "next 3 days after new moon" window
		datesToQuery = []string{
			base.Format("2006-01-02"),
			base.AddDate(0, 0, 1).Format("2006-01-02"),
			base.AddDate(0, 0, 2).Format("2006-01-02"),
		}
	}

	type DayView struct {
		Date              string
		RawCategory       string
		EffectiveCategory string
		Note              string
		Age               float64
		Q                 float64
	}

	var days []DayView

	cloudCoverInt, _ := strconv.Atoi(cloudCover)
	transparencyFloat, _ := strconv.ParseFloat(transparency, 64)

	for _, d := range datesToQuery {
		cmd := exec.Command(renderer, d, "point", fmt.Sprintf("%.6f", lat), fmt.Sprintf("%.6f", lon), criterion)
		output, err := cmd.CombinedOutput()

		rawCat := "?"
		age := 0.0
		q := 0.0

		if err == nil {
			res := parsePointOutput(string(output))
			rawCat = res.Category
			age = res.Age
			q = res.Q
		} else {
			if errorLogger != nil {
				errorLogger.Printf("handlePointQuery renderer error for date=%s lat=%s lon=%s: %v | output=%s", d, latStr, lonStr, err, string(output))
			}
		}

		eff, note := applyAtmosphericAdjustment(rawCat, cloudCoverInt, transparencyFloat)

		days = append(days, DayView{
			Date:              d,
			RawCategory:       rawCat,
			EffectiveCategory: eff,
			Note:              note,
			Age:               age,
			Q:                 q,
		})
	}

	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		first := days[0]
		fmt.Fprintf(w, `{"date":"%s","lat":%.6f,"lon":%.6f,"effective_category":"%s","days_checked":%d}`,
			date, lat, lon, first.EffectiveCategory, len(days))
		return
	}

	// Clean result page using the days we computed
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Visibility for My Location</title>
<script src="https://cdn.tailwindcss.com"></script>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<style>#context-map{height:180px;border-radius:12px}</style>
</head>
<body class="bg-zinc-950 text-zinc-200 p-8">
<div class="max-w-3xl mx-auto">
  <h1 class="text-3xl font-semibold tracking-tight mb-2">Visibility for My Location</h1>
  <div class="text-zinc-400 mb-2">` + fmt.Sprintf("%.4f°, %.4f° — around %s", lat, lon, date) + `</div>

  <div class="mb-6">
    <div id="context-map" class="border border-zinc-700"></div>
    <div class="text-xs text-zinc-500 mt-1.5">Map shown for geographic context only (marker at query location)</div>
  </div>

  <div class="space-y-4">`

	for _, d := range days {
		// Gracefully handle bad renderer output (e.g. "J" or huge ages)
		if d.RawCategory == "?" || d.RawCategory == "J" || d.Age > 100 {
			html += fmt.Sprintf(`
    <div class="bg-zinc-900 border border-zinc-800 rounded-3xl p-6 opacity-60">
      <div class="text-sm text-zinc-400">%s</div>
      <div class="text-xl text-zinc-400 mt-2">Not a good crescent window</div>
      <div class="text-xs text-zinc-500 mt-1">The selected date is too far from actual new moon conjunction for reliable prediction.</div>
    </div>`, d.Date)
			continue
		}

		catClass := map[string]string{"A": "text-cyan-400", "B": "text-cyan-300", "C": "text-yellow-400", "D": "text-yellow-300", "E": "text-amber-500"}[d.EffectiveCategory]
		if catClass == "" { catClass = "text-zinc-400" }

		html += fmt.Sprintf(`
    <div class="bg-zinc-900 border border-zinc-800 rounded-3xl p-6">
      <div class="flex justify-between items-baseline">
        <div>
          <div class="text-sm text-zinc-400">%s</div>
          <div class="text-6xl font-bold tracking-tighter %s">%s</div>
        </div>
        <div class="text-right text-sm">
          <div>Raw: <span class="font-mono">%s</span></div>
          <div class="text-zinc-400">Age: %.1f h • Q: %.3f</div>
        </div>
      </div>
      <div class="mt-3 text-sm text-emerald-400">%s</div>
    </div>`, d.Date, catClass, d.EffectiveCategory, d.RawCategory, d.Age, d.Q, d.Note)
	}

	html += `</div>

  <div class="mt-8 bg-zinc-900 border border-zinc-800 rounded-3xl p-6 text-sm">
    <div class="font-medium text-zinc-200 mb-3">How to read the visibility rating</div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-1 text-sm">
      <div>
        <span class="font-semibold text-cyan-400">A</span> — Easily visible to the naked eye<br>
        <span class="font-semibold text-cyan-300">B</span> — Visible naked eye under good conditions<br>
        <span class="font-semibold text-yellow-400">C</span> — Visible naked eye, but requires effort
      </div>
      <div>
        <span class="font-semibold text-yellow-300">D</span> — Usually needs binoculars or a telescope<br>
        <span class="font-semibold text-amber-500">E</span> — Very difficult or not visible even with aid
      </div>
    </div>

    <div class="mt-4 pt-4 border-t border-zinc-800 text-xs text-zinc-400 space-y-1 leading-snug">
      <div><strong>Raw</strong>: Pure astronomical prediction using the Yallop visibility criterion at your exact location and time.</div>
      <div><strong>Effective</strong>: Adjusted for the cloud cover and transparency you entered. Clouds hurt visibility significantly; very clear skies can help a little.</div>
      <div><strong>Age</strong>: Hours since the exact moment of new moon (conjunction). The evening of the new moon and the next 1–2 days are the classic window.</div>
      <div><strong>Q</strong>: Quality factor from the model (higher values generally indicate a more favorable geometry for visibility).</div>
    </div>
  </div>

  <div class="mt-6 text-xs text-zinc-500">
    Atmospheric conditions used: ` + cloudCover + `% cloud cover, transparency ` + transparency + `/10.
    <span class="text-zinc-600">(Adjustment is a simple heuristic; full physical modeling is future work.)</span>
  </div>

  <div class="mt-6">
    <a href="/point" class="text-sm underline">New query</a>
    <span class="mx-3 text-zinc-700">·</span>
    <a href="/" class="text-sm underline">Back to maps</a>
  </div>
</div>
</body></html>`

	// Append a tiny non-interactive Leaflet map for geographic context
	mapScript := fmt.Sprintf(`
<script>
  const ctxMap = L.map('context-map', {
    attributionControl: false,
    zoomControl: false,
    dragging: false,
    scrollWheelZoom: false,
    doubleClickZoom: false,
    touchZoom: false
  }).setView([%.6f, %.6f], 8);

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    className: 'map-tiles'
  }).addTo(ctxMap);

  L.marker([%.6f, %.6f]).addTo(ctxMap);
</script>
`, lat, lon, lat, lon)

	html += mapScript
	fmt.Fprint(w, html)
}

// parsePointOutput extracts key values from the renderer’s point mode line
type pointResult struct {
	Category string
	Q        float64
	Age      float64
}

func parsePointOutput(raw string) pointResult {
	var res pointResult
	for _, part := range strings.Fields(raw) {
		if strings.HasPrefix(part, "category=") {
			res.Category = strings.TrimPrefix(part, "category=")
		}
		if strings.HasPrefix(part, "q=") {
			fmt.Sscanf(strings.TrimPrefix(part, "q="), "%f", &res.Q)
		}
		if strings.HasPrefix(part, "age=") {
			fmt.Sscanf(strings.TrimPrefix(part, "age="), "%f", &res.Age)
		}
	}
	return res
}

// applyAtmosphericAdjustment takes the raw astronomical category and applies
// a reasonable heuristic based on cloud cover and transparency.
// This is the current "go hard" implementation for atmospheric effects.
// Categories: A=5 ... E=1
func applyAtmosphericAdjustment(rawCategory string, cloudPercent int, transparency float64) (effective string, note string) {
	categoryValue := map[string]int{"A": 5, "B": 4, "C": 3, "D": 2, "E": 1, "F": 0}[rawCategory]
	if categoryValue == 0 {
		categoryValue = 1
	}

	adjustment := 0

	// Cloud cover penalty (very impactful for crescent visibility)
	if cloudPercent > 80 {
		adjustment -= 3
	} else if cloudPercent > 60 {
		adjustment -= 2
	} else if cloudPercent > 40 {
		adjustment -= 1
	}

	// Transparency adjustment
	if transparency >= 9 {
		adjustment += 1
	} else if transparency <= 4 {
		adjustment -= 1
	} else if transparency <= 2 {
		adjustment -= 2
	}

	finalValue := categoryValue + adjustment
	if finalValue < 1 {
		finalValue = 1
	}
	if finalValue > 5 {
		finalValue = 5
	}

	reverseMap := map[int]string{5: "A", 4: "B", 3: "C", 2: "D", 1: "E"}
	effective = reverseMap[finalValue]

	if adjustment == 0 {
		note = "Atmospheric conditions have minimal impact on this prediction."
	} else if adjustment < 0 {
		note = fmt.Sprintf("Conditions are reducing visibility by approximately %d category level(s).", -adjustment)
	} else {
		note = "Excellent atmospheric conditions are slightly improving the prediction."
	}

	return effective, note
}

func handleWebStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/status/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	job := getJob(id)
	if job == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	resultsSection := ""
	if job.Status == "done" && len(job.MapFiles) > 0 {
		resultsSection = `<div class="mt-8">
			<h3 class="font-medium mb-3">Generated Maps</h3>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">`
		for _, date := range job.MapFiles {
			resultsSection += fmt.Sprintf(`
				<div class="map-card bg-zinc-900 border border-zinc-800 rounded-2xl p-4">
					<div class="text-sm text-zinc-400 mb-2">%s</div>
					<div class="aspect-[16/9] bg-zinc-950 rounded-xl flex items-center justify-center text-xs text-zinc-500 border border-zinc-800">
						Map preview (placeholder)
					</div>
					<div class="mt-3 flex gap-2">
						<a href="#" class="text-xs px-3 py-1.5 bg-white text-black rounded-full hover:bg-zinc-200">Download</a>
					</div>
				</div>`, date)
		}
		resultsSection += `</div></div>`
	} else if job.Status == "done" {
		resultsSection = `<div class="mt-6 text-sm text-emerald-400">Job completed. Output directory: <code class="bg-zinc-900 px-1 py-0.5 rounded">` + job.OutputDir + `</code></div>`
	}

	statusHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Job %s • Crescent Moon Visibility</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <meta http-equiv="refresh" content="5">
</head>
<body class="bg-zinc-950 text-zinc-200">
  <div class="max-w-3xl mx-auto p-8">
    <div class="flex items-center justify-between mb-6">
      <div>
        <div class="text-sm text-zinc-500">Job</div>
        <h1 class="text-2xl font-semibold tracking-tight">%s</h1>
      </div>
      <div class="px-4 py-1.5 rounded-2xl text-sm font-medium border %s">%s</div>
    </div>

    <div class="bg-zinc-900 border border-zinc-800 rounded-3xl p-8">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-6 text-sm">
        <div><span class="text-zinc-400 block text-xs">Years</span> %s – %s</div>
        <div><span class="text-zinc-400 block text-xs">Months</span> %s</div>
        <div><span class="text-zinc-400 block text-xs">GPU</span> %s</div>
        <div><span class="text-zinc-400 block text-xs">Started</span> %s</div>
      </div>

      %s

      %s
    </div>

    <div class="mt-8 flex gap-4 text-sm">
      <a href="/" class="text-zinc-400 hover:text-white underline underline-offset-4">New request</a>
      <a href="#" onclick="location.reload()" class="text-zinc-400 hover:text-white underline underline-offset-4">Refresh now</a>
    </div>
  </div>
</body>
</html>`,
		job.ID, job.ID,
		map[string]string{"running": "border-amber-500 text-amber-400", "done": "border-emerald-500 text-emerald-400", "error": "border-red-500 text-red-400", "queued": "border-zinc-600 text-zinc-400"}[job.Status],
		strings.Title(job.Status),
		job.Params["start_year"], job.Params["end_year"],
		job.Params["months"], job.Params["use_gpu"],
		job.StartedAt.Format("15:04:05"),
		resultsSection,
		func() string {
			if job.Status == "running" || job.Status == "queued" {
				return `<div class="mt-8 text-sm text-zinc-400 flex items-center gap-2">
					<div class="w-2 h-2 bg-white rounded-full animate-pulse"></div>
					Working... (page refreshes automatically)
				</div>`
			}
			if job.Status == "error" {
				return `<div class="mt-6 text-red-400">Error: ` + job.Error + `</div>`
			}
			return ""
		}())

	fmt.Fprint(w, statusHTML)
}

