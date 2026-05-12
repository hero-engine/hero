package async

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JobStatus represents the lifecycle of an async job.
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

// JobType distinguishes delivery jobs from diagnosis jobs.
type JobType string

const (
	JobDeliver  JobType = "deliver"
	JobDiagnose JobType = "diagnose"
)

// Job represents a single async task (delivery or diagnosis).
type Job struct {
	ID          string    `json:"id"`
	Type        JobType   `json:"type"`
	Slug        string    `json:"slug"`
	SpecPath    string    `json:"spec_path"`
	Branch      string    `json:"branch,omitempty"`
	BaseBranch  string    `json:"base_branch,omitempty"`
	Status      JobStatus `json:"status"`
	BatchID     string    `json:"batch_id,omitempty"`
	PID         int       `json:"pid,omitempty"`
	Error       string    `json:"error,omitempty"`
	PRNumber    int       `json:"pr_number,omitempty"`
	PRURL       string    `json:"pr_url,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LogFile     string    `json:"log_file,omitempty"`
}

// Pending returns all pending jobs, optionally filtered by batch ID.
func (s *JobStore) Pending(batchID string) ([]Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}
	var pending []Job
	for _, j := range jobs {
		if j.Status != StatusPending {
			continue
		}
		if batchID != "" && j.BatchID != batchID {
			continue
		}
		pending = append(pending, j)
	}
	return pending, nil
}

// BatchSummary returns counts by status for a given batch.
func (s *JobStore) BatchSummary(batchID string) (total, completed, failed, running int, err error) {
	jobs, err := s.Load()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, j := range jobs {
		if j.BatchID != batchID {
			continue
		}
		total++
		switch j.Status {
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		case StatusRunning:
			running++
		}
	}
	return
}

// JobStore manages async delivery jobs persisted to disk.
type JobStore struct {
	path string
	mu   sync.Mutex
}

// NewJobStore creates a job store at the given directory.
func NewJobStore(heroDir string) *JobStore {
	return &JobStore{
		path: filepath.Join(heroDir, "async-jobs.json"),
	}
}

// DefaultStore returns a job store at ~/.hero/async-jobs.json.
func DefaultStore() *JobStore {
	home, _ := os.UserHomeDir()
	return NewJobStore(filepath.Join(home, ".hero"))
}

// Load reads all jobs from disk.
func (s *JobStore) Load() ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading jobs: %w", err)
	}

	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("parsing jobs: %w", err)
	}
	return jobs, nil
}

// Save writes all jobs to disk.
func (s *JobStore) save(jobs []Job) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// Add creates a new job and persists it.
func (s *JobStore) Add(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	job.CreatedAt = time.Now()
	jobs = append(jobs, job)
	return s.save(jobs)
}

// Update modifies a job by ID.
func (s *JobStore) Update(id string, fn func(*Job)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == id {
			fn(&jobs[i])
			return s.save(jobs)
		}
	}
	return fmt.Errorf("job %s not found", id)
}

// Get retrieves a job by ID.
func (s *JobStore) Get(id string) (*Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == id {
			return &j, nil
		}
	}
	return nil, nil
}

// GetBySlug finds a job by spec slug (returns the most recent).
func (s *JobStore) GetBySlug(slug string) (*Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}
	for i := len(jobs) - 1; i >= 0; i-- {
		if jobs[i].Slug == slug {
			return &jobs[i], nil
		}
	}
	return nil, nil
}

// Active returns all non-terminal jobs.
func (s *JobStore) Active() ([]Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}
	var active []Job
	for _, j := range jobs {
		if j.Status == StatusPending || j.Status == StatusRunning {
			active = append(active, j)
		}
	}
	return active, nil
}

// Clean removes completed/failed/cancelled jobs older than the given duration.
func (s *JobStore) Clean(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	var kept []Job
	for _, j := range jobs {
		if (j.Status == StatusPending || j.Status == StatusRunning) || j.CreatedAt.After(cutoff) {
			kept = append(kept, j)
		}
	}
	return s.save(kept)
}

func (s *JobStore) loadUnsafe() ([]Job, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}
