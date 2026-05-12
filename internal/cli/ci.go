package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/environment"
	"github.com/spf13/cobra"
)

var (
	ciBranch string
	ciFormat string
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Show CI pipeline status for the current branch",
	Long: `Queries the configured CI provider for the most recent pipeline run
on the current branch. Shows pass/fail status, failed step details,
and a link to the run.

Requires environment.ci in hero.json:

  { "environment": { "ci": { "provider": "github-actions" } } }

Examples:
  hero ci                    # status for current branch
  hero ci --branch main      # status for a specific branch
  hero ci --format json      # machine-readable output`,
	RunE: runCI,
}

func init() {
	ciCmd.Flags().StringVar(&ciBranch, "branch", "", "branch to check (defaults to current git branch)")
	ciCmd.Flags().StringVar(&ciFormat, "format", "", "output format: text (default), json")
}

func runCI(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Get CI provider config
	if cfg.Environment == nil || cfg.Environment.CI == nil {
		fmt.Println("No CI provider configured.")
		fmt.Println("Add to hero.json:")
		fmt.Println(`  { "environment": { "ci": { "provider": "github-actions" } } }`)
		return nil
	}

	ciCfg := cfg.Environment.CI
	project := ""
	token := ""

	// Reuse tracker config for project/token if available
	if cfg.Tracker != nil {
		project = cfg.Tracker.Project
		if cfg.Tracker.TokenEnv != "" {
			token = os.Getenv(cfg.Tracker.TokenEnv)
		}
	}

	provider, err := environment.NewCIProvider(ciCfg.Provider, project, token)
	if err != nil {
		return err
	}

	// Resolve branch
	branch := ciBranch
	if branch == "" {
		branch = currentGitBranch()
	}
	if branch == "" {
		branch = "main"
	}

	status, err := provider.PipelineStatus(branch)
	if err != nil {
		return err
	}

	if ciFormat == "json" {
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Human output
	icon := "?"
	switch status.Status {
	case "passing":
		icon = "✓"
	case "failing":
		icon = "✗"
	case "running":
		icon = "⟳"
	}

	fmt.Printf("%s  %s  [%s]", icon, status.Branch, status.Status)
	if status.Duration != "" {
		fmt.Printf("  (%s)", status.Duration)
	}
	fmt.Println()

	if status.CommitSHA != "" {
		fmt.Printf("   commit: %s\n", status.CommitSHA)
	}
	if status.FailedStep != "" {
		fmt.Printf("   failed: %s\n", status.FailedStep)
	}
	if status.RunURL != "" {
		fmt.Printf("   url:    %s\n", status.RunURL)
	}

	return nil
}

func currentGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
