//go:build web

package astro

import "time"

// Stub implementations for web-only builds.
// These allow `go run -tags=web . web` to succeed even when the full
// CGO astronomy bindings have environment issues.

func MoonIlluminationFraction(t time.Time) float64 {
	return 0.1
}

// NewMoonsInYear returns a reasonable list of new moon dates for the year.
// This is used by the web UI's "My Location" feature to populate the
// new moon dropdown when the user selects a year.
//
// For the web build we use a simple but decent lunar cycle approximation
// so the dropdown actually contains real-ish new moon dates.
func NewMoonsInYear(year int) []time.Time {
	return approximateNewMoonsForYear(year)
}

func GetMoonAgeHours(latitude, longitude float64, t time.Time) float64 {
	return 24.0
}

// approximateNewMoonsForYear uses a mean synodic month (~29.530588853 days)
// starting from a known recent new moon. This produces dates that are
// usually within ~1 day of the real new moons — good enough for the web UI demo.
func approximateNewMoonsForYear(year int) []time.Time {
	// A known new moon close to 2000 (for easy calculation)
	// 2000-01-06 18:14 UTC was a real new moon. We use noon for simplicity.
	epoch := time.Date(2000, 1, 6, 12, 0, 0, 0, time.UTC)

	synodicMonth := time.Duration(29.530588853 * float64(24*time.Hour))

	var moons []time.Time

	// Go backwards to before the target year
	t := epoch
	for t.Year() > year-1 {
		t = t.Add(-synodicMonth)
	}

	// Now step forward and collect all new moons in the target year
	for {
		if t.Year() > year {
			break
		}
		if t.Year() == year {
			moons = append(moons, t)
		}
		t = t.Add(synodicMonth)
	}

	return moons
}
