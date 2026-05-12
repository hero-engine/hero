package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
)

var jobsSubmitCmd = &cobra.Command{
	Use:   "submit <command> <args>",
	Short: "Submit a job to the team server queue",
	Long: `Submit an agent job to the local team server job queue.
The job will be picked up by a worker and executed headlessly.

Examples:
  hero jobs submit deliver csv-export
  hero jobs submit diagnose login-crash --budget 2.00
  hero jobs submit deliver --all --type bug`,
	Args: cobra.MinimumNArgs(1),
	RunE: runJobsSubmit,
}

var (
	jobsSubmitProvider string
	jobsSubmitModel    string
	jobsSubmitBudget   float64
	jobsSubmitMaxTurns int
)

func init() {
	jobsSubmitCmd.Flags().StringVar(&jobsSubmitProvider, "provider", "anthropic", "LLM provider")
	jobsSubmitCmd.Flags().StringVar(&jobsSubmitModel, "model", "", "model name")
	jobsSubmitCmd.Flags().Float64Var(&jobsSubmitBudget, "budget", 0, "cost cap in dollars")
	jobsSubmitCmd.Flags().IntVar(&jobsSubmitMaxTurns, "max-turns", 100, "max agent loop iterations")

	jobsCmd.AddCommand(jobsSubmitCmd)
}

func runJobsSubmit(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found")
	}

	jq, err := serve.NewJobQueue(heroDir)
	if err != nil {
		return fmt.Errorf("opening job queue: %w", err)
	}
	defer jq.Close()

	command := args[0]
	jobArgs := ""
	if len(args) > 1 {
		jobArgs = strings.Join(args[1:], " ")
	}

	job := &serve.Job{
		Command:  command,
		Args:     jobArgs,
		Provider: jobsSubmitProvider,
		Model:    jobsSubmitModel,
		Budget:   jobsSubmitBudget,
		MaxTurns: jobsSubmitMaxTurns,
	}

	if err := jq.Submit(job); err != nil {
		return fmt.Errorf("submitting job: %w", err)
	}

	fmt.Printf("Job submitted: %s\n", job.ID)
	fmt.Printf("  Command: %s %s\n", job.Command, job.Args)
	fmt.Printf("  Status: queued\n")
	fmt.Println("\nThe next available worker will pick it up.")
	return nil
}
