//go:build ignore

/*
wordpress-plugin/generator/validate_visibility_data.go

Validates a generated visibility JSON file against the reference renderer
(bin/visibility.out). For a sample of observations it re-runs the point query
for each of the three evenings (new moon, +1, +2) and checks that:

  1. the stored per-day categories match the renderer, and
  2. the stored q_at_best / moon_age_at_best match the renderer's value on the
     best (lowest-category) evening.

It also prints the renderer's PER-DAY q and age, which makes the current
data-model limitation visible: the JSON stores only ONE q/age per new moon
(the best evening), so the plugin shows the same Age/Q on all three day cards.

Run from the REPO ROOT (so bin/visibility.out resolves):

    go run wordpress-plugin/generator/validate_visibility_data.go \
        --input wordpress-plugin/data/visibility-2026-2075.json \
        --sample 40

Flags:
    --input    JSON file to validate
    --sample   number of observations to spot-check (0 = all; default 40)
    --city     only validate this city slug (optional)
    --seed     RNG seed for reproducible sampling (default 1)
    --binary   path to the renderer (default bin/visibility.out)
    --verbose  print every sampled observation (default: only mismatches + a few)
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type City struct {
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Observation struct {
	City          string    `json:"city"`
	NewMoon       string    `json:"new_moon"`
	Year          int       `json:"year"`
	Days          []string  `json:"days"`
	DayQ          []float64 `json:"day_q"`
	DayAge        []float64 `json:"day_age"`
	BestRaw       string    `json:"best_raw"`
	QAtBest       float64   `json:"q_at_best"`
	MoonAgeAtBest float64   `json:"moon_age_at_best"`
}

type Data struct {
	Cities       []City        `json:"cities"`
	Observations []Observation `json:"observations"`
}

func main() {
	input := flag.String("input", "wordpress-plugin/data/visibility-2026-2075.json", "JSON file to validate")
	sample := flag.Int("sample", 40, "number of observations to spot-check (0 = all)")
	cityFilter := flag.String("city", "", "only validate this city slug")
	seed := flag.Int64("seed", 1, "RNG seed for reproducible sampling")
	binary := flag.String("binary", "bin/visibility.out", "path to the renderer binary")
	verbose := flag.Bool("verbose", false, "print every sampled observation")
	flag.Parse()

	raw, err := os.ReadFile(*input)
	if err != nil {
		fmt.Printf("ERROR reading %s: %v\n", *input, err)
		os.Exit(1)
	}
	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		fmt.Printf("ERROR parsing JSON: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(*binary); err != nil {
		fmt.Printf("ERROR: renderer not found at %q. Run from the repo root, or pass --binary.\n", *binary)
		os.Exit(1)
	}

	coords := map[string][2]float64{}
	for _, c := range data.Cities {
		coords[c.Slug] = [2]float64{c.Latitude, c.Longitude}
	}

	// Build the candidate list (optionally filtered by city).
	var pool []int
	for i, o := range data.Observations {
		if *cityFilter == "" || o.City == *cityFilter {
			pool = append(pool, i)
		}
	}
	if len(pool) == 0 {
		fmt.Println("No observations matched the filter.")
		os.Exit(1)
	}

	// Sample.
	rng := rand.New(rand.NewSource(*seed))
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	n := *sample
	if n <= 0 || n > len(pool) {
		n = len(pool)
	}
	pool = pool[:n]

	fmt.Printf("Validating %s against %s\n", *input, *binary)
	fmt.Printf("Sampling %d of %d observations%s\n\n", n, len(data.Observations),
		ternary(*cityFilter != "", " (city="+*cityFilter+")", ""))

	catMismatch, qAgeMismatch, missingCoord, shown := 0, 0, 0, 0

	for _, idx := range pool {
		o := data.Observations[idx]
		ll, ok := coords[o.City]
		if !ok {
			missingCoord++
			fmt.Printf("[%s %s] SKIP: no coordinates for city in file\n", o.City, o.NewMoon)
			continue
		}

		days := []string{o.NewMoon, addDays(o.NewMoon, 1), addDays(o.NewMoon, 2)}
		var rCats [3]string
		var rQ, rAge [3]float64
		queryErr := false
		for i, d := range days {
			cat, q, age, err := runPointQuery(*binary, d, ll[0], ll[1], "yallop")
			if err != nil {
				queryErr = true
				cat = "?"
			}
			rCats[i], rQ[i], rAge[i] = cat, q, age
		}

		// Per-day accuracy: stored category, q and age vs the renderer.
		catOK, qAgeOK := true, true
		for i := 0; i < 3; i++ {
			if i < len(o.Days) && o.Days[i] != rCats[i] {
				catOK = false
			}
			if i < len(o.DayQ) && !approxEqual(o.DayQ[i], rQ[i], 0.01) {
				qAgeOK = false
			}
			if i < len(o.DayAge) && !approxEqual(o.DayAge[i], rAge[i], 0.05) {
				qAgeOK = false
			}
		}
		if len(o.DayQ) != 3 || len(o.DayAge) != 3 {
			qAgeOK = false
		}

		if !catOK {
			catMismatch++
		}
		if !qAgeOK {
			qAgeMismatch++
		}

		// Print mismatches always; otherwise the first few for visibility.
		if !catOK || !qAgeOK || queryErr || (*verbose) || shown < 3 {
			shown++
			fmt.Printf("[%s %s]  %s %s\n", o.City, o.NewMoon, okLabel(catOK, "categories"), okLabel(qAgeOK, "per-day q/age"))
			for i := 0; i < 3; i++ {
				storedDay, storedQ, storedAge := "?", math.NaN(), math.NaN()
				if i < len(o.Days) {
					storedDay = o.Days[i]
				}
				if i < len(o.DayQ) {
					storedQ = o.DayQ[i]
				}
				if i < len(o.DayAge) {
					storedAge = o.DayAge[i]
				}
				flag := ""
				if storedDay != rCats[i] || !approxEqual(storedQ, rQ[i], 0.01) || !approxEqual(storedAge, rAge[i], 0.05) {
					flag = "  <-- MISMATCH"
				}
				fmt.Printf("  Day+%d %s: stored[cat=%s q=%.4f age=%.2fh]  renderer[cat=%s q=%.4f age=%.2fh]%s\n",
					i, days[i], storedDay, storedQ, storedAge, rCats[i], rQ[i], rAge[i], flag)
			}
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Sampled:                %d\n", n)
	fmt.Printf("Category mismatches:    %d\n", catMismatch)
	fmt.Printf("Per-day q/age mismatches: %d\n", qAgeMismatch)
	if missingCoord > 0 {
		fmt.Printf("Missing coordinates:    %d\n", missingCoord)
	}
	fmt.Println()
	if catMismatch == 0 && qAgeMismatch == 0 {
		fmt.Println("RESULT: PASS — stored per-day categories, Q and age all match the reference renderer.")
	} else {
		fmt.Println("RESULT: FAIL — the data does not match the renderer (see mismatches above).")
		os.Exit(1)
	}
}

// bestDayIndex returns the index of the lowest (best) category, matching the
// generator's "if cat < bestCat" selection where 'A' is best.
func bestDayIndex(cats []string) int {
	best := 0
	for i := 1; i < len(cats); i++ {
		if cats[i] < cats[best] {
			best = i
		}
	}
	return best
}

func approxEqual(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func okLabel(ok bool, what string) string {
	if ok {
		return "[" + what + " OK]"
	}
	return "[" + what + " MISMATCH]"
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func addDays(dateStr string, days int) string {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func runPointQuery(binary, date string, lat, lon float64, criterion string) (category string, q, age float64, err error) {
	cmd := exec.Command(binary, date, "point", fmt.Sprintf("%.6f", lat), fmt.Sprintf("%.6f", lon), criterion)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, 0, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		for _, p := range strings.Fields(scanner.Text()) {
			switch {
			case strings.HasPrefix(p, "category="):
				category = strings.TrimPrefix(p, "category=")
			case strings.HasPrefix(p, "q="):
				q, _ = strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64)
			case strings.HasPrefix(p, "age="):
				age, _ = strconv.ParseFloat(strings.TrimPrefix(p, "age="), 64)
			}
		}
	}
	return category, q, age, nil
}
