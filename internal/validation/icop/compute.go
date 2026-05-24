package icop

import (
	"math"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/astro"
)

// ApproximateYallopCategory computes a visibility category using the exact
// Yallop (1997) cubic formula with a simplified but reasonable model for ARCV
// and crescent width based on moon illumination and latitude.
//
// This is an early but improved approximation for the validation harness.
// The long-term goal is to drive the exact CPU renderer ("point" mode) or
// share the precise calculation from visibility.cc for 100% fidelity.
func ApproximateYallopCategory(s Sighting) (category string, q float64, err error) {
	t, err := time.Parse("2006-01-02", s.Date)
	if err != nil {
		return "F", 0, err
	}

	illum := astro.MoonIlluminationFraction(t)

	// Model tuned to produce plausible variation on young crescents
	// (this is still an approximation — real fidelity will come from the renderer)
	elongation := 8.5 + (1.0 - illum) * 9.0

	latFactor := 0.9 + (math.Abs(s.Latitude) / 90.0 * 0.35)
	arcv := 5.0 + elongation * latFactor * 0.65

	w := 9.8 * (1 - math.Cos(elongation*math.Pi/180.0))

	// Exact Yallop 1997 cubic
	threshold := 11.8371 - 6.3226*w + 0.7319*w*w - 0.1018*w*w*w
	q = (arcv - threshold) / 10.0

	switch {
	case q > 0.216:
		category = "A"
	case q > -0.014:
		category = "B"
	case q > -0.160:
		category = "C"
	case q > -0.232:
		category = "D"
	case q > -0.293:
		category = "E"
	default:
		category = "F"
	}

	return category, q, nil
}