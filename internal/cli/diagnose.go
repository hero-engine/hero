package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hero-engine/hero/internal/async"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	diagnoseBatch bool
	diagnoseJSON  bool
	diagnoseAsync bool
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [spec-slug]",
	Short: "Diagnose bugs — single, batch, or async",
	Long: `Investigate bug specs to find root causes and suggest fixes.

Single mode:
  hero diagnose <slug>           — output spec + investigation instructions
  hero diagnose --async <slug>   — background agent diagnosis

Batch mode:
  hero diagnose --batch          — list all undiagnosed bugs
  hero diagnose --batch --async  — enqueue all undiagnosed bugs for background diagnosis`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiagnose,
}

func init() {
	diagnoseCmd.Flags().BoolVar(&diagnoseBatch, "batch", false, "operate on all undiagnosed bugs")
	diagnoseCmd.Flags().BoolVar(&diagnoseJSON, "json", false, "output as JSON")
	diagnoseCmd.Flags().BoolVar(&diagnoseAsync, "async", false, "background agent diagnosis")
}

// DiagnoseBatchEntry represents a bug ready for diagnosis.
type DiagnoseBatchEntry struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	TrackerID string `json:"tracker_id,omitempty"`
	Path      string `json:"path"`
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if diagnoseBatch {
		return runDiagnoseBatch(projectRoot, heroDir, specs)
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a spec slug, or use --batch to list undiagnosed bugs")
	}

	if diagnoseAsync {
		return runDiagnoseAsync(projectRoot, heroDir, specs, args[0])
	}

	return runDiagnoseSingle(specs, args[0])
}

func runDiagnoseAsync(projectRoot, heroDir string, specs []*spec.Spec, slug string) error {
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("spec %q not found", slug)
	}
	if target.Type != spec.TypeBug {
		return fmt.Errorf("spec %q is type %q, not bug", slug, target.Type)
	}
	if target.Status == spec.StatusCompleted {
		return fmt.Errorf("spec %q is already completed", slug)
	}

	store := async.DefaultStore()
	jobID := generateDiagnoseJobID()

	job := async.Job{
		ID:       jobID,
		Type:     async.JobDiagnose,
		Slug:     target.Slug,
		SpecPath: target.Path,
		Status:   async.StatusPending,
	}

	if err := store.Add(job); err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	// Launch background process
	return launchBackgroundJob(projectRoot, jobID)
}

func runDiagnoseSingle(specs []*spec.Spec, slug string) error {
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("spec %q not found", slug)
	}

	if target.Type != spec.TypeBug {
		return fmt.Errorf("spec %q is type %q, not bug", slug, target.Type)
	}

	if target.Status == spec.StatusCompleted {
		return fmt.Errorf("spec %q is already completed", slug)
	}

	stage := classifyReadiness(target)
	if stage == "diagnosed" || stage == "approved" {
		fmt.Fprintf(os.Stderr, "warning: spec %q appears already diagnosed (stage: %s)\n", slug, stage)
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	if diagnoseJSON {
		out := map[string]interface{}{
			"slug":       target.Slug,
			"title":      target.Title,
			"type":       string(target.Type),
			"status":     string(target.Status),
			"tracker_id": target.TrackerID,
			"path":       target.Path,
			"stage":      stage,
			"content":    string(data),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Bug: %s\n", target.Title)
	if target.TrackerID != "" {
		fmt.Printf("Tracker: %s\n", target.TrackerID)
	}
	fmt.Printf("Spec: %s\n", target.Path)
	fmt.Printf("Stage: %s\n\n", stage)

	fmt.Println("--- Spec Content ---")
	fmt.Println(string(data))
	fmt.Println("--- End Spec ---")
	fmt.Println()
	fmt.Println("Investigation Instructions:")
	fmt.Println("  1. Load the debugging-investigation skill")
	fmt.Printf("  2. Read the spec at: %s\n", target.Path)
	fmt.Println("  3. Investigate the root cause in the codebase")
	fmt.Println("  4. Write findings into the spec file on disk:")
	fmt.Println("     - ## Investigation (what you found)")
	fmt.Println("     - ## Root Cause (classified root cause)")
	fmt.Println("     - ## Suggested Fix Approach (how to fix)")
	fmt.Println("  5. Do NOT move, delete, or rename the spec file")

	return nil
}

func runDiagnoseBatch(projectRoot, heroDir string, specs []*spec.Spec) error {
	var undiagnosed []DiagnoseBatchEntry

	for _, s := range specs {
		if s.Type != spec.TypeBug {
			continue
		}
		if s.Status == spec.StatusCompleted {
			continue
		}
		stage := classifyReadiness(s)
		if stage != "imported" {
			continue
		}
		undiagnosed = append(undiagnosed, DiagnoseBatchEntry{
			Slug:      s.Slug,
			Title:     s.Title,
			TrackerID: s.TrackerID,
			Path:      s.Path,
		})
	}

	if len(undiagnosed) == 0 {
		if diagnoseJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]interface{}{"count": 0, "bugs": []DiagnoseBatchEntry{}})
		}
		fmt.Println("No undiagnosed bugs found in the pipeline.")
		fmt.Println("Import bugs with 'hero import --type bug' first.")
		return nil
	}

	// If --async, enqueue all as background jobs
	if diagnoseAsync {
		return runDiagnoseBatchAsync(projectRoot, undiagnosed)
	}

	if diagnoseJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"count": len(undiagnosed),
			"bugs":  undiagnosed,
		})
	}

	fmt.Printf("Undiagnosed Bugs (%d)\n", len(undiagnosed))
	fmt.Printf("%s\n\n", strings.Repeat("─", 50))

	for i, b := range undiagnosed {
		tracker := ""
		if b.TrackerID != "" {
			tracker = " [" + b.TrackerID + "]"
		}
		title := b.Title
		if title == "" {
			title = b.Slug
		}
		fmt.Printf("  %d. %s%s\n", i+1, title, tracker)
		fmt.Printf("     %s\n", b.Path)
	}

	fmt.Println()
	fmt.Println("To diagnose a single bug:")
	fmt.Println("  hero diagnose <slug>")
	fmt.Println()
	fmt.Println("To batch diagnose via agent:")
	fmt.Println("  hero diagnose --batch --async")

	return nil
}

func runDiagnoseBatchAsync(projectRoot string, bugs []DiagnoseBatchEntry) error {
	store := async.DefaultStore()

	// Generate batch ID to group these jobs
	batchID := generateDiagnoseJobID()

	for _, b := range bugs {
		jobID := generateDiagnoseJobID()
		job := async.Job{
			ID:       jobID,
			Type:     async.JobDiagnose,
			Slug:     b.Slug,
			SpecPath: b.Path,
			Status:   async.StatusPending,
			BatchID:  batchID,
		}
		if err := store.Add(job); err != nil {
			return fmt.Errorf("creating job for %s: %w", b.Slug, err)
		}
	}

	// Launch a single background process to work through the batch
	if err := launchBackgroundBatch(projectRoot, batchID); err != nil {
		return fmt.Errorf("starting batch process: %w", err)
	}

	fmt.Printf("Enqueued %d bugs for async diagnosis\n", len(bugs))
	fmt.Printf("  Batch: %s\n", batchID)
	fmt.Println()
	for _, b := range bugs {
		title := b.Title
		if title == "" {
			title = b.Slug
		}
		fmt.Printf("  - %s\n", title)
	}
	fmt.Println()
	fmt.Println("Run 'hero status' to check progress.")
	return nil
}

func generateDiagnoseJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// launchBackgroundJob starts a detached hero process to run a single job.
func launchBackgroundJob(projectRoot, jobID string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding hero binary: %w", err)
	}

	bgCmd := exec.Command(exe, "job-run", "--job", jobID)
	bgCmd.Dir = projectRoot
	bgCmd.Stdout = nil
	bgCmd.Stderr = nil
	bgCmd.Stdin = nil

	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("starting background process: %w", err)
	}
	bgCmd.Process.Release()

	fmt.Printf("  Job:  %s\n", jobID)
	fmt.Println()
	fmt.Println("Run 'hero status' to check progress.")
	return nil
}

// launchBackgroundBatch starts a detached hero process to run all jobs in a batch.
func launchBackgroundBatch(projectRoot, batchID string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding hero binary: %w", err)
	}

	bgCmd := exec.Command(exe, "job-run", "--batch", batchID)
	bgCmd.Dir = projectRoot
	bgCmd.Stdout = nil
	bgCmd.Stderr = nil
	bgCmd.Stdin = nil

	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("starting background process: %w", err)
	}
	bgCmd.Process.Release()
	return nil
}
