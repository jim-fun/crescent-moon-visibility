//go:build web

package main

import (
	"sync"
	"time"

	// Ensure the new internal packages are pulled in during web builds
	// (even if not fully wired yet) so the normal import lines in main.go
	// do not cause "imported and not used" errors under -tags=web.
	_ "github.com/jim-fun/crescent-moon-visibility/internal/job"
	_ "github.com/jim-fun/crescent-moon-visibility/internal/jobrunner"
)

// Legacy job tracking stubs — only built when using -tags=web.
// This lets developers iterate on the web UI (`crescent_maps web`) even
// on machines where the full CGO + Astronomy Engine build currently fails.

type WebJob struct {
	ID          string
	Status      string
	Params      map[string]string
	OutputDir   string
	MapFiles    []string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

var (
	legacyJobs   = make(map[string]*WebJob)
	legacyJobsMu sync.Mutex
)

func newJobID() string {
	return time.Now().Format("20060102150405.000000000")
}

func getJob(id string) *WebJob {
	legacyJobsMu.Lock()
	defer legacyJobsMu.Unlock()
	return legacyJobs[id]
}

func saveJob(j *WebJob) {
	legacyJobsMu.Lock()
	defer legacyJobsMu.Unlock()
	legacyJobs[j.ID] = j
}
