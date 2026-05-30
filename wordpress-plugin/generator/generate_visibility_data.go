//go:build ignore

/*
wordpress-plugin/generator/generate_visibility_data.go

Real Data Generator for the WordPress Pre-computed Visibility Plugin.

This version makes actual calls to bin/visibility.out in point mode
to generate accurate Yallop data.

The plugin's dropdown is built dynamically from whatever cities and years are
present in the generated file, so to change what users see you simply change
what you generate here.

YEARS — controlled by --start / --end (inclusive). The plugin's Year dropdown
lists exactly the years found in the data.

    # 2026 through 2035
    go run generate_visibility_data.go --start 2026 --end 2035 \
        --output ../data/visibility-2026-2035-real.json

CITIES — controlled by --cities (comma-separated slugs). Default = all built-in
cities. The plugin shows every city in the data, with jerusalem, dallas and
melbourne guaranteed as the minimum set, so include at least those three.

    # Only the three required cities
    go run generate_visibility_data.go --start 2026 --end 2035 \
        --cities jerusalem,dallas,melbourne \
        --output ../data/visibility-2026-2035-real.json

    # The three required plus a few extras
    go run generate_visibility_data.go --start 2026 --end 2035 \
        --cities jerusalem,dallas,melbourne,london,tokyo,cairo \
        --output ../data/visibility-2026-2035-real.json

To add a NEW city, add a {slug, name, lat, lon} entry to the `cities` list
below, then reference its slug in --cities. After generating, import the file
under Tools → Crescent Visibility and the new city appears in the dropdown.
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/jobspec"
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
	Days          []string  `json:"days"`     // per-evening category
	DayQ          []float64 `json:"day_q"`    // per-evening Q value
	DayAge        []float64 `json:"day_age"`  // per-evening moon age (hours)
	BestRaw       string    `json:"best_raw"`
	BestEffective string    `json:"best_effective"`
	QAtBest       float64   `json:"q_at_best,omitempty"`
	MoonAgeAtBest float64   `json:"moon_age_at_best,omitempty"`
}

func roundN(f float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(f*p) / p
}

type YearlySummary struct {
	City                     string `json:"city"`
	Year                     int    `json:"year"`
	NewMoonsCount            int    `json:"new_moons_count"`
	BestEffective            string `json:"best_effective"`
	WindowsWithA             int    `json:"windows_with_a"`
	WindowsWithBOrBetter     int    `json:"windows_with_b_or_better"`
	WindowsWithCOrBetter     int    `json:"windows_with_c_or_better"`
}

type Output struct {
	Meta struct {
		GeneratedAt string `json:"generated_at"`
		Generator   string `json:"generator"`
		StartYear   int    `json:"start_year"`
		EndYear     int    `json:"end_year"`
		Cities      int    `json:"cities"`
		Note        string `json:"note"`
	} `json:"meta"`
	Cities          []City            `json:"cities"`
	Observations    []Observation     `json:"observations"`
	YearlySummaries []YearlySummary   `json:"yearly_summaries"`
}

// Deduped representative cities (~800-mile / 400-mi-radius clustering of the
// observer location list). Required cities first, then one representative per
// region; nearby observers fold into the nearest representative.
var cities = []City{
	// Required
	{"jerusalem", "Israel — Jerusalem", 31.7683, 35.2137},   // Levant / Israel
	{"dallas", "USA — Dallas", 32.7767, -96.7970},         // TX, OK, AR, N-LA, S-KS
	{"melbourne", "Australia — Melbourne", -37.8136, 144.9631},  // Victoria
	// North America
	{"seattle", "USA — Seattle", 47.6062, -122.3321},      // WA, OR, Vancouver BC
	{"princegeorge", "Canada — Prince George", 53.9171, -122.7497}, // central/north BC
	{"losangeles", "USA — Los Angeles", 34.0522, -118.2437},     // all California
	{"phoenix", "USA — Phoenix", 33.4484, -112.0740},      // AZ, S-NV
	{"denver", "USA — Denver", 39.7392, -104.9903},        // CO, NM
	{"kansascity", "USA — Kansas City", 39.0997, -94.5786},// MO, KS, IA, NE, SD, MN
	{"chicago", "USA — Chicago", 41.8781, -87.6298},       // IL, WI, IN, MI, OH, KY
	{"atlanta", "USA — Atlanta", 33.7490, -84.3880},       // GA, TN, MS, AL, SC, NC, N-FL
	{"orlando", "USA — Orlando", 28.5383, -81.3792},       // FL peninsula, Grand Bahama
	{"washington", "USA — Washington, DC", 38.9072, -77.0369}, // MD, VA, WV, PA, NJ, NY, Ontario
	{"boston", "USA — Boston", 42.3601, -71.0589},         // New England (MA, RI)
	{"regina", "Canada — Regina", 50.4452, -104.6189},        // Saskatchewan
	// Caribbean / Latin America
	{"sanjuan", "Puerto Rico — San Juan", 18.4655, -66.1057},      // Puerto Rico, USVI, Antigua
	{"portofspain", "Trinidad & Tobago — Port of Spain", 10.6549, -61.5019},
	{"montegobay", "Jamaica — Montego Bay", 18.4762, -77.8939},// Jamaica
	{"mexicocity", "Mexico — Mexico City", 19.4326, -99.1332},// central Mexico
	{"merida", "Mexico — Mérida", 20.9674, -89.5926},         // Yucatán, Chiapas
	{"panamacity", "Panama — Panama City", 8.9824, -79.5199},
	{"pasto", "Colombia — Pasto", 1.2136, -77.2811},            // S Colombia
	// Europe / Africa / Middle East
	{"london", "United Kingdom — London", 51.5074, -0.1278},
	{"fethiye", "Turkey — Fethiye", 36.6213, 29.1162},        // SW Turkey
	{"johannesburg", "South Africa — Johannesburg", -26.2041, 28.0473}, // Gauteng + KZN
	{"gobabis", "Namibia — Gobabis", -22.4500, 18.9667},       // Namibia
	{"dodoma", "Tanzania — Dodoma", -6.1630, 35.7516},          // Tanzania
	// Asia-Pacific
	{"honolulu", "USA — Honolulu", 21.3069, -157.8583},    // Hawaii
	{"adelaide", "Australia — Adelaide", -34.9285, 138.6007},    // S Australia
	{"perth", "Australia — Perth", -31.9523, 115.8613},          // W Australia
}

// requiredSlugs are the cities the WordPress plugin always offers in its
// dropdown. Generating data for them keeps the imported dataset aligned with
// the plugin's minimum set (the plugin still lists them even if absent, but
// then they'd show "no data").
var requiredSlugs = []string{"jerusalem", "dallas", "melbourne"}

// selectCities filters the built-in list down to the requested comma-separated
// slugs (order follows the request). Unknown slugs are reported and skipped.
func selectCities(all []City, csv string) []City {
	bySlug := make(map[string]City, len(all))
	for _, c := range all {
		bySlug[c.Slug] = c
	}
	var out []City
	seen := make(map[string]bool)
	for _, raw := range strings.Split(csv, ",") {
		slug := strings.ToLower(strings.TrimSpace(raw))
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		if c, ok := bySlug[slug]; ok {
			out = append(out, c)
		} else {
			log.Printf("WARNING: unknown city slug %q — add it to the cities list in this generator first", slug)
		}
	}
	for _, req := range requiredSlugs {
		if !seen[req] {
			log.Printf("NOTE: required city %q is not in --cities; the plugin will list it but show \"no data\".", req)
		}
	}
	return out
}

func main() {
	startYear := flag.Int("start", 2026, "First year to generate (inclusive)")
	endYear := flag.Int("end", 2028, "Last year to generate (inclusive)")
	citiesFlag := flag.String("cities", "", "Comma-separated city slugs to generate (default: all built-in cities). e.g. jerusalem,dallas,melbourne")
	outputPath := flag.String("output", "../data/visibility-2026-2028-real.json", "Output file")
	flag.Parse()

	if *citiesFlag != "" {
		cities = selectCities(cities, *citiesFlag)
	}
	if len(cities) == 0 {
		log.Fatal("No cities selected. Check your --cities slugs against the cities list in this generator.")
	}
	if *endYear < *startYear {
		log.Fatalf("--end (%d) must be >= --start (%d)", *endYear, *startYear)
	}

	fmt.Printf("Generating REAL visibility data for %d cities, %d-%d...\n", len(cities), *startYear, *endYear)

	var observations []Observation
	var summaries []YearlySummary

	for year := *startYear; year <= *endYear; year++ {
		newMoons := jobspec.GetNewMoonsForYear(year)
		fmt.Printf("  Year %d: %d new moons\n", year, len(newMoons))

		for _, city := range cities {
			yearBest := "E"
			countA, countB, countC := 0, 0, 0

			for _, nmDate := range newMoons {
				// Get the 3 days after new moon
				day0 := nmDate
				day1 := addDays(nmDate, 1)
				day2 := addDays(nmDate, 2)

				cats := make([]string, 3)
				dayQ := make([]float64, 3)
				dayAge := make([]float64, 3)
				bestQ := 0.0
				bestAge := 0.0
				bestCat := "E"

				for i, d := range []string{day0, day1, day2} {
					cat, q, age, err := runPointQuery(d, city.Latitude, city.Longitude, "yallop")
					if err != nil {
						log.Printf("Error for %s %s: %v", city.Slug, d, err)
						cat = "?"
					}
					cats[i] = cat
					dayQ[i] = roundN(q, 4)
					dayAge[i] = roundN(age, 2)

					if cat < bestCat {
						bestCat = cat
						bestQ = q
						bestAge = age
					}
				}

				bestEff := bestCat // Assuming clear skies for pre-computed data

				obs := Observation{
					City:          city.Slug,
					NewMoon:       nmDate,
					Year:          year,
					Days:          cats,
					DayQ:          dayQ,
					DayAge:        dayAge,
					BestRaw:       bestCat,
					BestEffective: bestEff,
					QAtBest:       roundN(bestQ, 4),
					MoonAgeAtBest: roundN(bestAge, 2),
				}
				observations = append(observations, obs)

				// Update yearly stats
				if bestEff == "A" {
					countA++
				}
				if bestEff <= "B" {
					countB++
				}
				if bestEff <= "C" {
					countC++
				}
				if bestEff < yearBest {
					yearBest = bestEff
				}
			}

			summaries = append(summaries, YearlySummary{
				City:                 city.Slug,
				Year:                 year,
				NewMoonsCount:        len(newMoons),
				BestEffective:        yearBest,
				WindowsWithA:         countA,
				WindowsWithBOrBetter: countB,
				WindowsWithCOrBetter: countC,
			})
		}
	}

	out := Output{}
	out.Meta.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	out.Meta.Generator = "crescent-moon-visibility real renderer (Yallop)"
	out.Meta.StartYear = *startYear
	out.Meta.EndYear = *endYear
	out.Meta.Cities = len(cities)
	out.Meta.Note = "Generated with real visibility.out point queries. Standard clear skies assumption for effective categories."

	out.Cities = cities
	out.Observations = observations
	out.YearlySummaries = summaries

	dir := filepath.Dir(*outputPath)
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	os.WriteFile(*outputPath, data, 0644)

	fmt.Printf("\nDone! Wrote %d observations to %s\n", len(observations), *outputPath)
}

func addDays(dateStr string, days int) string {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func runPointQuery(date string, lat, lon float64, criterion string) (category string, q, age float64, err error) {
	cmd := exec.Command("bin/visibility.out", date, "point", fmt.Sprintf("%.6f", lat), fmt.Sprintf("%.6f", lon), criterion)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, 0, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		for _, p := range parts {
			if strings.HasPrefix(p, "category=") {
				category = strings.TrimPrefix(p, "category=")
			}
			if strings.HasPrefix(p, "q=") {
				q, _ = strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64)
			}
			if strings.HasPrefix(p, "age=") {
				age, _ = strconv.ParseFloat(strings.TrimPrefix(p, "age="), 64)
			}
		}
	}
	return category, q, age, nil
}
