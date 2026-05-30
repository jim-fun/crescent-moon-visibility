//go:build ignore

// scripts/generate_city_visibility/main.go
//
// Stub / planning skeleton for generating pre-computed visibility data
// for a fixed set of cities over many years.
//
// This program is meant to be run offline (with the full project built)
// to produce the JSON that a minimal WordPress plugin would import.
//
// Usage (once implemented):
//   go run scripts/generate_city_visibility/main.go \
//     --cities "jerusalem,cairo,london" \
//     --start 2006 --end 2030 \
//     --output visibility-data.json

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jim-fun/crescent-moon-visibility/internal/jobspec"
)

// City definition (lat/lon chosen for the pre-computed dataset)
type City struct {
	Slug string
	Name string
	Lat  float64
	Lon  float64
}

var defaultCities = []City{
	{"jerusalem", "Jerusalem", 31.7683, 35.2137},
	{"dallas", "Dallas", 32.7767, -96.7970},
	{"melbourne", "Melbourne", -37.8136, 144.9631},
	{"cairo", "Cairo", 30.0444, 31.2357},
	{"london", "London", 51.5074, -0.1278},
	{"tokyo", "Tokyo", 35.6762, 139.6503},
	{"rio", "Rio de Janeiro", -22.9068, -43.1729},
	{"capetown", "Cape Town", -33.9249, 18.4241},
	{"mumbai", "Mumbai", 19.0760, 72.8777},
	{"istanbul", "Istanbul", 41.0136, 28.9550},
}

func main() {
	citiesFlag := flag.String("cities", "", "Comma-separated city slugs (default: all)")
	startYear := flag.Int("start", 2006, "Start year")
	endYear := flag.Int("end", 2030, "End year (inclusive)")
	output := flag.String("output", "visibility-data.json", "Output JSON file")
	flag.Parse()

	selected := defaultCities
	if *citiesFlag != "" {
		requested := strings.Split(*citiesFlag, ",")
		selected = nil
		for _, slug := range requested {
			slug = strings.TrimSpace(slug)
			found := false
			for _, c := range defaultCities {
				if c.Slug == slug {
					selected = append(selected, c)
					found = true
					break
				}
			}
			if !found {
				log.Fatalf("Unknown city slug: %s", slug)
			}
		}
	}

	fmt.Printf("Generating visibility data for %d cities, years %d-%d...\n",
		len(selected), *startYear, *endYear)

	// Placeholder: In the real implementation we would:
	// 1. For each year, get accurate new moons via jobspec.GetNewMoonsForYear(year)
	// 2. For each new moon + city, run the renderer in "point" mode for 3 days
	//    (requires the visibility binary to be built and discoverable).
	// 3. Record raw categories + compute effective under standard conditions.
	//
	// For now we just emit the structure + a tiny example so the WP side can be prototyped.

	data := map[string]any{
		"meta": map[string]any{
			"generated_at": "2026-05-28",
			"generator":    "crescent-moon-visibility (pre-computed mode)",
			"start_year":   *startYear,
			"end_year":     *endYear,
			"cities":       len(selected),
			"note":         "This is a stub. Real data requires the full renderer.",
		},
		"cities": selected,
		"data":   []any{}, // In real run this would contain many rows
	}

	// Example row (what a real row would look like)
	example := map[string]any{
		"city":          "jerusalem",
		"new_moon":      "2025-03-29",
		"year":          2025,
		"days":          []string{"B", "A", "C"},
		"best_raw":      "A",
		"best_effective": "A",
	}
	_ = example // placeholder

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(*output, b, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Wrote stub data to %s\n", *output)
	fmt.Println("Replace this stub with real calls to the renderer + jobspec for production data.")
}