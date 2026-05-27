// Package hmnao compares HM Nautical Almanac Office (HMNAO / UKHO) published
// crescent first-visibility predictions against this project's exact Yallop
// implementation, and reports the deltas.
//
// Accuracy First: this package never fabricates a baseline. Records that are
// still placeholders (no curated numeric latitude/longitude) are reported as
// "pending" and excluded from delta statistics. It also stays free of exec/CGO
// so it is unit-testable; the caller injects the renderer via YallopFunc.
package hmnao

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// FirstVisibilityQBoundary is the Yallop q at the limit of naked-eye crescent
// visibility under perfect conditions (the B/C category boundary). At a point
// HMNAO reports as "first visible", our model should sit near this value, so
// QDelta = OurQ - FirstVisibilityQBoundary is the headline agreement metric.
//
// Source: B.D. Yallop, "A Method for Predicting the First Sighting of the New
// Crescent Moon", NAO Technical Note No. 69 (HMNAO, 1998), Table of q limits.
const FirstVisibilityQBoundary = -0.014

// Record is one curated HMNAO/UKHO published-baseline entry. Latitude and
// longitude are pointers so an uncurated (TBD) record is distinguishable from a
// genuine 0.0 and is honestly reported as pending rather than scored.
type Record struct {
	ID                      string   `json:"id"`
	Date                    string   `json:"date"` // YYYY-MM-DD
	Latitude                *float64 `json:"latitude,omitempty"`
	PredictedFirstVisLonDeg *float64 `json:"predicted_first_visibility_longitude_deg,omitempty"`
	Source                  string   `json:"source"` // citation: publication, year, page/table
	Notes                   string   `json:"notes"`
}

// LoadRecords reads the curated HMNAO baseline JSON file.
func LoadRecords(path string) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read HMNAO baseline file: %w", err)
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to parse HMNAO baseline JSON: %w", err)
	}

	return records, nil
}

// Validate performs basic sanity checks on a record's curated fields.
func (r Record) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("missing id")
	}
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return fmt.Errorf("invalid date %q: %w", r.Date, err)
	}
	if r.Source == "" {
		return fmt.Errorf("missing source citation")
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return fmt.Errorf("invalid latitude %f", *r.Latitude)
	}
	if r.PredictedFirstVisLonDeg != nil && (*r.PredictedFirstVisLonDeg < -180 || *r.PredictedFirstVisLonDeg > 180) {
		return fmt.Errorf("invalid longitude %f", *r.PredictedFirstVisLonDeg)
	}
	return nil
}

// curated reports whether a record carries the numeric fields needed to score it.
func (r Record) curated() bool {
	return r.Latitude != nil && r.PredictedFirstVisLonDeg != nil
}

// YallopFunc computes our model's Yallop category and q at a date/lat/lon.
// The caller wires the exact CPU renderer; injecting it keeps this package free
// of exec/CGO and makes Compare unit-testable with a fake.
type YallopFunc func(date string, lat, lon float64) (category string, q float64, err error)

// Delta is the comparison outcome for one record.
type Delta struct {
	RecordID       string  `json:"record_id"`
	Date           string  `json:"date"`
	BaselineLonDeg float64 `json:"baseline_lon_deg"`
	OurCategory    string  `json:"our_category"`
	OurQ           float64 `json:"our_q"`
	QBoundary      float64 `json:"q_boundary"`
	QDelta         float64 `json:"q_delta"` // OurQ - QBoundary; ~0 => agreement at HMNAO's first-visibility point
	Status         string  `json:"status"`  // computed, pending, error
	Detail         string  `json:"detail"`
}

// Compare scores each record against our Yallop implementation. Pending
// (uncurated) and errored records are recorded with that status and a zero
// QDelta, never a fabricated one.
func Compare(records []Record, yallop YallopFunc) []Delta {
	deltas := make([]Delta, 0, len(records))

	for _, r := range records {
		d := Delta{
			RecordID:       r.ID,
			Date:           r.Date,
			QBoundary:      FirstVisibilityQBoundary,
			BaselineLonDeg: 0,
		}

		if !r.curated() {
			d.Status = "pending"
			d.Detail = "placeholder: latitude/longitude not yet curated"
			deltas = append(deltas, d)
			continue
		}

		d.BaselineLonDeg = *r.PredictedFirstVisLonDeg

		cat, q, err := yallop(r.Date, *r.Latitude, *r.PredictedFirstVisLonDeg)
		if err != nil {
			d.Status = "error"
			d.Detail = err.Error()
			deltas = append(deltas, d)
			continue
		}

		d.OurCategory = cat
		d.OurQ = q
		d.QDelta = q - FirstVisibilityQBoundary
		d.Status = "computed"
		d.Detail = "compared against injected YallopFunc"

		deltas = append(deltas, d)
	}

	return deltas
}
