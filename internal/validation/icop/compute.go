package icop

import (
	"fmt"
)

// ApproximateYallopCategory is a fallback approximation path (non-default).
// In the current build the exact astro dependency for illumination is not
// linked into the validation cmd (CGO/astro package build issue in this env);
// the recommended and default path for PR2 is the exact CPU renderer "point"
// mode which provides 100% fidelity + exact moon age. This stub returns an
// error directing the user to the renderer path.
func ApproximateYallopCategory(s Sighting) (category string, q float64, err error) {
	return "", 0, fmt.Errorf("approximate path disabled in this build (use default exact renderer point mode for PR2 ICOP validation; astro CGO resolution pending separate fix)")
}

// InstrumentAwareMatch implements the instrument-aware matching logic required for PR2.
// It respects the distinction between naked-eye (A/B) and aided (C/D/E) visibility
// per the Yallop categories and the ICOP "instrument" field.
//
// Rules (documented for transparency and future PR3/PR4 use):
//   - naked_eye + "seen"     → only A or B count as match
//   - naked_eye + "not_seen" → C/D/E/F count as match
//   - binoculars/telescope + "seen"     → A-E count as match (not F)
//   - binoculars/telescope + "not_seen" → only F counts as match
//
// Returns (match, reason) for clear diagnostics in the harness output.
func InstrumentAwareMatch(s Sighting, predictedCat string) (bool, string) {
	if len(predictedCat) == 0 {
		return false, "no predicted category"
	}

	isAided := s.Instrument == "binoculars" || s.Instrument == "telescope"
	isSeen := s.ReportedResult == "seen"

	switch {
	case isSeen && !isAided:
		// Naked eye "seen" — only true naked-eye visible categories count
		if predictedCat == "A" || predictedCat == "B" {
			return true, "naked_eye seen matches A/B"
		}
		return false, "naked_eye seen but predicted " + predictedCat + " (needs optical aid)"

	case isSeen && isAided:
		// Aided "seen" — A-E are acceptable
		if predictedCat >= "A" && predictedCat <= "E" {
			return true, "aided seen matches A-E"
		}
		return false, "aided seen but predicted " + predictedCat

	case !isSeen && !isAided:
		// Naked eye "not_seen" — C+ are acceptable
		if predictedCat >= "C" {
			return true, "naked_eye not_seen matches C+"
		}
		return false, "naked_eye not_seen but predicted " + predictedCat + " (would have been visible naked-eye)"

	case !isSeen && isAided:
		// Aided "not_seen" — only F is acceptable
		if predictedCat == "F" {
			return true, "aided not_seen matches F"
		}
		return false, "aided not_seen but predicted " + predictedCat + " (would have been visible with aid)"
	}

	return false, "unhandled instrument/reported combination"
}
