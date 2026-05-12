package serve

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// JobStatus represents the lifecycle state of a job.
type JobStatus string

const (
	JobQueued           JobStatus = "queued"
	JobRunning          JobStatus = "running"
	JobCompleted        JobStatus = "completed"
	JobFailed           JobStatus = "failed"
	JobCancelled        JobStatus = "cancelled"
	JobAwaitingApproval JobStatus = "awaiting_approval"
	JobApproved         JobStatus = "approved"
	JobBudgetExceeded   JobStatus = "budget_exceeded"
	JobTurnLimit        JobStatus = "turn_limit"
)

// Job represents a queued or completed agent job.
type Job struct {
	ID           string    `json:"id"`
	Command      string    `json:"command"`
	Args         string    `json:"args"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Status       JobStatus `json:"status"`
	SubmittedBy  string    `json:"submitted_by"`
	SubmittedAt  time.Time `json:"submitted_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Turns        int       `json:"turns"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	EstCost      float64   `json:"est_cost"`
	Budget       float64   `json:"budget"`
	MaxTurns     int       `json:"max_turns"`
	Error        string    `json:"error,omitempty"`
	WorkerID     string    `json:"worker_id,omitempty"`
}

// JobQueue manages the job lifecycle backed by SQLite.
type JobQueue struct {
	db  *sql.DB
	mu  sync.RWMutex
}

// NewJobQueue opens or creates the job queue database.
func NewJobQueue(heroDir string) (*JobQueue, error) {
	dbPath := filepath.Join(heroDir, "jobs.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening jobs db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	jq := &JobQueue{db: db}
	if err := jq.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating jobs db: %w", err)
	}

	return jq, nil
}

func (jq *JobQueue) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		args TEXT NOT NULL DEFAULT '',
		provider TEXT NOT NULL DEFAULT 'anthropic',
		model TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'queued',
		submitted_by TEXT NOT NULL DEFAULT '',
		submitted_at TEXT NOT NULL,
		started_at TEXT,
		completed_at TEXT,
		turns INTEGER NOT NULL DEFAULT 0,
		input_tokens INTEGER NOT NULL DEFAULT 0,
		output_tokens INTEGER NOT NULL DEFAULT 0,
		est_cost REAL NOT NULL DEFAULT 0,
		budget REAL NOT NULL DEFAULT 0,
		max_turns INTEGER NOT NULL DEFAULT 100,
		error TEXT NOT NULL DEFAULT '',
		worker_id TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_submitted_by ON jobs(submitted_by);
	CREATE INDEX IF NOT EXISTS idx_jobs_submitted_at ON jobs(submitted_at);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		agent TEXT NOT NULL,
		spec_slug TEXT NOT NULL DEFAULT '',
		command TEXT NOT NULL DEFAULT '',
		started_at TEXT NOT NULL,
		last_seen TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}'
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

	CREATE TABLE IF NOT EXISTS usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		job_id TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER NOT NULL DEFAULT 0,
		output_tokens INTEGER NOT NULL DEFAULT 0,
		est_cost REAL NOT NULL DEFAULT 0,
		recorded_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_usage_user ON usage(user_id);
	CREATE INDEX IF NOT EXISTS idx_usage_recorded ON usage(recorded_at);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL DEFAULT '',
		display_name TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'member',
		oauth_provider TEXT NOT NULL DEFAULT '',
		oauth_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		last_login TEXT NOT NULL DEFAULT ''
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_oauth ON users(oauth_provider, oauth_id);

	CREATE TABLE IF NOT EXISTS claims (
		slug TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		claimed_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS feed (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT '',
		event_type TEXT NOT NULL,
		spec_slug TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_feed_created ON feed(created_at);
	CREATE INDEX IF NOT EXISTS idx_feed_type ON feed(event_type);
	`
	_, err := jq.db.Exec(schema)
	return err
}

// Close closes the database.
func (jq *JobQueue) Close() error {
	return jq.db.Close()
}

// Submit adds a new job to the queue.
func (jq *JobQueue) Submit(job *Job) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job.ID == "" {
		job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if job.Status == "" {
		job.Status = JobQueued
	}
	job.SubmittedAt = time.Now()

	_, err := jq.db.Exec(`
		INSERT INTO jobs (id, command, args, provider, model, status, submitted_by,
			submitted_at, budget, max_turns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Command, job.Args, job.Provider, job.Model,
		job.Status, job.SubmittedBy, job.SubmittedAt.Format(time.RFC3339),
		job.Budget, job.MaxTurns,
	)
	return err
}

// Dequeue claims the next queued job for a worker.
func (jq *JobQueue) Dequeue(workerID string) (*Job, error) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	tx, err := jq.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT id, command, args, provider, model, status, submitted_by,
			submitted_at, budget, max_turns
		FROM jobs WHERE status = 'queued'
		ORDER BY submitted_at ASC LIMIT 1`)

	var job Job
	var submittedAt string
	err = row.Scan(&job.ID, &job.Command, &job.Args, &job.Provider,
		&job.Model, &job.Status, &job.SubmittedBy, &submittedAt,
		&job.Budget, &job.MaxTurns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	job.SubmittedAt, _ = time.Parse(time.RFC3339, submittedAt)
	now := time.Now()
	job.StartedAt = &now
	job.Status = JobRunning
	job.WorkerID = workerID

	_, err = tx.Exec(`
		UPDATE jobs SET status = ?, started_at = ?, worker_id = ?
		WHERE id = ?`,
		job.Status, now.Format(time.RFC3339), workerID, job.ID)
	if err != nil {
		return nil, err
	}

	return &job, tx.Commit()
}

// Update updates a job's status and metrics.
func (jq *JobQueue) Update(job *Job) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	var completedAt *string
	if job.CompletedAt != nil {
		s := job.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	_, err := jq.db.Exec(`
		UPDATE jobs SET status = ?, turns = ?, input_tokens = ?,
			output_tokens = ?, est_cost = ?, error = ?, completed_at = ?
		WHERE id = ?`,
		job.Status, job.Turns, job.InputTokens, job.OutputTokens,
		job.EstCost, job.Error, completedAt, job.ID)
	return err
}

// Get retrieves a job by ID.
func (jq *JobQueue) Get(id string) (*Job, error) {
	jq.mu.RLock()
	defer jq.mu.RUnlock()

	return jq.scanJob(jq.db.QueryRow(`
		SELECT id, command, args, provider, model, status, submitted_by,
			submitted_at, started_at, completed_at, turns, input_tokens,
			output_tokens, est_cost, budget, max_turns, error, worker_id
		FROM jobs WHERE id = ?`, id))
}

// List returns jobs filtered by status (empty = all), most recent first.
func (jq *JobQueue) List(status string, limit int) ([]*Job, error) {
	jq.mu.RLock()
	defer jq.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = jq.db.Query(`
			SELECT id, command, args, provider, model, status, submitted_by,
				submitted_at, started_at, completed_at, turns, input_tokens,
				output_tokens, est_cost, budget, max_turns, error, worker_id
			FROM jobs WHERE status = ?
			ORDER BY submitted_at DESC LIMIT ?`, status, limit)
	} else {
		rows, err = jq.db.Query(`
			SELECT id, command, args, provider, model, status, submitted_by,
				submitted_at, started_at, completed_at, turns, input_tokens,
				output_tokens, est_cost, budget, max_turns, error, worker_id
			FROM jobs ORDER BY submitted_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := jq.scanJobRow(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Cancel marks a job as cancelled.
func (jq *JobQueue) Cancel(id string) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	result, err := jq.db.Exec(`
		UPDATE jobs SET status = 'cancelled', completed_at = ?
		WHERE id = ? AND status IN ('queued', 'running', 'awaiting_approval')`, now, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job %s not found or not cancellable", id)
	}
	return nil
}

// Approve marks a gated job as approved.
func (jq *JobQueue) Approve(id string) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	result, err := jq.db.Exec(`
		UPDATE jobs SET status = 'queued'
		WHERE id = ? AND status = 'awaiting_approval'`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job %s not found or not awaiting approval", id)
	}
	return nil
}

// Reject marks a gated job as cancelled with a reason.
func (jq *JobQueue) Reject(id, reason string) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	result, err := jq.db.Exec(`
		UPDATE jobs SET status = 'cancelled', error = ?, completed_at = ?
		WHERE id = ? AND status = 'awaiting_approval'`, reason, now, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job %s not found or not awaiting approval", id)
	}
	return nil
}

// RegisterSession registers or updates an active session.
func (jq *JobQueue) RegisterSession(id, userID, agent, specSlug, command string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := jq.db.Exec(`
		INSERT INTO sessions (id, user_id, agent, spec_slug, command, started_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_seen = ?, spec_slug = ?, command = ?`,
		id, userID, agent, specSlug, command, now, now, now, specSlug, command)
	return err
}

// UnregisterSession removes a session.
func (jq *JobQueue) UnregisterSession(id string) error {
	_, err := jq.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// ActiveSessions returns all sessions seen in the last 10 minutes.
func (jq *JobQueue) ActiveSessions() ([]map[string]string, error) {
	cutoff := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	rows, err := jq.db.Query(`
		SELECT id, user_id, agent, spec_slug, command, started_at, last_seen
		FROM sessions WHERE last_seen > ?
		ORDER BY last_seen DESC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []map[string]string
	for rows.Next() {
		var id, userID, agent, specSlug, command, startedAt, lastSeen string
		if err := rows.Scan(&id, &userID, &agent, &specSlug, &command, &startedAt, &lastSeen); err != nil {
			return nil, err
		}
		sessions = append(sessions, map[string]string{
			"id": id, "user_id": userID, "agent": agent,
			"spec_slug": specSlug, "command": command,
			"started_at": startedAt, "last_seen": lastSeen,
		})
	}
	return sessions, nil
}

// RecordUsage logs API usage for a user.
func (jq *JobQueue) RecordUsage(userID, jobID, provider, model string, inputTokens, outputTokens int, cost float64) error {
	_, err := jq.db.Exec(`
		INSERT INTO usage (user_id, job_id, provider, model, input_tokens, output_tokens, est_cost, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, jobID, provider, model, inputTokens, outputTokens, cost,
		time.Now().Format(time.RFC3339))
	return err
}

// UserUsageToday returns total cost for a user today.
func (jq *JobQueue) UserUsageToday(userID string) (float64, error) {
	today := time.Now().Format("2006-01-02")
	var total sql.NullFloat64
	err := jq.db.QueryRow(`
		SELECT SUM(est_cost) FROM usage
		WHERE user_id = ? AND recorded_at >= ?`, userID, today).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}

// UsageSummary returns per-user usage for a time period.
func (jq *JobQueue) UsageSummary(since time.Time) ([]map[string]interface{}, error) {
	rows, err := jq.db.Query(`
		SELECT user_id, COUNT(*) as jobs, SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens, SUM(est_cost) as total_cost
		FROM usage WHERE recorded_at >= ?
		GROUP BY user_id ORDER BY total_cost DESC`, since.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []map[string]interface{}
	for rows.Next() {
		var userID string
		var jobs, inputTokens, outputTokens int
		var totalCost float64
		if err := rows.Scan(&userID, &jobs, &inputTokens, &outputTokens, &totalCost); err != nil {
			return nil, err
		}
		summaries = append(summaries, map[string]interface{}{
			"user_id":       userID,
			"jobs":          jobs,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_cost":    totalCost,
		})
	}
	return summaries, nil
}

func (jq *JobQueue) scanJob(row *sql.Row) (*Job, error) {
	var job Job
	var submittedAt string
	var startedAt, completedAt sql.NullString

	err := row.Scan(&job.ID, &job.Command, &job.Args, &job.Provider,
		&job.Model, &job.Status, &job.SubmittedBy, &submittedAt,
		&startedAt, &completedAt, &job.Turns, &job.InputTokens,
		&job.OutputTokens, &job.EstCost, &job.Budget, &job.MaxTurns,
		&job.Error, &job.WorkerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found")
		}
		return nil, err
	}

	job.SubmittedAt, _ = time.Parse(time.RFC3339, submittedAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	return &job, nil
}

func (jq *JobQueue) scanJobRow(rows *sql.Rows) (*Job, error) {
	var job Job
	var submittedAt string
	var startedAt, completedAt sql.NullString

	err := rows.Scan(&job.ID, &job.Command, &job.Args, &job.Provider,
		&job.Model, &job.Status, &job.SubmittedBy, &submittedAt,
		&startedAt, &completedAt, &job.Turns, &job.InputTokens,
		&job.OutputTokens, &job.EstCost, &job.Budget, &job.MaxTurns,
		&job.Error, &job.WorkerID)
	if err != nil {
		return nil, err
	}

	job.SubmittedAt, _ = time.Parse(time.RFC3339, submittedAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	return &job, nil
}
