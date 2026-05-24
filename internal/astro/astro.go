// Package astro provides CGO bindings to the Astronomy Engine C library for
// high-precision lunar phase calculations. It wraps the SearchMoonPhase and
// UtcFromTime functions to compute new moon dates used by the orchestrator.
package astro

/*
#cgo CFLAGS: -I${SRCDIR}/../../thirdparty -O3
#cgo LDFLAGS: -lm
#include "astronomy.c"
*/
import "C"
import "time"

// MoonIlluminationFraction returns the illuminated fraction of the Moon at the
// given UT time, as a value in [0.0, 1.0]. 0.0 = exact new moon, 1.0 = full.
// Used by the orchestrator to decide whether a candidate calendar day has a
// crescent thin enough to be physically interesting.
func MoonIlluminationFraction(t time.Time) float64 {
	ct := C.Astronomy_MakeTime(
		C.int(t.Year()), C.int(t.Month()), C.int(t.Day()),
		C.int(t.Hour()), C.int(t.Minute()), C.double(t.Second()),
	)
	info := C.Astronomy_Illumination(C.BODY_MOON, ct)
	if info.status != C.ASTRO_SUCCESS {
		return 0.0
	}
	return float64(info.phase_fraction)
}

// NewMoonsInYear returns the dates of all astronomical new moons in the given
// calendar year, computed via the Astronomy Engine's lunar phase search.
// The full hour/minute/second of conjunction is preserved so callers can
// reason about exact moon age at any observation moment.
func NewMoonsInYear(year int) []time.Time {
	var moons []time.Time
	startTime := C.Astronomy_MakeTime(C.int(year), 1, 1, 0, 0, 0)
	for {
		result := C.Astronomy_SearchMoonPhase(0, startTime, 40)
		if result.status != C.ASTRO_SUCCESS {
			break
		}
		utc := C.Astronomy_UtcFromTime(result.time)
		if int(utc.year) != year {
			break
		}
		moonTime := time.Date(
			int(utc.year), time.Month(utc.month), int(utc.day),
			int(utc.hour), int(utc.minute), int(float64(utc.second)),
			0, time.UTC,
		)
		moons = append(moons, moonTime)
		startTime = C.Astronomy_AddDays(result.time, 20)
	}
	return moons
}
