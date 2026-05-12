package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/runner"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs [id]",
	Short: "List or inspect headless agent jobs",
	Long: `View recent hero run job history or inspect a specific job.

Examples:
  hero jobs                # list recent jobs
  hero jobs <id>           # show details for a specific job`,
	RunE: runJobs,
}

var jobsLimit int

func init() {
	jobsCmd.Flags().IntVar(&jobsLimit, "limit", 20, "max jobs to display")
}

func runJobs(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)

	// Single job detail
	if len(args) > 0 {
		jobPath := filepath.Join(heroDir, "jobs", args[0]+".json")
		data, err := os.ReadFile(jobPath)
		if err != nil {
			return fmt.Errorf("job %q not found", args[0])
		}
		var job runner.JobRecord
		if err := json.Unmarshal(data, &job); err != nil {
			return fmt.Errorf("parsing job: %w", err)
		}
		fmt.Printf("Job: %s\n", job.ID)
		fmt.Printf("Command: %s %s\n", job.Command, job.Args)
		fmt.Printf("Provider: %s, Model: %s\n", job.Provider, job.Model)
		fmt.Printf("Status: %s\n", job.Status)
		fmt.Printf("Turns: %d\n", job.Turns)
		fmt.Printf("Tokens: %d in / %d out\n", job.InputTokens, job.OutputTokens)
		fmt.Printf("Cost: $%.2f\n", job.EstCost)
		fmt.Printf("Started: %s\n", job.StartedAt.Format(time.RFC3339))
		if !job.CompletedAt.IsZero() {
			dur := job.CompletedAt.Sub(job.StartedAt)
			fmt.Printf("Completed: %s (%s)\n", job.CompletedAt.Format(time.RFC3339), dur.Round(time.Second))
		}
		if job.Error != "" {
			fmt.Printf("Error: %s\n", job.Error)
		}
		return nil
	}

	// Try SQLite job queue first (team server jobs)
	jq, jqErr := serve.NewJobQueue(heroDir)
	if jqErr == nil {
		defer jq.Close()
		queueJobs, listErr := jq.List("", jobsLimit)
		if listErr == nil && len(queueJobs) > 0 {
			fmt.Printf("Recent jobs (%d):\n\n", len(queueJobs))
			for _, j := range queueJobs {
				dur := ""
				if j.CompletedAt != nil {
					dur = fmt.Sprintf(" (%s)", j.CompletedAt.Sub(j.SubmittedAt).Round(time.Second))
				}
				idDisplay := j.ID
				if len(idDisplay) > 16 {
					idDisplay = idDisplay[:16] + "..."
				}
				model := j.Model
				if model == "" {
					model = "default"
				}
				fmt.Printf("  %-20s  %-18s  %-15s  %s %s  $%.2f%s\n",
					idDisplay,
					j.Status,
					j.Provider+"/"+model,
					j.Command,
					j.Args,
					j.EstCost,
					dur,
				)
			}
			return nil
		}
	}

	// Fall back to JSON file jobs (legacy / hero run direct)
	jobs, err := runner.ListJobs(heroDir, jobsLimit)
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs found. Run `hero run <command> <args>` or `hero jobs submit` to create a job.")
		return nil
	}

	fmt.Printf("Recent jobs (%d):\n\n", len(jobs))
	for _, j := range jobs {
		dur := ""
		if !j.CompletedAt.IsZero() {
			dur = fmt.Sprintf(" (%s)", j.CompletedAt.Sub(j.StartedAt).Round(time.Second))
		}
		idDisplay := j.ID
		if len(idDisplay) > 16 {
			idDisplay = idDisplay[:16] + "..."
		}
		fmt.Printf("  %-20s  %-10s  %-15s  %s %s  $%.2f%s\n",
			idDisplay,
			j.Status,
			j.Provider+"/"+j.Model,
			j.Command,
			j.Args,
			j.EstCost,
			dur,
		)
	}

	return nil
}
