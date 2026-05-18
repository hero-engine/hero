package async

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner executes async jobs (delivery or diagnosis).
type Runner struct {
	store      *JobStore
	projectDir string
	heroDir    string
}

// NewRunner creates a runner for the given project.
func NewRunner(store *JobStore, projectDir, heroDir string) *Runner {
	return &Runner{
		store:      store,
		projectDir: projectDir,
		heroDir:    heroDir,
	}
}

// Run executes a single job, dispatching by job type.
func (r *Runner) Run(jobID string) error {
	job, err := r.store.Get(jobID)
	if err != nil || job == nil {
		return fmt.Errorf("job %s not found", jobID)
	}

	switch job.Type {
	case JobDiagnose:
		return r.runDiagnose(jobID, job)
	default:
		return r.runDeliver(jobID, job)
	}
}

// RunBatch processes all pending jobs in a batch sequentially.
func (r *Runner) RunBatch(batchID string) error {
	for {
		pending, err := r.store.Pending(batchID)
		if err != nil {
			return fmt.Errorf("loading pending jobs: %w", err)
		}
		if len(pending) == 0 {
			return nil
		}
		// Run the first pending job — continue even if it fails
		if err := r.Run(pending[0].ID); err != nil {
			fmt.Fprintf(os.Stderr, "job %s (%s) failed: %v\n", pending[0].ID, pending[0].Slug, err)
		}
	}
}

// runDiagnose executes a diagnosis job: run agent in-place, no branch/PR.
func (r *Runner) runDiagnose(jobID string, job *Job) error {
	now := time.Now()
	r.store.Update(jobID, func(j *Job) {
		j.Status = StatusRunning
		j.StartedAt = now
		j.PID = os.Getpid()
	})

	logDir := filepath.Join(r.heroDir, "async-logs")
	os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, jobID+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return r.fail(jobID, fmt.Errorf("creating log file: %w", err))
	}
	defer logFile.Close()

	r.store.Update(jobID, func(j *Job) {
		j.LogFile = logPath
	})

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintln(logFile, time.Now().Format("15:04:05"), msg)
	}

	// Run agent with /diagnose command
	log("Starting agent diagnosis for %s", job.Slug)
	agentCmd := r.resolveAgentCommand(job)
	if agentCmd == nil {
		return r.fail(jobID, fmt.Errorf("no agent command configured — set HERO_AGENT_CMD or install opencode"))
	}

	agentCmd.Dir = r.projectDir
	agentCmd.Stdout = logFile
	agentCmd.Stderr = logFile

	if err := agentCmd.Run(); err != nil {
		log("Agent failed: %v", err)
		return r.fail(jobID, fmt.Errorf("agent diagnosis failed: %w", err))
	}

	log("Agent diagnosis completed")

	r.store.Update(jobID, func(j *Job) {
		j.Status = StatusCompleted
		j.CompletedAt = time.Now()
	})

	log("Job completed successfully")
	return nil
}

// runDeliver executes a delivery job: create branch, run agent, commit, push, PR.
func (r *Runner) runDeliver(jobID string, job *Job) error {
	now := time.Now()
	r.store.Update(jobID, func(j *Job) {
		j.Status = StatusRunning
		j.StartedAt = now
		j.PID = os.Getpid()
	})

	logDir := filepath.Join(r.heroDir, "async-logs")
	os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, jobID+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return r.fail(jobID, fmt.Errorf("creating log file: %w", err))
	}
	defer logFile.Close()

	r.store.Update(jobID, func(j *Job) {
		j.LogFile = logPath
	})

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintln(logFile, time.Now().Format("15:04:05"), msg)
	}

	// 1. Create feature branch
	log("Creating branch %s from %s", job.Branch, job.BaseBranch)
	if err := r.git("checkout", "-b", job.Branch); err != nil {
		return r.fail(jobID, fmt.Errorf("creating branch: %w", err))
	}
	defer func() {
		j, _ := r.store.Get(jobID)
		if j != nil && j.Status == StatusFailed {
			r.git("checkout", job.BaseBranch)
		}
	}()

	// 2. Run agent delivery
	log("Starting agent delivery for %s", job.Slug)
	agentCmd := r.resolveAgentCommand(job)
	if agentCmd == nil {
		return r.fail(jobID, fmt.Errorf("no agent command configured — set HERO_AGENT_CMD or install opencode"))
	}

	agentCmd.Dir = r.projectDir
	agentCmd.Stdout = logFile
	agentCmd.Stderr = logFile

	if err := agentCmd.Run(); err != nil {
		log("Agent failed: %v", err)
		return r.fail(jobID, fmt.Errorf("agent delivery failed: %w", err))
	}

	log("Agent delivery completed")

	// 2b. Auto-archive the spec when the agent left it at status: completed.
	// /deliver's contract ends with "status: completed" in the spec
	// frontmatter — without this hook the spec strands under planning/
	// and the user has to remember `hero spec complete`. We invoke the
	// CLI verb so the move + reindex stay in one place (complete.go).
	// `spec complete` is idempotent (a non-completed status no-ops).
	// Run this *before* the commit so the archive move travels with
	// the agent's delivery PR rather than dangling on disk.
	if exe, err := os.Executable(); err == nil {
		archiveCmd := exec.Command(exe, "spec", "complete", job.SpecPath)
		archiveCmd.Dir = r.projectDir
		archiveCmd.Stdout = logFile
		archiveCmd.Stderr = logFile
		if err := archiveCmd.Run(); err != nil {
			log("Auto-archive failed: %v", err)
		} else {
			log("Auto-archived spec %s", job.Slug)
		}
	}

	// 3. Check for changes
	statusOut, err := r.gitOutput("status", "--porcelain")
	if err != nil {
		return r.fail(jobID, fmt.Errorf("checking git status: %w", err))
	}
	if strings.TrimSpace(statusOut) == "" {
		log("No changes produced by agent")
		return r.fail(jobID, fmt.Errorf("agent produced no changes"))
	}

	// 4. Stage and commit
	log("Committing changes")
	if err := r.git("add", "-A"); err != nil {
		return r.fail(jobID, fmt.Errorf("staging changes: %w", err))
	}

	commitMsg := fmt.Sprintf("feat: deliver %s (async)", job.Slug)
	if err := r.git("commit", "-m", commitMsg); err != nil {
		return r.fail(jobID, fmt.Errorf("committing: %w", err))
	}

	// 5. Push branch
	log("Pushing branch %s", job.Branch)
	if err := r.git("push", "-u", "origin", job.Branch); err != nil {
		return r.fail(jobID, fmt.Errorf("pushing branch: %w", err))
	}

	// 6. Create PR
	log("Creating pull request")
	prURL, err := r.createPR(job)
	if err != nil {
		log("PR creation failed: %v (branch is pushed, create PR manually)", err)
	}

	// 7. Switch back to base branch
	r.git("checkout", job.BaseBranch)

	r.store.Update(jobID, func(j *Job) {
		j.Status = StatusCompleted
		j.CompletedAt = time.Now()
		if prURL != "" {
			j.PRURL = prURL
		}
	})

	log("Job completed successfully")
	return nil
}

func (r *Runner) fail(jobID string, err error) error {
	r.store.Update(jobID, func(j *Job) {
		j.Status = StatusFailed
		j.Error = err.Error()
		j.CompletedAt = time.Now()
	})
	return err
}

func (r *Runner) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.projectDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *Runner) gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.projectDir
	out, err := cmd.Output()
	return string(out), err
}

// resolveAgentCommand determines which agent to use.
// Priority: HERO_AGENT_CMD env var > opencode > claude > error
func (r *Runner) resolveAgentCommand(job *Job) *exec.Cmd {
	prompt := r.agentPrompt(job)

	if agentCmd := os.Getenv("HERO_AGENT_CMD"); agentCmd != "" {
		parts := strings.Fields(agentCmd)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdin = strings.NewReader(prompt)
		return cmd
	}

	if path, err := exec.LookPath("opencode"); err == nil {
		cmd := exec.Command(path, "--prompt", prompt)
		return cmd
	}

	if path, err := exec.LookPath("claude"); err == nil {
		cmd := exec.Command(path, "--print", prompt)
		return cmd
	}

	return nil
}

// agentPrompt returns the slash command for the agent based on job type.
func (r *Runner) agentPrompt(job *Job) string {
	switch job.Type {
	case JobDiagnose:
		return fmt.Sprintf("/diagnose %s", job.Slug)
	default:
		return fmt.Sprintf("/deliver %s", job.Slug)
	}
}

func (r *Runner) createPR(job *Job) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found")
	}

	title := fmt.Sprintf("feat: deliver %s", job.Slug)
	body := fmt.Sprintf("Async delivery of spec `%s`.\n\nSpec: `%s`", job.Slug, job.SpecPath)

	cmd := exec.Command("gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--base", job.BaseBranch,
	)
	cmd.Dir = r.projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// BranchName generates a branch name for async delivery of a spec.
func BranchName(slug string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, slug)
	return fmt.Sprintf("hero/deliver/%s", safe)
}
