package cli

import (
	"fmt"

	"github.com/hero-engine/hero/internal/async"
	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var (
	jobRunJobID   string
	jobRunBatchID string
)

var jobRunCmd = &cobra.Command{
	Use:    "job-run",
	Short:  "Internal: execute async job(s) in background",
	Hidden: true,
	RunE:   runJobRun,
}

func init() {
	jobRunCmd.Flags().StringVar(&jobRunJobID, "job", "", "single job ID to execute")
	jobRunCmd.Flags().StringVar(&jobRunBatchID, "batch", "", "batch ID — execute all pending jobs sequentially")
}

func runJobRun(cmd *cobra.Command, args []string) error {
	if jobRunJobID == "" && jobRunBatchID == "" {
		return fmt.Errorf("specify --job or --batch")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	store := async.DefaultStore()
	runner := async.NewRunner(store, projectRoot, heroDir)

	if jobRunBatchID != "" {
		return runner.RunBatch(jobRunBatchID)
	}

	return runner.Run(jobRunJobID)
}
