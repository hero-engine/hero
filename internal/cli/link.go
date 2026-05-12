package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <spec-path> <issue-id>",
	Short: "Link a spec to an existing tracker issue",
	Long: `Associates a spec with an existing tracker issue by writing tracker_id
to the spec's frontmatter. Use this when the issue was created outside of Hero
(e.g. from a Jira board or GitHub issue template).`,
	Args: cobra.ExactArgs(2),
	RunE: runLink,
}

func runLink(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	specPath := args[0]
	issueID := args[1]

	// Parse the spec to verify it exists
	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	if s.TrackerID != "" {
		return fmt.Errorf("spec %s is already linked to tracker issue %s", s.Slug, s.TrackerID)
	}

	// Optionally verify the issue exists in the tracker
	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("verifying issue %s: %w", issueID, err)
	}

	// Write tracker_id to the spec frontmatter
	if err := writeTrackerID(specPath, issueID); err != nil {
		return fmt.Errorf("writing tracker_id to spec: %w", err)
	}

	fmt.Printf("Linked spec %s to %s issue %s (%s)\n", s.Slug, t.Name(), issue.ID, issue.Title)
	return nil
}
