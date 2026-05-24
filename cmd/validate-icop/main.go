package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jim-fun/crescent-moon-visibility/internal/validation/icop"
)

func main() {
	dataset := flag.String("dataset", "data/validation/icop/sightings.json", "Path to curated sightings JSON")
	flag.Parse()

	sightings, err := icop.LoadSightings(*dataset)
	if err != nil {
		log.Fatalf("Failed to load dataset: %v", err)
	}

	fmt.Printf("ICOP External Validation Harness\n")
	fmt.Printf("Loaded %d sightings from %s\n\n", len(sightings), *dataset)

	validCount := 0
	for _, s := range sightings {
		if err := s.Validate(); err != nil {
			fmt.Printf("  INVALID: %s - %v\n", s.ID, err)
			continue
		}
		validCount++
	}

	fmt.Printf("%d valid sightings loaded.\n\n", validCount)

	useRenderer := true // we now default to exact renderer
	rendererPath := "./bin/visibility.out"

	if useRenderer {
		fmt.Println("Using EXACT Yallop calculation from CPU renderer (point mode)")
	} else {
		fmt.Println("Using approximation")
	}
	fmt.Println("-------------------------------------------------------------")

	matches := 0
	for _, s := range sightings {
		var cat string
		var q float64

		if useRenderer {
			// Call the real renderer for 100% fidelity
			out, err := exec.Command(rendererPath, s.Date, "point", fmt.Sprintf("%.4f", s.Latitude), fmt.Sprintf("%.4f", s.Longitude), "yallop").Output()
			if err != nil {
				fmt.Printf("%s: renderer error - %v\n", s.ID, err)
				continue
			}
			line := string(out)
			if idx := strings.LastIndex(line, "category="); idx != -1 {
				cat = string(line[idx+9])
			}
			if idx := strings.LastIndex(line, "q="); idx != -1 {
				end := strings.IndexAny(line[idx+2:], " \n")
				if end == -1 {
					end = len(line) - idx - 2
				}
				q, _ = strconv.ParseFloat(line[idx+2:idx+2+end], 64)
			}
		} else {
			cat, q, err = icop.ApproximateYallopCategory(s)
			if err != nil {
				fmt.Printf("%s: error - %v\n", s.ID, err)
				continue
			}
		}

		reported := "F"
		if s.ReportedResult == "seen" {
			reported = "A"
		}

		match := false
		if (cat == "A" || cat == "B") && s.ReportedResult == "seen" {
			match = true
		}
		if (cat == "F" || cat == "E") && s.ReportedResult == "not_seen" {
			match = true
		}

		if match {
			matches++
		}

		fmt.Printf("%s  pred=%s (q=%.3f)  reported=%s  match=%v\n",
			s.ID, cat, q, reported, match)
	}

	rate := 0.0
	if len(sightings) > 0 {
		rate = float64(matches) * 100.0 / float64(len(sightings))
	}

	fmt.Printf("\nMatch rate: %.1f%% (%d/%d)\n", rate, matches, len(sightings))
	if useRenderer {
		fmt.Println("Results produced using the exact Yallop implementation from the CPU reference renderer.")
	}
}