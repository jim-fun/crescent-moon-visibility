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

// NewMoonsInYear returns the dates of all astronomical new moons in the given
// calendar year, computed via the Astronomy Engine's lunar phase search.
// Results are truncated to midnight UTC of the day of each new moon.
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
		moonDate := time.Date(int(utc.year), time.Month(utc.month), int(utc.day), 0, 0, 0, 0, time.UTC)
		moons = append(moons, moonDate)
		startTime = C.Astronomy_AddDays(result.time, 20)
	}
	return moons
}
