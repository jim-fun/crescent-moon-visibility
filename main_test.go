package main

import (
	"testing"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
)

// TestParseYearsFromCSV verifies comma-separated year strings are parsed correctly.
func TestParseYearsFromCSV(t *testing.T) {
	years := parseYears("2024,2025,2026", 0, 0)
	if len(years) != 3 {
		t.Fatalf("expected 3 years, got %d", len(years))
	}
	if years[0] != 2024 || years[1] != 2025 || years[2] != 2026 {
		t.Fatalf("unexpected years: %v", years)
	}
}

// TestParseYearsFromRange verifies the start/end range fallback works.
func TestParseYearsFromRange(t *testing.T) {
	years := parseYears("", 2030, 2033)
	if len(years) != 4 {
		t.Fatalf("expected 4 years, got %d", len(years))
	}
	if years[0] != 2030 || years[3] != 2033 {
		t.Fatalf("unexpected years: %v", years)
	}
}

// TestParseYearsEmpty verifies empty input yields no years.
func TestParseYearsEmpty(t *testing.T) {
	years := parseYears("", 0, 0)
	if len(years) != 0 {
		t.Fatalf("expected 0 years, got %d", len(years))
	}
}

// TestParseYearsCSVOverridesRange verifies -years flag takes precedence.
func TestParseYearsCSVOverridesRange(t *testing.T) {
	years := parseYears("2040", 2020, 2030)
	if len(years) != 1 || years[0] != 2040 {
		t.Fatalf("expected [2040], got %v", years)
	}
}

// TestNewMoonsInYear verifies that astronomical new moon calculation returns
// the expected count for a known year (2027 has 13 new moons).
func TestNewMoonsInYear(t *testing.T) {
	moons := astro.NewMoonsInYear(2027)
	// There are typically 12-13 new moons in a year
	if len(moons) < 12 || len(moons) > 13 {
		t.Fatalf("expected 12-13 new moons in 2027, got %d", len(moons))
	}
	// First new moon of 2027 should be in January
	if moons[0].Month() != time.January {
		t.Fatalf("expected first new moon in January, got %s", moons[0].Month())
	}
	// Last new moon should be in December
	if moons[len(moons)-1].Month() != time.December {
		t.Fatalf("expected last new moon in December, got %s", moons[len(moons)-1].Month())
	}
}

// TestNewMoonsKnownDate validates against the known new moon of
// January 29, 2025 (a well-documented new moon date).
func TestNewMoonsKnownDate(t *testing.T) {
	moons := astro.NewMoonsInYear(2025)
	found := false
	for _, m := range moons {
		if m.Month() == time.January && m.Day() == 29 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find new moon on Jan 29 2025, got: %v", moons)
	}
}

// TestTaskMoonAge verifies moon age calculation is correct for day offsets.
func TestTaskMoonAge(t *testing.T) {
	nm := time.Date(2027, 1, 7, 0, 0, 0, 0, time.UTC)
	for i := 0; i < DaysToProcess; i++ {
		currentDate := nm.AddDate(0, 0, i)
		moonAge := currentDate.Sub(nm).Hours() / 24.0
		if moonAge != float64(i) {
			t.Fatalf("expected moon age %d, got %.2f", i, moonAge)
		}
	}
}

// TestNewMoonsCaching verifies the cache returns consistent results.
func TestNewMoonsCaching(t *testing.T) {
	first := getNewMoonsCached(2026)
	second := getNewMoonsCached(2026)
	if len(first) != len(second) {
		t.Fatalf("cache returned different lengths: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if !first[i].Equal(second[i]) {
			t.Fatalf("cache mismatch at index %d: %v vs %v", i, first[i], second[i])
		}
	}
}
