package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jim-fun/crescent-moon-visibility/internal/validation/icop"
)

func main() {
	dataset := flag.String("dataset", "data/validation/icop/sightings.json", "Path to curated sightings JSON")
	report := flag.String("report", "", "Optional machine-readable report: json")
	flag.Parse()

	sightings, err := icop.LoadSightings(*dataset)
	if err != nil {
		log.Fatalf("Failed to load dataset: %v", err)
	}

	fmt.Printf("ICOP External Validation Harness\n")
	fmt.Printf("Loaded %d sightings from %s\n\n", len(sightings), *dataset)

	// First pass: validate and collect only valid sightings (PR 2 robustness)
	var validSightings []icop.Sighting
	for _, s := range sightings {
		if err := s.Validate(); err != nil {
			fmt.Printf("  INVALID: %s - %v\n", s.ID, err)
			continue
		}
		validSightings = append(validSightings, s)
	}

	fmt.Printf("%d valid sightings loaded.\n\n", len(validSightings))

	useRenderer := true // default to exact renderer for 100% fidelity (PR 2)
	rendererPath := "./bin/visibility.out"

	if useRenderer {
		fmt.Println("Using EXACT Yallop calculation from CPU renderer (point mode)")
	} else {
		fmt.Println("Using approximation")
	}
	fmt.Println("-------------------------------------------------------------")

	var results []icop.Result
	matches := 0
	scored := 0
	var totalAge float64
	var nakedMatches, nakedTotal, aidedMatches, aidedTotal int

	for _, s := range validSightings {
		var cat string
		var q float64
		var parseErr error

		var age float64
		if useRenderer {
			// Call the real renderer for 100% fidelity (existing pattern)
			out, err := exec.Command(rendererPath, s.Date, "point", fmt.Sprintf("%.4f", s.Latitude), fmt.Sprintf("%.4f", s.Longitude), "yallop").Output()
			if err != nil {
				fmt.Printf("%s: renderer error - %v\n", s.ID, err)
				continue
			}
			line := string(out)

			// Robust parsing with guards (PR 2 improvement)
			if idx := strings.LastIndex(line, "category="); idx != -1 {
				cat = string(line[idx+9])
			}
			if idx := strings.LastIndex(line, "q="); idx != -1 {
				end := strings.IndexAny(line[idx+2:], " \n")
				if end == -1 {
					end = len(line) - idx - 2
				}
				q, parseErr = strconv.ParseFloat(line[idx+2:idx+2+end], 64)
				if parseErr != nil {
					fmt.Printf("%s: warning - could not parse q value\n", s.ID)
				}
			}
			// Parse exact moon age emitted by renderer (identical to the one used for this category/q decision)
			if idx := strings.LastIndex(line, "age="); idx != -1 {
				end := strings.IndexAny(line[idx+4:], " \n")
				if end == -1 {
					end = len(line) - idx - 4
				}
				age, _ = strconv.ParseFloat(line[idx+4:idx+4+end], 64)
			}

			// Guard: require valid category A-J (covers pre-conjunction G/H/I/J too)
			if cat == "" || cat[0] < 'A' || cat[0] > 'J' {
				fmt.Printf("%s: failed to parse valid category from renderer output\n", s.ID)
				continue
			}
		} else {
			cat, q, err = icop.ApproximateYallopCategory(s)
			if err != nil {
				fmt.Printf("%s: error - %v\n", s.ID, err)
				continue
			}
			// No exact age available from approx path; leave age=0 (alignNote omitted)
		}

		// Instrument-aware match (PR 2 core: A/B for naked seen; C/D/E allowed for aided)
		match, reason := icop.InstrumentAwareMatch(s, cat)

		// Alignment diagnostic: prefer exact age from renderer (high fidelity); otherwise omit
		alignNote := ""
		if age > 0.1 {
			alignNote = fmt.Sprintf(" age=%.1fh", age)
		}

		fmt.Printf("%s  pred=%s (q=%.3f)  reported=%s  instr=%s  match=%v  (%s)%s\n",
			s.ID, cat, q, s.ReportedResult, s.Instrument, match, reason, alignNote)

		// Collect structured result for summary/JSON (PR 2)
		results = append(results, icop.Result{
			SightingID: s.ID,
			Predicted:  cat,
			Reported:   s.ReportedResult,
			Match:      match,
			Notes:      reason + alignNote,
		})

		if match {
			matches++
		}
		scored++
		totalAge += age

		if s.Instrument == "naked_eye" {
			nakedTotal++
			if match {
				nakedMatches++
			}
		} else {
			aidedTotal++
			if match {
				aidedMatches++
			}
		}
	}

	rate := 0.0
	if scored > 0 {
		rate = float64(matches) * 100.0 / float64(scored)
	}

	meanAge := 0.0
	if scored > 0 {
		meanAge = totalAge / float64(scored)
	}

	fmt.Printf("\nMatch rate: %.1f%% (%d/%d scored)\n", rate, matches, scored)
	fmt.Printf("Scored records: %d (validated + parse + alignment successful)\n", scored)
	if nakedTotal > 0 {
		fmt.Printf("  naked_eye: %.1f%% (%d/%d)\n", float64(nakedMatches)*100.0/float64(nakedTotal), nakedMatches, nakedTotal)
	}
	if aidedTotal > 0 {
		fmt.Printf("  aided (binoculars/telescope): %.1f%% (%d/%d)\n", float64(aidedMatches)*100.0/float64(aidedTotal), aidedMatches, aidedTotal)
	}
	if meanAge > 0 {
		fmt.Printf("  mean moon age (exact, from renderer best-time moon_age_prev): %.1f h\n", meanAge)
	}

	// Emit richer Summary (PR 2 foundation for later PR3/PR4)
	summary := icop.Summary{
		Total:      len(validSightings),
		Matches:    matches,
		Mismatches: scored - matches,
		MatchRate:  rate,
		ByCategory: map[string]int{
			"total_scored": scored,
			"naked_total":  nakedTotal,
			"aided_total":  aidedTotal,
		},
	}

	if len(results) > 0 {
		fmt.Printf("\nTotal records processed for summary: %d\n", len(results))
	}

	if useRenderer {
		fmt.Println("\nResults produced using the exact Yallop implementation from the CPU reference renderer (point mode).")
		fmt.Println("Instrument-aware matching (A/B naked vs A-E aided) + exact renderer moon age alignment enabled (PR 2).")
	}

	// Optional JSON report (additive, per design)
	if *report == "json" {
		reportData := map[string]interface{}{
			"summary": summary,
			"results": results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fmt.Println("\n--- JSON REPORT ---")
		if err := enc.Encode(reportData); err != nil {
			log.Printf("JSON encode error: %v", err)
		}
	}
}
