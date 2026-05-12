package serve

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/runner"
)

// WorkerPool manages a pool of job execution workers.
type WorkerPool struct {
	jq          *JobQueue
	projectRoot string
	heroDir     string
	count       int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewWorkerPool creates a worker pool that processes jobs from the queue.
func NewWorkerPool(jq *JobQueue, projectRoot, heroDir string, count int) *WorkerPool {
	if count <= 0 {
		count = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		jq:          jq,
		projectRoot: projectRoot,
		heroDir:     heroDir,
		count:       count,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start launches the worker goroutines.
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.count; i++ {
		wp.wg.Add(1)
		go wp.worker(fmt.Sprintf("worker-%d", i))
	}
	log.Printf("hero team: started %d worker(s)", wp.count)
}

// Stop gracefully shuts down all workers.
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	log.Printf("hero team: all workers stopped")
}

func (wp *WorkerPool) worker(id string) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
		}

		job, err := wp.jq.Dequeue(id)
		if err != nil {
			log.Printf("hero team %s: dequeue error: %v", id, err)
			sleep(wp.ctx, 5*time.Second)
			continue
		}

		if job == nil {
			sleep(wp.ctx, 2*time.Second)
			continue
		}

		log.Printf("hero team %s: executing job %s (%s %s)", id, job.ID, job.Command, job.Args)
		wp.executeJob(id, job)
	}
}

func (wp *WorkerPool) executeJob(workerID string, job *Job) {
	result, err := runner.Run(runner.RunConfig{
		ProjectRoot: wp.projectRoot,
		HeroDir:     wp.heroDir,
		Provider:    job.Provider,
		Model:       job.Model,
		Command:     job.Command,
		Args:        job.Args,
		MaxTurns:    job.MaxTurns,
		Budget:      job.Budget,
	})

	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		job.Status = JobFailed
		job.Error = err.Error()
		log.Printf("hero team %s: job %s failed: %v", workerID, job.ID, err)
	} else if result != nil {
		job.Turns = result.Turns
		job.InputTokens = result.InputTokens
		job.OutputTokens = result.OutputTokens
		job.EstCost = result.EstCost
		job.Status = JobStatus(result.Status)
		job.Error = result.Error
		log.Printf("hero team %s: job %s %s (%d turns, $%.2f)",
			workerID, job.ID, result.Status, result.Turns, result.EstCost)
	}

	if err := wp.jq.Update(job); err != nil {
		log.Printf("hero team %s: failed to update job %s: %v", workerID, job.ID, err)
	}

	// Record usage
	if job.SubmittedBy != "" && job.EstCost > 0 {
		wp.jq.RecordUsage(job.SubmittedBy, job.ID, job.Provider, job.Model,
			job.InputTokens, job.OutputTokens, job.EstCost)
	}

	// Post-delivery checks (non-blocking)
	if job.Status == JobCompleted && (job.Command == "deliver" || job.Command == "diagnose") {
		go wp.runPostDeliveryChecks(workerID, job)
	}
}

func (wp *WorkerPool) runPostDeliveryChecks(workerID string, job *Job) {
	slug := job.Args
	if slug == "" {
		return
	}

	log.Printf("hero team %s: running post-delivery checks for %s", workerID, slug)

	// Drift check
	cmd := exec.Command("hero", "drift", slug)
	cmd.Dir = wp.projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("hero team %s: drift check for %s: %s", workerID, slug, string(out))
	}

	// Coverage check
	cmd = exec.Command("hero", "coverage", slug)
	cmd.Dir = wp.projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("hero team %s: coverage check for %s: %s", workerID, slug, string(out))
	}

	// Contract check
	cmd = exec.Command("hero", "contract", "check", "--slug", slug)
	cmd.Dir = wp.projectRoot
	cmd.CombinedOutput() // best effort

	log.Printf("hero team %s: post-delivery checks complete for %s", workerID, slug)
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
