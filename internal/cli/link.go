package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var linkForce bool

var linkCmd = &cobra.Command{
	Use:   "link <spec> <issue-id>",
	Short: "Link a spec to an existing tracker issue",
	Long: `Associates a spec with an existing tracker issue by writing tracker_id
to the spec's frontmatter. Use this when the issue was created outside of Hero
(e.g. from a Jira board or GitHub issue template).

The spec argument accepts a spec.md path, a spec directory, or a bare slug.
Use --force to re-point a spec that is already linked (e.g. during a tracker
migration); the new issue is still verified to exist before the overwrite.`,
	Args: cobra.ExactArgs(2),
	RunE: runLink,
}

func init() {
	linkCmd.Flags().BoolVar(&linkForce, "force", false, "overwrite an existing tracker_id (re-point a migrated spec)")
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

	specArg := args[0]
	issueID := args[1]

	// Resolve the spec by directory, file path, or slug.
	s, err := resolveSpec(specArg, heroDir)
	if err != nil {
		return fmt.Errorf("resolving spec %s: %w", specArg, err)
	}

	oldTrackerID := s.TrackerID
	if oldTrackerID != "" && !linkForce {
		return fmt.Errorf("spec %s is already linked to tracker issue %s (use --force to re-point)", s.Slug, oldTrackerID)
	}

	// Verify the (new) issue exists in the tracker — always, even under --force.
	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("verifying issue %s: %w", issueID, err)
	}

	// Write tracker_id to the resolved on-disk spec. For a three-file layout
	// s.Path is a virtual <dir>/spec.md, so target requirements.md instead.
	writePath := s.Path
	if s.ThreeFile {
		writePath = filepath.Join(filepath.Dir(s.Path), "requirements.md")
	}
	if err := writeTrackerID(writePath, issueID); err != nil {
		return fmt.Errorf("writing tracker_id to spec: %w", err)
	}

	if oldTrackerID != "" && linkForce {
		fmt.Printf("Re-pointed spec %s: %s → %s\n", s.Slug, oldTrackerID, issueID)
	}
	fmt.Printf("Linked spec %s to %s issue %s (%s)\n", s.Slug, t.Name(), issue.ID, issue.Title)
	return nil
}
