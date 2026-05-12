package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/runner"
	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve <job-id>",
	Short: "Approve a gated job to continue execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runApprove,
}

func runApprove(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	jobPath := filepath.Join(heroDir, "jobs", args[0]+".json")

	data, err := os.ReadFile(jobPath)
	if err != nil {
		return fmt.Errorf("job %q not found", args[0])
	}

	var job runner.JobRecord
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("parsing job: %w", err)
	}

	if job.Status != "awaiting_approval" {
		fmt.Printf("Job %s is %s — not awaiting approval.\n", job.ID, job.Status)
		return nil
	}

	job.Status = "approved"
	updated, _ := json.MarshalIndent(job, "", "  ")
	if err := os.WriteFile(jobPath, updated, 0o644); err != nil {
		return fmt.Errorf("updating job: %w", err)
	}

	fmt.Printf("Job %s approved. Resume with: hero run %s %s\n", job.ID, job.Command, job.Args)
	return nil
}
