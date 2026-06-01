package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <spec-path>",
	Short: "Sync tracker status back to spec frontmatter",
	Long: `Fetches the current status of the linked tracker issue and updates
the spec's frontmatter to match.

Requires the spec to have a tracker_id in its frontmatter and a tracker
to be configured in hero.json.`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	specPath := args[0]

	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	if s.TrackerID == "" {
		return fmt.Errorf("spec %s has no tracker_id — use 'hero link' or 'hero sync' first", s.Slug)
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	issue, err := t.GetIssue(s.TrackerID)
	if err != nil {
		return fmt.Errorf("fetching issue %s: %w", s.TrackerID, err)
	}

	fmt.Printf("Tracker issue %s: %s\n", issue.ID, issue.Title)
	fmt.Printf("  Tracker status: %s\n", issue.Status)
	fmt.Printf("  Spec status:    %s\n", s.Status)

	// Size mapping sync (tracker → local). Non-destructive: seeds
	// local `size:` only when local is unset; surfaces conflicts as
	// warnings without auto-resolving. No-op when size_mapping is
	// absent (no tracker mapping → never touched).
	sizePlan := tracker.PlanSizePull(t, issue, s.Size)
	switch sizePlan.Action {
	case tracker.SizeSyncSeedLocal:
		content := readSpecContent(specPath)
		content = spec.SetFrontmatterField(content, "size", sizePlan.WriteValue)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not seed local size: %v\n", err)
		} else {
			fmt.Printf("  size: (unset) → %s  (seeded from tracker value %q)\n", sizePlan.WriteValue, sizePlan.TrackerValue)
		}
	case tracker.SizeSyncConflict:
		fmt.Fprintf(os.Stderr, "  Warning: %s\n", sizePlan.Message)
	}

	// Update tracker-prefixed fields if the spec uses them
	if prefix := detectTrackerPrefix(readSpecContent(specPath)); prefix != "" {
		content := readSpecContent(specPath)
		if issue.Status != "" {
			content = spec.SetFrontmatterField(content, prefix+"_status", issue.Status)
		}
		if issue.Priority != "" {
			content = spec.SetFrontmatterField(content, prefix+"_priority", issue.Priority)
		}
		if issue.Severity != "" {
			content = spec.SetFrontmatterField(content, prefix+"_severity", issue.Severity)
		}
		if issue.Assignee != "" {
			content = spec.SetFrontmatterField(content, prefix+"_assignee", issue.Assignee)
		}
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not update tracker fields: %v\n", err)
		}
	}

	// Map tracker status to spec status
	newStatus := mapTrackerStatus(issue.Status, t.Name())
	if newStatus == "" {
		fmt.Printf("  No mapping for tracker status %q — spec unchanged\n", issue.Status)
		return nil
	}

	if spec.Status(newStatus) == s.Status {
		fmt.Println("  Already in sync")
		return nil
	}

	// Update spec frontmatter
	if err := updateFrontmatterStatus(specPath, newStatus); err != nil {
		return fmt.Errorf("updating spec status: %w", err)
	}

	fmt.Printf("  Updated spec status: %s → %s\n", s.Status, newStatus)
	return nil
}

// mapTrackerStatus attempts to map a tracker-native status string to a spec Status string.
// Returns empty string if no mapping is found.
func mapTrackerStatus(trackerStatus, trackerType string) string {
	normalized := strings.ToLower(strings.TrimSpace(trackerStatus))

	// Common mappings across tracker types
	switch normalized {
	case "open", "to do", "todo", "backlog", "new":
		return string(spec.StatusPlanning)
	case "in progress", "in_progress", "started", "doing":
		return string(spec.StatusDelivering)
	case "in review", "in_review", "review":
		return string(spec.StatusInReview)
	case "closed", "done", "resolved", "completed", "complete":
		return string(spec.StatusCompleted)
	case "cancelled", "canceled", "rejected", "won't do", "wont do", "won't fix", "wontfix", "duplicate":
		return string(spec.StatusSuperseded)
	}

	// GitHub-specific
	if trackerType == "github" {
		switch normalized {
		case "open":
			return string(spec.StatusPlanning)
		case "closed":
			return string(spec.StatusCompleted)
		}
	}

	return ""
}
