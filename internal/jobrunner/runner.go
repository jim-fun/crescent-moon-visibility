package jobrunner

import (
	"sync"
	"time"

	"github.com/jim-fun/crescent-moon-visibility/internal/job"
)

// Runner manages the execution of web-initiated map generation jobs.
// This is a minimal Phase 2 implementation — it coordinates job state
// and will eventually delegate to internal/task and renderer logic.
type Runner struct {
	jobs map[string]*job.Job
	mu   sync.Mutex
}

// New creates a new Runner.
func New() *Runner {
	return &Runner{
		jobs: make(map[string]*job.Job),
	}
}

// Submit creates a new job and starts it in the background (for now, simulated).
func (r *Runner) Submit(params map[string]string) *job.Job {
	id := time.Now().Format("20060102150405.000000000")
	j := job.New(id, params)

	r.mu.Lock()
	r.jobs[id] = j
	r.mu.Unlock()

	go r.run(j)

	return j
}

// Get returns a job by ID (thread-safe).
func (r *Runner) Get(id string) *job.Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	if j, ok := r.jobs[id]; ok {
		return j
	}
	return nil
}

func (r *Runner) run(j *job.Job) {
	r.updateStatus(j, job.StatusRunning)

	// Phase 2: For now we keep the limited real work that was in the spike.
	// In later iterations this will call into internal/task + renderer packages.
	// The heavy lifting (renderer exec, blending) stays in the trusted paths.

	// Simulate some work + record success for the web UI demo
	time.Sleep(2 * time.Second)

	// For the spike we just mark success with placeholder map references.
	// Real map files would come from the actual generation logic.
	j.MapFiles = []string{
		j.Params["start_year"] + "-03-01",
		j.Params["end_year"] + "-04-01",
	}
	j.OutputDir = "web_outputs/" + j.ID

	r.updateStatus(j, job.StatusDone)
}

func (r *Runner) updateStatus(j *job.Job, s job.Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j.Status = s
	if s == job.StatusDone || s == job.StatusError {
		j.CompletedAt = time.Now()
	}
}
