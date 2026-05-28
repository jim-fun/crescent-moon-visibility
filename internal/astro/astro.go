// Package astro provides CGO bindings to the Astronomy Engine C library for
// high-precision lunar phase calculations. It wraps the SearchMoonPhase and
// UtcFromTime functions to compute new moon dates used by the orchestrator.
//
// This file is excluded when building with the "web" tag (see astro_stub.go).
//go:build !web
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

// GetMoonAgeHours calculates the exact moon age in hours at the best-time of visibility
// for the given latitude, longitude, and base date.
func GetMoonAgeHours(latitude, longitude float64, t time.Time) float64 {
	ct := C.Astronomy_MakeTime(
		C.int(t.Year()), C.int(t.Month()), C.int(t.Day()),
		C.int(12), C.int(0), C.double(0), // Use noon UTC as base_time
	)
	// Base time adjusted by longitude
	baseTime := C.Astronomy_AddDays(ct, C.double(-longitude/360.0))
	observer := C.astro_observer_t{
		latitude:  C.double(latitude),
		longitude: C.double(longitude),
		height:    C.double(0),
	}
	sunset := C.Astronomy_SearchRiseSet(C.BODY_SUN, observer, C.DIRECTION_SET, baseTime, 1)
	moonset := C.Astronomy_SearchRiseSet(C.BODY_MOON, observer, C.DIRECTION_SET, baseTime, 1)
	if sunset.status != C.ASTRO_SUCCESS || moonset.status != C.ASTRO_SUCCESS {
		return -1.0
	}
	lagTime := (moonset.time.ut - sunset.time.ut)
	var bestTime C.astro_time_t
	if lagTime < 0 {
		bestTime = sunset.time
	} else {
		bestTime = C.Astronomy_AddDays(sunset.time, C.double(lagTime*4.0/9.0))
	}
	newMoonPrev := C.Astronomy_SearchMoonPhase(0, sunset.time, -35).time
	newMoonNext := C.Astronomy_SearchMoonPhase(0, sunset.time, 35).time
	var nmNearest C.astro_time_t
	if (sunset.time.ut - newMoonPrev.ut) <= (newMoonNext.ut - sunset.time.ut) {
		nmNearest = newMoonPrev
	} else {
		nmNearest = newMoonNext
	}
	return float64(bestTime.ut-nmNearest.ut) * 24.0
}

