package icop

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Sighting represents a single curated ICOP observation record.
type Sighting struct {
	ID              string  `json:"id"`
	Date            string  `json:"date"` // YYYY-MM-DD
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Instrument      string  `json:"instrument"`       // naked_eye, binoculars, telescope
	ReportedResult  string  `json:"reported_result"`  // seen, not_seen
	Notes           string  `json:"notes"`
}

// LoadSightings reads the curated sightings JSON file.
func LoadSightings(path string) ([]Sighting, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sightings file: %w", err)
	}

	var sightings []Sighting
	if err := json.Unmarshal(data, &sightings); err != nil {
		return nil, fmt.Errorf("failed to parse sightings JSON: %w", err)
	}

	return sightings, nil
}

// Validate performs basic sanity checks on a sighting record.
func (s Sighting) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("missing id")
	}
	if _, err := time.Parse("2006-01-02", s.Date); err != nil {
		return fmt.Errorf("invalid date %q: %w", s.Date, err)
	}
	if s.Latitude < -90 || s.Latitude > 90 {
		return fmt.Errorf("invalid latitude %f", s.Latitude)
	}
	if s.Longitude < -180 || s.Longitude > 180 {
		return fmt.Errorf("invalid longitude %f", s.Longitude)
	}
	if s.Instrument != "naked_eye" && s.Instrument != "binoculars" && s.Instrument != "telescope" {
		return fmt.Errorf("unknown instrument: %s", s.Instrument)
	}
	if s.ReportedResult != "seen" && s.ReportedResult != "not_seen" {
		return fmt.Errorf("unknown reported_result: %s", s.ReportedResult)
	}
	return nil
}

// Result represents the outcome of running our model against one sighting.
type Result struct {
	SightingID     string
	Predicted      string // A, B, C, D, E, F (or error)
	Reported       string
	Match          bool
	Notes          string
}

// Summary contains aggregate statistics for a validation run.
type Summary struct {
	Total     int
	Matches   int
	Mismatches int
	MatchRate float64
	ByCategory map[string]int // e.g. "A/B", "C/D/E", etc.
}