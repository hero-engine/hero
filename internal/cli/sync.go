package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync specs / state with external trackers and the team server",
	Long: `Bidirectional sync between hero and external systems (Jira, GitHub
Issues, Linear, hero team server).

Subverbs:
  sync connect   set up the external tracker / team server
  sync pull      pull external state into hero (status, comments)
  sync push      push hero state out (creates/updates issues)
  sync jira      bulk push status transitions to Jira
  sync link      link a spec to an existing tracker issue
  sync import    bulk-import open issues as spec scaffolds
  sync evidence  fetch one complete tracker ticket for diagnosis
  sync comment   post a comment to a tracker issue
  sync attach    attach a file to a tracker issue
  sync spec      sync a single spec to the tracker (creates if no tracker_id)
  sync cloud     sync with the hero team server`,
}

var syncSpecCmd = &cobra.Command{
	Use:   "spec <spec-path>",
	Short: "Sync a spec to the configured work tracker",
	Long: `Creates or updates a tracker issue for the given spec.
If the spec has a tracker_id in its frontmatter, updates the existing issue.
Otherwise, creates a new issue and writes the tracker_id back to the spec.`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

var syncJiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Push Hero spec statuses back to Jira via workflow transitions",
	Long: `Walks all specs with a tracker_id and pushes their current Hero status
to Jira using configured workflow transitions (see jira.push_status_transitions
in hero.json). Dry-run by default; add --push to apply.

Examples:
  hero sync jira                        # dry-run
  hero sync jira --push --status delivering # perform one explicit transition cohort`,
	RunE: runSyncJira,
}

var (
	syncJiraPush         bool
	syncJiraStatusFilter string
	syncIntegration      string
)

func init() {
	syncCmd.PersistentFlags().StringVar(&syncIntegration, "integration", "", "stable integration ID (overrides delivery role/default)")
	syncJiraCmd.Flags().BoolVar(&syncJiraPush, "push", false, "actually push transitions (default: dry-run)")
	syncJiraCmd.Flags().StringVar(&syncJiraStatusFilter, "status", "", "only sync specs with this status")

	syncCmd.AddCommand(syncSpecCmd)
	syncCmd.AddCommand(syncJiraCmd)
	syncCmd.AddCommand(syncCloudCmd)

	// Subverbs migrated from top-level commands.
	syncCmd.AddCommand(connectCmd)
	syncCmd.AddCommand(pullCmd)
	syncCmd.AddCommand(linkCmd)
	syncCmd.AddCommand(syncImportCmd) // tracker bulk import (was top-level `import`)
	syncCmd.AddCommand(commentCmd)
	syncCmd.AddCommand(attachCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := selectSyncIntegration(&cfg); err != nil {
		return err
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	specPath := args[0]
	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	if s.TrackerID != "" {
		// Size-mapping push check (local → tracker). Non-destructive:
		// we only inspect the current tracker value and warn on
		// conflict; we never silently overwrite a value a human set.
		// When size_mapping is absent or local size is unset, the
		// plan is a clean noop and stays silent. When the planned
		// write would be a clean push (tracker empty), we surface
		// the plan as a hint — actually writing the value back is
		// part of the existing per-tracker push paths (e.g.
		// hero sync jira) and out of scope for this command.
		if issue, gerr := t.GetIssue(s.TrackerID); gerr == nil {
			sizePlan := tracker.PlanSizePush(t, issue, s.Size)
			switch sizePlan.Action {
			case tracker.SizeSyncConflict:
				// Non-destructive contract: never write on conflict.
				fmt.Fprintf(os.Stderr, "Warning: %s\n", sizePlan.Message)
			case tracker.SizeSyncPushToTracker:
				if uerr := t.UpdateSize(s.TrackerID, s.Size); uerr != nil {
					if errors.Is(uerr, tracker.ErrSizeUpdateNotSupported) {
						fmt.Fprintf(os.Stderr, "Note: %s tracker does not support size updates; skipping.\n", t.Name())
					} else {
						return fmt.Errorf("updating size on %s: %w", s.TrackerID, uerr)
					}
				} else {
					fmt.Printf("Updated %s size for issue %s → %s\n", t.Name(), s.TrackerID, s.Size)
				}
			}
		}
		if err := t.UpdateStatus(s.TrackerID, s.Status); err != nil {
			return fmt.Errorf("updating issue %s: %w", s.TrackerID, err)
		}
		fmt.Printf("Updated %s issue %s — status: %s\n", t.Name(), s.TrackerID, tracker.StatusLabel(s.Status))
		return nil
	}

	issueID, err := t.CreateIssue(s)
	if err != nil {
		return fmt.Errorf("creating issue: %w", err)
	}

	if err := writeTrackerID(specPath, issueID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: created issue %s but could not write tracker_id to spec: %v\n", issueID, err)
		fmt.Fprintf(os.Stderr, "Add tracker_id: %s to the spec frontmatter manually.\n", issueID)
	}

	fmt.Printf("Created %s issue %s for spec %s\n", t.Name(), issueID, s.Slug)
	return nil
}

func runSyncJira(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := selectSyncIntegration(&cfg); err != nil {
		return err
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Tracker == nil || cfg.Tracker.Type != "jira" {
		return fmt.Errorf("hero sync jira requires tracker.type = jira in hero.json")
	}
	if err := validateJiraBulkPush(cfg, syncJiraPush, syncJiraStatusFilter); err != nil {
		return err
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing jira tracker: %w", err)
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if !syncJiraPush {
		fmt.Println("Dry-run mode (add --push to perform transitions):")
	}

	var pushed, skipped, failed int
	for _, s := range specs {
		if s.TrackerID == "" {
			skipped++
			continue
		}
		if syncJiraStatusFilter != "" && string(s.Status) != syncJiraStatusFilter {
			skipped++
			continue
		}

		statusLabel := tracker.StatusLabel(s.Status)

		if !syncJiraPush {
			fmt.Printf("  [dry-run] %s — would push %q to Jira issue %s\n", s.Slug, statusLabel, s.TrackerID)
			pushed++
			continue
		}

		if err := t.UpdateStatus(s.TrackerID, s.Status); err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %s (%s): %v\n", s.Slug, s.TrackerID, err)
			failed++
			continue
		}
		fmt.Printf("  Pushed %q → %s (%s)\n", statusLabel, s.TrackerID, s.Slug)
		pushed++
	}

	if syncJiraPush {
		fmt.Printf("\nPushed: %d, Skipped: %d, Failed: %d\n", pushed, skipped, failed)
	} else {
		fmt.Printf("\n%d spec(s) would be synced. Run with --push to apply.\n", pushed)
	}
	return nil
}

func validateJiraBulkPush(cfg config.Config, push bool, statusFilter string) error {
	if !push {
		return nil
	}
	if statusFilter == "" {
		return fmt.Errorf("bulk Jira --push requires an explicit --status cohort; refusing an unbounded all-spec write")
	}
	if cfg.Jira == nil || cfg.Jira.PushStatusTransitions[statusFilter] == "" {
		return fmt.Errorf("bulk Jira --push requires jira.push_status_transitions[%q]; refusing comment-only fallback across a cohort", statusFilter)
	}
	return nil
}

// writeTrackerID injects tracker_id into a spec's frontmatter.
func writeTrackerID(path, trackerID string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newContent := spec.SetFrontmatterField(string(data), "tracker_id", trackerID)
	return os.WriteFile(path, []byte(newContent), 0o644)
}
