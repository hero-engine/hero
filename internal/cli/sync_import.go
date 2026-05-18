package cli

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var syncImportCmd = &cobra.Command{
	Use:   "import [bugs|features]",
	Short: "Import issues from the configured tracker as spec scaffolds",
	Long: `Fetches open issues from the configured work tracker (GitHub, Jira, Linear)
and creates spec scaffolds for issues that don't already have linked specs.

Each imported issue becomes a spec under .hero/planning/{type}/{slug}/spec.md
with the tracker_id pre-populated, so future syncs are bidirectional.
The spec type (bug or feature) is auto-inferred from the tracker's issue type
field — Bugs become bug specs, Stories/Tasks become feature specs.

An optional positional argument filters by inferred type:
  hero import bugs       — only import issues that map to bug specs
  hero import features   — only import issues that map to feature specs
  hero import            — import all issues (each placed in the correct directory)

The --type flag works the same as the positional argument. If both are given,
the positional argument takes precedence.

An inventory report is generated alongside the import at
.hero/planning/{type}/inventory.md — a single scannable file with issue ID,
severity, age, and description for easy triage.

Presets:
  Define named filter presets in hero.json under "import.presets" to avoid
  retyping common queries:

    "import": {
      "presets": {
        "my-bugs":  { "assignee": "alice", "issue_type": "Bug" },
        "triage":   { "assignee": "unassigned", "priority": "Critical" }
      }
    }

  Then: hero import --preset my-bugs

Filtering:
  Configure default filters in hero.json under "import.filter" or use CLI flags.
  Precedence: --jql > --filter > CLI field flags > --preset > hero.json defaults.

Examples:
  hero import
  hero import bugs
  hero import features
  hero import --preset triage
  hero import bugs --jql "project = PROJ AND type = Bug AND assignee = EMPTY"
  hero import --filter 12345
  hero import --assignee unassigned --issue-type Bug
  hero import bugs --label critical --limit 50 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncImport,
}

var (
	importLabel     string
	importLimit     int
	syncImportDryRun    bool
	syncImportType      string
	importJQL       string
	importFilterID  string
	importAssignee  string
	importIssueType string
	importStatus    string
	importPriority  string
	importOrderBy   string
	importNoReport  bool
	importRefresh   bool
	importPreset    string
)

func init() {
	syncImportCmd.Flags().StringVar(&importLabel, "label", "", "filter issues by label (can be comma-separated)")
	syncImportCmd.Flags().IntVar(&importLimit, "limit", 0, "maximum number of issues to fetch (default from config or 100)")
	syncImportCmd.Flags().BoolVar(&syncImportDryRun, "dry-run", false, "preview imports without creating files")
	syncImportCmd.Flags().StringVar(&syncImportType, "type", "", "filter to only import issues of this spec type: feature, bug (same as positional arg)")

	// Advanced query flags
	syncImportCmd.Flags().StringVar(&importJQL, "jql", "", "raw JQL query (Jira only, overrides all other filters)")
	syncImportCmd.Flags().StringVar(&importFilterID, "filter", "", "saved Jira filter ID (overrides field-level filters)")
	syncImportCmd.Flags().StringVar(&importAssignee, "assignee", "", "filter by assignee (use 'unassigned' for unassigned issues)")
	syncImportCmd.Flags().StringVar(&importIssueType, "issue-type", "", "filter by tracker issue type (e.g. Bug, Story, Task)")
	syncImportCmd.Flags().StringVar(&importStatus, "status", "", "filter by tracker-native status (e.g. New, Open)")
	syncImportCmd.Flags().StringVar(&importPriority, "priority", "", "filter by priority (e.g. Critical, High)")
	syncImportCmd.Flags().StringVar(&importOrderBy, "order-by", "", "sort order (e.g. 'created DESC', 'priority ASC')")
	syncImportCmd.Flags().BoolVar(&importNoReport, "no-report", false, "skip generating the inventory report")
	syncImportCmd.Flags().BoolVar(&importRefresh, "refresh", false, "also sync status of previously imported specs from tracker")
	syncImportCmd.Flags().StringVar(&importPreset, "preset", "", "use a named filter preset from hero.json import.presets")
}

func runSyncImport(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Resolve effective settings: positional arg > CLI flags > config > defaults
	typeFilter := resolveTypeFilter(args, cfg)
	limit := resolveImportLimit(cfg)
	query, err := resolveImportQuery(cfg)
	if err != nil {
		return err
	}

	// Validate type filter if set
	if typeFilter != "" && typeFilter != "feature" && typeFilter != "bug" {
		return fmt.Errorf("import type filter must be 'feature' or 'bug', got %q", typeFilter)
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "none" || cfg.Tracker.Type == "" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	// Fetch issues — use Search if we have any query parameters, otherwise fall back to ListIssues
	var issues []tracker.Issue
	if query.RawQuery != "" || query.FilterID != "" || query.IssueType != "" ||
		query.Assignee != "" || len(query.Labels) > 0 || query.Status != "" ||
		query.Priority != "" || query.OrderBy != "" {
		query.Limit = limit
		if importPreset != "" {
			fmt.Printf("Searching %s with preset %q...\n", t.Name(), importPreset)
		} else {
			fmt.Printf("Searching %s with filters...\n", t.Name())
		}
		if query.RawQuery != "" {
			fmt.Printf("  JQL: %s\n", query.RawQuery)
		} else if query.FilterID != "" {
			fmt.Printf("  Filter ID: %s\n", query.FilterID)
		} else {
			printQuerySummary(query)
		}
		issues, err = t.Search(query)
	} else {
		fmt.Printf("Fetching open issues from %s...\n", t.Name())
		issues, err = t.ListIssues(importLabel, limit)
	}
	if err != nil {
		return fmt.Errorf("fetching issues from %s: %w", t.Name(), err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found matching the query.")
		return nil
	}

	fmt.Printf("Found %d issues.\n\n", len(issues))

	// Find existing specs with tracker_ids to avoid duplicates and check placement
	linkedSpecs := findLinkedSpecs(heroDir)

	var created, skipped, filtered, relocated int
	var importedIssues []tracker.Issue
	typeCounts := map[string]int{} // track how many of each type we imported

	for _, issue := range issues {
		// Infer spec type from the tracker's issue type
		specType := inferSpecType(issue)

		// Apply type filter: skip issues that don't match
		if typeFilter != "" && specType != typeFilter {
			filtered++
			continue
		}

		// If already linked to a spec, check if it needs relocation
		if existing, ok := linkedSpecs[issue.ID]; ok {
			if !syncImportDryRun {
				moved, err := relocateSpec(heroDir, existing, specType)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not relocate %s: %v\n", existing.Slug, err)
				} else if moved {
					fmt.Printf("  Relocated %s: %s → %s/\n", existing.Slug, filepath.Base(filepath.Dir(filepath.Dir(existing.Path))), specType+"s")
					relocated++
				}
			}
			skipped++
			continue
		}

		slug := issueToSlug(issue)
		if syncImportDryRun {
			fmt.Printf("  [dry-run] would create %s spec: %s (%s)\n", specType, slug, issue.ID)
			created++
			typeCounts[specType]++
			importedIssues = append(importedIssues, issue)
			continue
		}

		targetDir, err := specTargetDir(heroDir, specType, slug)
		if err != nil {
			return err
		}
		specPath := filepath.Join(targetDir, "spec.md")

		// Skip if spec already exists by slug
		if _, err := os.Stat(specPath); err == nil {
			skipped++
			continue
		}

		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		content := generateImportedSpec(issue, specType, t.Name(), slug)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing spec: %w", err)
		}

		fmt.Printf("  Created %s: %s (from %s %s)\n", specType, slug, t.Name(), issue.ID)
		created++
		typeCounts[specType]++
		importedIssues = append(importedIssues, issue)
	}

	// Summary line
	summary := fmt.Sprintf("\nImported: %d", created)
	if len(typeCounts) > 1 {
		var parts []string
		for t, c := range typeCounts {
			parts = append(parts, fmt.Sprintf("%d %ss", c, t))
		}
		summary += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	if relocated > 0 {
		summary += fmt.Sprintf(", Relocated: %d", relocated)
	}
	if filtered > 0 {
		summary += fmt.Sprintf(", Filtered out: %d", filtered)
	}
	summary += fmt.Sprintf(", Skipped (already linked): %d, Total fetched: %d\n", skipped, len(issues))
	fmt.Print(summary)

	// Refresh previously imported specs from tracker (bulk pull)
	if importRefresh {
		fmt.Println()
		refreshResults := refreshImportedSpecs(cfg, heroDir, t, issues)
		if refreshResults.total > 0 {
			fmt.Printf("\nRefresh: %d checked, %d updated, %d reassigned, %d resolved, %d errors\n",
				refreshResults.total, refreshResults.updated, refreshResults.reassigned,
				refreshResults.resolved, refreshResults.errors)
		} else {
			fmt.Println("No previously imported specs to refresh.")
		}
	}

	// Generate inventory report
	shouldReport := !importNoReport && resolveInventoryEnabled(cfg)
	if shouldReport && len(issues) > 0 {
		// Determine report path based on what was actually imported
		reportSpecType := typeFilter
		if reportSpecType == "" {
			// Mixed import — use whatever type had the most items, or "all"
			reportSpecType = dominantType(typeCounts)
		}
		reportPath := resolveInventoryPath(cfg, heroDir, reportSpecType)
		if syncImportDryRun {
			fmt.Printf("\n[dry-run] would write inventory report: %s\n", reportPath)
		} else {
			filterDesc := describeQuery(query, t.Name())
			if err := writeInventoryReport(reportPath, issues, filterDesc, reportSpecType); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write inventory report: %v\n", err)
			} else {
				// Make path relative for display
				relPath, _ := filepath.Rel(projectRoot, reportPath)
				if relPath == "" {
					relPath = reportPath
				}
				fmt.Printf("\nInventory report written: %s\n", relPath)
			}
		}
	}

	return nil
}

// resolveTypeFilter returns the spec-type filter from positional arg, CLI flag, or config.
// Returns "" if no filter is set (import all types).
// Positional arg ("bugs"/"features") takes precedence over --type flag.
func resolveTypeFilter(args []string, cfg config.Config) string {
	// Positional arg: "bugs" or "features"
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "bug", "bugs":
			return "bug"
		case "feature", "features":
			return "feature"
		default:
			// Unknown positional arg — treat as empty (will fail validation later if needed)
			return args[0]
		}
	}
	// CLI --type flag
	if syncImportType != "" {
		return syncImportType
	}
	// Config default_type is no longer used as a forced type.
	// If the user configured it, we treat it as a filter preference,
	// but only if they explicitly set it (not the default "feature").
	if cfg.Import != nil && cfg.Import.DefaultType != "" {
		return cfg.Import.DefaultType
	}
	return ""
}

// inferSpecType maps a tracker issue's type to a Hero spec type.
// Bug/Defect → "bug", everything else → "feature".
func inferSpecType(issue tracker.Issue) string {
	switch strings.ToLower(issue.IssueType) {
	case "bug", "defect":
		return "bug"
	case "epic":
		return "initiative"
	case "story", "task", "feature request", "feature", "sub-task", "subtask",
		"improvement", "enhancement", "new feature":
		return "feature"
	default:
		// Unknown or empty issue type — default to feature
		return "feature"
	}
}

// dominantType returns the type with the highest count, or "all" for mixed/empty.
func dominantType(counts map[string]int) string {
	if len(counts) == 0 {
		return "all"
	}
	if len(counts) == 1 {
		for t := range counts {
			return t
		}
	}
	// Find dominant
	best := ""
	bestCount := 0
	for t, c := range counts {
		if c > bestCount {
			best = t
			bestCount = c
		}
	}
	return best
}

// resolveImportLimit returns the effective limit from CLI flags or config.
func resolveImportLimit(cfg config.Config) int {
	if importLimit > 0 {
		return importLimit
	}
	if cfg.Import != nil && cfg.Import.Limit > 0 {
		return cfg.Import.Limit
	}
	return 100
}

// resolveImportQuery builds a SearchQuery from CLI flags, falling back to config defaults.
// Precedence: CLI flags > preset > filter > base_filter (with built-in defaults).
//
// The base_filter provides project-level defaults (e.g. "we work on New unassigned Bugs").
// The filter layer adds additional narrowing on top (e.g. labels, priority).
// A preset replaces the filter layer entirely. CLI flags override any layer.
func resolveImportQuery(cfg config.Config) (tracker.SearchQuery, error) {
	// Start from the base filter (defaults to Bug + unassigned + New)
	var importCfg *config.ImportConfig
	if cfg.Import != nil {
		importCfg = cfg.Import
	}
	base := importCfg.EffectiveBaseFilter()
	query := tracker.SearchQueryFromConfig(base, 0)

	// Layer on the additional filter (non-empty fields override base)
	if importCfg != nil && importCfg.Filter != nil {
		overlay := tracker.SearchQueryFromConfig(importCfg.Filter, 0)
		mergeQuery(&query, overlay)
	}

	// If a preset is specified, it replaces the filter layer (but base still applies underneath)
	if importPreset != "" && importCfg != nil && importCfg.Presets != nil {
		preset, ok := importCfg.Presets[importPreset]
		if !ok {
			// List available presets in the error
			var names []string
			for name := range importCfg.Presets {
				names = append(names, name)
			}
			if len(names) == 0 {
				return query, fmt.Errorf("preset %q not found — no presets configured in hero.json import.presets", importPreset)
			}
			return query, fmt.Errorf("preset %q not found — available presets: %s", importPreset, strings.Join(names, ", "))
		}
		// Preset replaces everything above base — start fresh from base, then apply preset
		query = tracker.SearchQueryFromConfig(base, 0)
		presetQuery := tracker.SearchQueryFromConfig(preset, 0)
		mergeQuery(&query, presetQuery)
	} else if importPreset != "" {
		return query, fmt.Errorf("preset %q not found — no import.presets configured in hero.json", importPreset)
	}

	// CLI flags override everything
	if importJQL != "" {
		query.RawQuery = importJQL
	}
	if importFilterID != "" {
		query.FilterID = importFilterID
	}
	if importIssueType != "" {
		query.IssueType = importIssueType
	}
	if importAssignee != "" {
		query.Assignee = importAssignee
	}
	if importLabel != "" {
		// Merge label flag into labels list
		labels := strings.Split(importLabel, ",")
		for _, l := range labels {
			l = strings.TrimSpace(l)
			if l != "" {
				query.Labels = append(query.Labels, l)
			}
		}
	}
	if importStatus != "" {
		query.Status = importStatus
	}
	if importPriority != "" {
		query.Priority = importPriority
	}
	if importOrderBy != "" {
		query.OrderBy = importOrderBy
	}

	return query, nil
}

// mergeQuery overlays non-empty fields from src onto dst.
func mergeQuery(dst *tracker.SearchQuery, src tracker.SearchQuery) {
	if src.RawQuery != "" {
		dst.RawQuery = src.RawQuery
	}
	if src.FilterID != "" {
		dst.FilterID = src.FilterID
	}
	if src.IssueType != "" {
		dst.IssueType = src.IssueType
	}
	if src.Assignee != "" {
		dst.Assignee = src.Assignee
	}
	if len(src.Labels) > 0 {
		dst.Labels = src.Labels
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.Priority != "" {
		dst.Priority = src.Priority
	}
	if src.OrderBy != "" {
		dst.OrderBy = src.OrderBy
	}
}

// resolveInventoryEnabled returns whether the inventory report should be generated.
func resolveInventoryEnabled(cfg config.Config) bool {
	if cfg.Import != nil {
		return cfg.Import.InventoryEnabled()
	}
	return true
}

// resolveInventoryPath returns the absolute path for the inventory report.
func resolveInventoryPath(cfg config.Config, heroDir, specType string) string {
	var relPath string
	if cfg.Import != nil {
		relPath = cfg.Import.EffectiveInventoryPath(specType)
	} else {
		switch specType {
		case "bug":
			relPath = "bugs/inventory.md"
		case "feature":
			relPath = "features/inventory.md"
		default:
			relPath = "inventory.md"
		}
	}
	return filepath.Join(heroDir, "planning", relPath)
}

// printQuerySummary prints a human-readable summary of the query filters.
func printQuerySummary(query tracker.SearchQuery) {
	if query.IssueType != "" {
		fmt.Printf("  Issue type: %s\n", query.IssueType)
	}
	if query.Assignee != "" {
		fmt.Printf("  Assignee: %s\n", query.Assignee)
	}
	if len(query.Labels) > 0 {
		fmt.Printf("  Labels: %s\n", strings.Join(query.Labels, ", "))
	}
	if query.Status != "" {
		fmt.Printf("  Status: %s\n", query.Status)
	}
	if query.Priority != "" {
		fmt.Printf("  Priority: %s\n", query.Priority)
	}
	if query.OrderBy != "" {
		fmt.Printf("  Order: %s\n", query.OrderBy)
	}
}

// describeQuery returns a human-readable description of the query for the inventory report.
func describeQuery(query tracker.SearchQuery, trackerName string) string {
	if query.RawQuery != "" {
		return fmt.Sprintf("%s query: %s", trackerName, query.RawQuery)
	}
	if query.FilterID != "" {
		return fmt.Sprintf("%s saved filter #%s", trackerName, query.FilterID)
	}

	var parts []string
	if query.IssueType != "" {
		parts = append(parts, fmt.Sprintf("type=%s", query.IssueType))
	}
	if query.Assignee != "" {
		parts = append(parts, fmt.Sprintf("assignee=%s", query.Assignee))
	}
	if len(query.Labels) > 0 {
		parts = append(parts, fmt.Sprintf("labels=%s", strings.Join(query.Labels, ",")))
	}
	if query.Status != "" {
		parts = append(parts, fmt.Sprintf("status=%s", query.Status))
	}
	if query.Priority != "" {
		parts = append(parts, fmt.Sprintf("priority=%s", query.Priority))
	}
	if query.OrderBy != "" {
		parts = append(parts, fmt.Sprintf("order=%s", query.OrderBy))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s open issues", trackerName)
	}
	return fmt.Sprintf("%s: %s", trackerName, strings.Join(parts, ", "))
}

// --- Refresh: bulk sync tracker status back into imported specs ---

type refreshStats struct {
	total      int
	updated    int
	reassigned int
	resolved   int
	errors     int
}

// refreshImportedSpecs walks all specs with tracker_ids and syncs their status
// from the tracker. Handles reassignment detection and resolved issue cleanup.
// fetchedIssues contains issues already retrieved by the search phase; these are
// used directly instead of re-fetching via GetIssue.
func refreshImportedSpecs(cfg config.Config, heroDir string, t tracker.Tracker, fetchedIssues []tracker.Issue) refreshStats {
	var stats refreshStats

	// Build lookup from already-fetched issues to avoid redundant API calls.
	issueByID := map[string]*tracker.Issue{}
	for i := range fetchedIssues {
		issueByID[fetchedIssues[i].ID] = &fetchedIssues[i]
	}

	behavior := &config.RefreshBehavior{}
	if cfg.Import != nil && cfg.Import.OnRefresh != nil {
		behavior = cfg.Import.OnRefresh
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not discover specs: %v\n", err)
		return stats
	}

	// Only refresh work specs (features/bugs) that have a tracker_id and are
	// tagged as imported or are in planning/delivering states
	for _, s := range specs {
		if s.TrackerID == "" {
			continue
		}
		if !s.IsWorkSpec() {
			continue
		}
		// Skip already completed/superseded specs
		if s.Status == spec.StatusCompleted || s.Status == spec.StatusSuperseded {
			continue
		}

		stats.total++

		// Use already-fetched issue if available, otherwise fetch individually.
		issue, ok := issueByID[s.TrackerID]
		if !ok {
			fetched, err := t.GetIssue(s.TrackerID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not fetch %s: %v\n", s.TrackerID, err)
				stats.errors++
				continue
			}
			issue = fetched
		}

		changed := false

		// Compute desired field values from the tracker issue
		if !syncImportDryRun {
			trackerName := t.Name()
			desired := specFieldsFromIssue(*issue, trackerName)

			content := readSpecContent(s.Path)
			updated := false

			// Walk desired fields: set any that are missing or different on the spec.
			for key, want := range desired {
				current := currentSpecFieldValue(s, key)
				if current != want {
					fmt.Printf("  Updating %s: %s %q → %q\n", s.Slug, key, current, want)
					content = spec.SetFrontmatterField(content, key, want)
					updated = true
				}
			}

			if updated {
				_ = os.WriteFile(s.Path, []byte(content), 0o644)
				stats.updated++
				changed = true
			}
		}

		// Check for reassignment
		if behavior.ShouldMarkReassigned() && issue.Assignee != "" && s.ClaimedBy != "" {
			if !strings.EqualFold(issue.Assignee, s.ClaimedBy) {
				fmt.Printf("  Reassigned: %s — was %q, now %q in tracker\n",
					s.Slug, s.ClaimedBy, issue.Assignee)

				if !syncImportDryRun {
					content := spec.SetFrontmatterField(readSpecContent(s.Path), "claimed_by", issue.Assignee)
					content = addTag(content, "reassigned")
					_ = os.WriteFile(s.Path, []byte(content), 0o644)
				}
				stats.reassigned++
				changed = true
			}
		}

		// Check for resolved/closed in tracker
		mappedStatus := mapTrackerStatus(issue.Status, t.Name())
		if mappedStatus == string(spec.StatusCompleted) || mappedStatus == string(spec.StatusSuperseded) {
			action := behavior.ResolvedAction()
			switch action {
			case "mark":
				fmt.Printf("  Resolved: %s — tracker status %q → marking completed\n", s.Slug, issue.Status)
				if !syncImportDryRun {
					_ = updateFrontmatterStatus(s.Path, string(spec.StatusCompleted))
				}
			case "archive":
				fmt.Printf("  Resolved: %s — tracker status %q → archiving to specs/\n", s.Slug, issue.Status)
				if !syncImportDryRun {
					_ = updateFrontmatterStatus(s.Path, string(spec.StatusCompleted))
					specsDir := filepath.Join(heroDir, "specs")
					targetDir := filepath.Join(specsDir, s.Slug)
					if err := os.MkdirAll(targetDir, 0o755); err == nil {
						targetPath := filepath.Join(targetDir, "spec.md")
						if err := os.Rename(s.Path, targetPath); err == nil {
							// Remove the empty parent directory
							parentDir := filepath.Dir(s.Path)
							_ = os.Remove(parentDir)
						}
					}
				}
			case "keep":
				fmt.Printf("  Resolved: %s — tracker status %q (keeping as-is per config)\n", s.Slug, issue.Status)
			}
			stats.resolved++
			changed = true
		} else if behavior.ShouldUpdateStatus() && mappedStatus != "" && !changed {
			// Sync status if it has changed but isn't a resolution
			if spec.Status(mappedStatus) != s.Status {
				fmt.Printf("  Updated: %s — %s → %s (from tracker)\n", s.Slug, s.Status, mappedStatus)
				if !syncImportDryRun {
					_ = updateFrontmatterStatus(s.Path, mappedStatus)
				}
				stats.updated++
			}
		}
	}

	return stats
}

// currentSpecFieldValue returns the current value of a frontmatter field from a
// parsed spec. For hero-level fields it reads from the struct; for tracker-prefixed
// fields it reads from the tracker metadata.
func currentSpecFieldValue(s *spec.Spec, key string) string {
	switch key {
	case "created":
		if s.CreatedAt.IsZero() {
			return ""
		}
		return s.CreatedAt.Format("2006-01-02")
	case "priority":
		return s.Priority
	case "severity":
		return s.Severity
	default:
		// Tracker-prefixed fields: read from parsed tracker metadata
		for _, prefix := range []string{"jira_", "github_", "linear_"} {
			if strings.HasPrefix(key, prefix) {
				field := strings.TrimPrefix(key, prefix)
				switch field {
				case "status":
					return s.TrackerStatus
				case "priority":
					return s.TrackerPriority
				case "severity":
					return s.TrackerSeverity
				case "assignee":
					return s.TrackerAssignee
				}
			}
		}
	}
	return ""
}

// readSpecContent reads a spec file's content as a string, returning empty on error.
func readSpecContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// addTag adds a tag to the frontmatter tags list if not already present.
func addTag(content, tag string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "tags:") {
			// Check if tag already exists
			if strings.Contains(lines[i], tag) {
				return content
			}
			// Add to existing tags list: tags: [imported] -> tags: [imported, reassigned]
			line := lines[i]
			if idx := strings.LastIndex(line, "]"); idx >= 0 {
				lines[i] = line[:idx] + ", " + tag + "]"
			}
			return strings.Join(lines, "\n")
		}
	}

	// No tags line found — inject one
	return spec.SetFrontmatterField(content, "tags", "["+tag+"]")
}

// --- Inventory report generation ---

// writeInventoryReport generates a scannable markdown report of fetched issues.
// Follows the issue-list-report skill format for consistency with the issue-tracker agent.
func writeInventoryReport(reportPath string, issues []tracker.Issue, filterDesc, specType string) error {
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02 15:04 MST")

	// Count stats
	var criticalCount, highCount, mediumCount, lowCount, newTodayCount int
	priorityCounts := map[string]int{}
	for _, issue := range issues {
		p := strings.ToLower(issue.Priority)
		priorityCounts[p]++
		switch p {
		case "critical", "highest", "blocker":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low", "lowest", "trivial":
			lowCount++
		}

		if isCreatedToday(issue.CreatedAt) {
			newTodayCount++
		}
	}

	var sb strings.Builder

	// Dynamic heading based on what was imported
	heading := "Issue Inventory"
	switch specType {
	case "bug":
		heading = "Bug Inventory"
	case "feature":
		heading = "Feature Inventory"
	}
	sb.WriteString("# " + heading + "\n\n")
	fmt.Fprintf(&sb, "Generated: %s | Total issues: %d\n", dateStr, len(issues))
	fmt.Fprintf(&sb, "Filter: %s\n\n", filterDesc)

	sb.WriteString("## Highlights\n\n")
	fmt.Fprintf(&sb, "- Total issues: %d\n", len(issues))
	fmt.Fprintf(&sb, "- New today: %d\n", newTodayCount)
	if criticalCount > 0 {
		fmt.Fprintf(&sb, "- Critical: %d\n", criticalCount)
	}
	if highCount > 0 {
		fmt.Fprintf(&sb, "- High: %d\n", highCount)
	}
	if mediumCount > 0 {
		fmt.Fprintf(&sb, "- Medium: %d\n", mediumCount)
	}
	if lowCount > 0 {
		fmt.Fprintf(&sb, "- Low: %d\n", lowCount)
	}
	sb.WriteString("\n")

	sb.WriteString("## Issue List\n\n")

	for _, issue := range issues {
		fmt.Fprintf(&sb, "### %s - %s\n", issue.ID, issue.Title)

		severity := issue.Priority
		if issue.Severity != "" {
			severity = issue.Severity
		}
		if severity == "" {
			severity = "unset"
		}
		reporter := issue.Reporter
		if reporter == "" {
			reporter = "unknown"
		}
		createdDate := formatCreatedDate(issue.CreatedAt)
		age := formatIssueAge(issue.CreatedAt)

		fmt.Fprintf(&sb, "Severity: %s | Reporter: %s | Created: %s | Age: %s\n",
			severity, reporter, createdDate, age)

		// Show any additional custom fields that have values.
		if len(issue.CustomFields) > 0 {
			for name, val := range issue.CustomFields {
				// Skip severity-like fields already shown in the Severity line above.
				isSeverity := false
				for _, sn := range severityLikeNames {
					if name == sn {
						isSeverity = true
						break
					}
				}
				if !isSeverity {
					fmt.Fprintf(&sb, "%s: %s\n", capitalizeFieldName(name), val)
				}
			}
		}

		// One or two sentence description
		desc := issue.Description
		if desc != "" {
			// Take first 2 sentences or 200 chars, whichever is shorter
			desc = summarizeDescription(desc, 200)
			fmt.Fprintf(&sb, "%s\n", desc)
		}

		sb.WriteString("\n")
	}

	return os.WriteFile(reportPath, []byte(sb.String()), 0o644)
}

// summarizeDescription extracts a brief summary from a longer description.
func summarizeDescription(desc string, maxLen int) string {
	// Strip leading markdown headers and blank lines
	lines := strings.Split(desc, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	desc = strings.Join(cleaned, " ")

	if len(desc) <= maxLen {
		return desc
	}

	// Try to break at sentence boundary
	truncated := desc[:maxLen]
	for _, sep := range []string{". ", "! ", "? "} {
		if idx := strings.LastIndex(truncated, sep); idx > maxLen/3 {
			return truncated[:idx+1]
		}
	}

	// Break at word boundary
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return truncated[:idx] + "..."
	}

	return truncated + "..."
}

// isCreatedToday checks if a date string represents today.
func isCreatedToday(createdAt string) bool {
	if createdAt == "" {
		return false
	}
	// Try common formats
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, createdAt); err == nil {
			now := time.Now()
			return t.Year() == now.Year() && t.YearDay() == now.YearDay()
		}
	}
	return false
}

// formatCreatedDate formats a tracker date string for display.
func formatCreatedDate(createdAt string) string {
	if createdAt == "" {
		return "unknown"
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, createdAt); err == nil {
			return t.Format("2006-01-02")
		}
	}
	// Return raw if we can't parse
	if len(createdAt) > 10 {
		return createdAt[:10]
	}
	return createdAt
}

// formatIssueAge returns a human-readable age string like "3 days" or "2 weeks".
func formatIssueAge(createdAt string) string {
	if createdAt == "" {
		return "unknown"
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, createdAt); err == nil {
			days := int(math.Floor(time.Since(t).Hours() / 24))
			switch {
			case days == 0:
				return "today"
			case days == 1:
				return "1 day"
			case days < 7:
				return fmt.Sprintf("%d days", days)
			case days < 14:
				return "1 week"
			case days < 30:
				return fmt.Sprintf("%d weeks", days/7)
			case days < 60:
				return "1 month"
			default:
				return fmt.Sprintf("%d months", days/30)
			}
		}
	}
	return "unknown"
}

// --- Spec generation (unchanged logic, richer templates) ---

// findLinkedTrackerIDs discovers all specs that already have a tracker_id.
func findLinkedTrackerIDs(heroDir string) map[string]bool {
	ids := map[string]bool{}
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return ids
	}
	for _, s := range specs {
		if s.TrackerID != "" {
			ids[s.TrackerID] = true
		}
	}
	return ids
}

// findLinkedSpecs returns specs keyed by their tracker ID for relocation checks.
func findLinkedSpecs(heroDir string) map[string]*spec.Spec {
	m := map[string]*spec.Spec{}
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return m
	}
	for _, s := range specs {
		if s.TrackerID != "" {
			m[s.TrackerID] = s
		}
	}
	return m
}

// relocateSpec moves a spec from its current directory to the correct type
// directory if it was previously imported under the wrong type. Returns true
// if the spec was moved.
func relocateSpec(heroDir string, s *spec.Spec, correctType string) (bool, error) {
	correctDir, err := specTargetDir(heroDir, correctType, s.Slug)
	if err != nil {
		return false, err
	}
	correctPath := filepath.Join(correctDir, "spec.md")

	// Already in the right place
	if s.Path == correctPath {
		return false, nil
	}

	// Move: create target dir, rename file, remove old empty dir
	if err := os.MkdirAll(correctDir, 0o755); err != nil {
		return false, fmt.Errorf("creating target directory: %w", err)
	}
	if err := os.Rename(s.Path, correctPath); err != nil {
		return false, fmt.Errorf("moving spec: %w", err)
	}
	// Clean up empty parent directory
	oldDir := filepath.Dir(s.Path)
	_ = os.Remove(oldDir) // ignore error if not empty

	return true, nil
}

// issueToSlug converts a tracker issue to a kebab-case slug prefixed with
// the issue key. Example: PROJ-333 "Fix login timeout" → "proj-333-fix-login-timeout".
func issueToSlug(issue tracker.Issue) string {
	// Build the key prefix (lowercase the tracker ID, e.g. "PROJ-333" → "proj-333")
	prefix := strings.ToLower(issue.ID)

	// Clean the title: remove [feature], [bug] prefixes that hero sync adds
	title := issue.Title
	for _, pfx := range []string{"[feature] ", "[bug] ", "[initiative] "} {
		title = strings.TrimPrefix(strings.ToLower(title), pfx)
	}

	// Convert title to slug fragment
	titleSlug := strings.ToLower(title)
	titleSlug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '_' || r == '.' {
			return '-'
		}
		return -1
	}, titleSlug)

	// Clean up multiple hyphens
	for strings.Contains(titleSlug, "--") {
		titleSlug = strings.ReplaceAll(titleSlug, "--", "-")
	}
	titleSlug = strings.Trim(titleSlug, "-")

	// Combine: prefix-title, truncate the title portion to keep total reasonable
	maxTitleLen := 50 - len(prefix) - 1 // -1 for the joining hyphen
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}
	if len(titleSlug) > maxTitleLen {
		titleSlug = titleSlug[:maxTitleLen]
		titleSlug = strings.TrimRight(titleSlug, "-")
	}

	if titleSlug == "" {
		return prefix
	}

	return prefix + "-" + titleSlug
}

// specFieldsFromIssue computes the desired frontmatter field values for a spec
// based on the current tracker issue state. Both the initial import and the
// refresh path use this so the logic for which fields to write lives in one place.
// The returned map contains both hero-level and tracker-prefixed keys.
func specFieldsFromIssue(issue tracker.Issue, trackerName string) map[string]string {
	fields := map[string]string{}
	prefix := trackerName

	// Hero-level fields
	if issue.CreatedAt != "" {
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05.000Z"} {
			if parsed, err := time.Parse(layout, issue.CreatedAt); err == nil {
				fields["created"] = parsed.Format("2006-01-02")
				break
			}
		}
	}
	if issue.Priority != "" {
		fields["priority"] = strings.ToLower(issue.Priority)
	}
	if issue.Severity != "" {
		fields["severity"] = strings.ToLower(issue.Severity)
	}

	// Tracker-prefixed fields
	if issue.Status != "" {
		fields[prefix+"_status"] = issue.Status
	}
	if issue.Priority != "" {
		fields[prefix+"_priority"] = issue.Priority
	}
	if issue.Severity != "" {
		fields[prefix+"_severity"] = issue.Severity
	}
	if issue.Assignee != "" {
		fields[prefix+"_assignee"] = issue.Assignee
	}

	return fields
}

// generateImportedSpec creates spec content for an imported tracker issue.
// trackerName is the tracker type (e.g. "jira", "github", "linear") used to
// prefix tracker-specific fields in the frontmatter, keeping them visually
// distinct from Hero's own fields.
func generateImportedSpec(issue tracker.Issue, specType, trackerName, slug string) string {
	title := issue.Title
	// Strip hero-added prefixes for cleaner titles
	for _, prefix := range []string{"[feature] ", "[bug] ", "[Feature] ", "[Bug] "} {
		title = strings.TrimPrefix(title, prefix)
	}

	fields := specFieldsFromIssue(issue, trackerName)
	prefix := trackerName

	// created: use tracker date from fields, or fall back to today
	date := time.Now().Format("2006-01-02")
	if d, ok := fields["created"]; ok {
		date = d
	}

	// --- Hero section ---
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString("# Hero\n")
	fmt.Fprintf(&fm, "title: %q\n", title)
	fmt.Fprintf(&fm, "slug: %s\n", slug)
	fmt.Fprintf(&fm, "type: %s\n", specType)
	fm.WriteString("status: planning\n")
	fmt.Fprintf(&fm, "tracker_id: %s\n", issue.ID)
	fmt.Fprintf(&fm, "created: %s\n", date)
	if v, ok := fields["priority"]; ok {
		fmt.Fprintf(&fm, "priority: %s\n", v)
	}
	if v, ok := fields["severity"]; ok {
		fmt.Fprintf(&fm, "severity: %s\n", v)
	}
	fm.WriteString("tags: [imported]\n")

	// --- Tracker section ---
	fmt.Fprintf(&fm, "\n# %s\n", capitalizeFieldName(prefix))
	fmt.Fprintf(&fm, "%s_id: %s\n", prefix, issue.ID)
	if v, ok := fields[prefix+"_status"]; ok {
		fmt.Fprintf(&fm, "%s_status: %s\n", prefix, v)
	}
	if v, ok := fields[prefix+"_priority"]; ok {
		fmt.Fprintf(&fm, "%s_priority: %s\n", prefix, v)
	}
	if v, ok := fields[prefix+"_severity"]; ok {
		fmt.Fprintf(&fm, "%s_severity: %s\n", prefix, v)
	}
	if issue.IssueType != "" {
		fmt.Fprintf(&fm, "%s_type: %s\n", prefix, issue.IssueType)
	}
	if issue.Assignee != "" {
		fmt.Fprintf(&fm, "%s_assignee: %s\n", prefix, issue.Assignee)
	}
	if issue.URL != "" {
		fmt.Fprintf(&fm, "%s_url: %s\n", prefix, issue.URL)
	}
	fm.WriteString("---\n\n")

	// --- Body ---
	var body strings.Builder
	fmt.Fprintf(&body, "# %s\n\n", title)

	// Description: 2-3 sentence summary from tracker
	if issue.Description != "" {
		desc := summarizeDescription(issue.Description, 400)
		fmt.Fprintf(&body, "> %s\n\n", desc)
	}

	// Template sections
	switch specType {
	case "bug":
		body.WriteString("## Problem\n\n")
		body.WriteString("<!-- Imported from tracker. Describe the bug. -->\n\n")
		body.WriteString("## Root Cause\n\n")
		body.WriteString("<!-- To be investigated. -->\n\n")
		body.WriteString("## Fix\n\n")
		body.WriteString("<!-- Design the fix. -->\n\n")
		body.WriteString("## Changes\n\n")
		body.WriteString("<!-- Files to modify. -->\n")
	default:
		body.WriteString("## Goal\n\n")
		body.WriteString("<!-- Imported from tracker. Describe the goal. -->\n\n")
		body.WriteString("## Design\n\n")
		body.WriteString("<!-- To be designed. -->\n\n")
		body.WriteString("## Changes\n\n")
		body.WriteString("<!-- Files to modify. -->\n\n")
		body.WriteString("## Acceptance Criteria\n\n")
		body.WriteString("<!-- Define done. -->\n")
	}

	return fm.String() + body.String()
}

// severityLikeNames matches the auto-discovered severity field names from the tracker package.
// Used to avoid double-displaying them in reports (they're already shown in the Severity line).
var severityLikeNames = []string{
	"severity",
	"criticality",
	"impact",
	"bug severity",
	"defect severity",
	"issue severity",
}

// capitalizeFieldName converts a lowercase field name to title case for display.
func capitalizeFieldName(name string) string {
	if name == "" {
		return name
	}
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
