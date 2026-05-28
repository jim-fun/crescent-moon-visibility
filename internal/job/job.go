package job

import "time"

// Status represents the lifecycle of a web UI generation job.
type Status string

const (
	StatusQueued   Status = "queued"
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusError    Status = "error"
)

// Job represents a map generation request from the web UI.
type Job struct {
	ID          string
	Status      Status
	Params      map[string]string // e.g. start_year, end_year, months, use_gpu
	OutputDir   string
	MapFiles    []string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

// New creates a new Job with queued status.
func New(id string, params map[string]string) *Job {
	return &Job{
		ID:        id,
		Status:    StatusQueued,
		Params:    params,
		StartedAt: time.Now(),
	}
}
