package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Sprint-related commands",
	Long:  `Commands for loading and managing sprints from external trackers.`,
}

var sprintLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load a sprint from Jira or Linear and create Hero spec stubs",
	Long: `Fetches a sprint from an external tracker (Jira or Linear) and creates
Hero spec stubs for each work item. Deduplicates against existing specs by tracker_id.

Examples:
  hero sprint load --tracker jira --sprint "Sprint 42"
  hero sprint load --tracker jira --sprint 42
  hero sprint load --tracker jira --board "Engineering"
  hero sprint load --tracker linear --iteration "2026-04-14"
  hero sprint load --tracker jira --sprint 42 --dry-run`,
	RunE: runSprintLoad,
}

var (
	sprintLoadTracker   string
	sprintLoadSprint    string
	sprintLoadBoard     string
	sprintLoadIteration string
	sprintLoadDryRun    bool
	sprintLoadUpdate    bool
)

func init() {
	sprintLoadCmd.Flags().StringVar(&sprintLoadTracker, "tracker", "", "tracker type: jira or linear (default: from hero.json)")
	sprintLoadCmd.Flags().StringVar(&sprintLoadSprint, "sprint", "", "sprint name or ID to load (Jira)")
	sprintLoadCmd.Flags().StringVar(&sprintLoadBoard, "board", "", "board name or ID — loads the active sprint (Jira)")
	sprintLoadCmd.Flags().StringVar(&sprintLoadIteration, "iteration", "", "iteration date or name (Linear)")
	sprintLoadCmd.Flags().BoolVar(&sprintLoadDryRun, "dry-run", false, "preview what would be created without writing files")
	sprintLoadCmd.Flags().BoolVar(&sprintLoadUpdate, "update", false, "refresh frontmatter on existing specs from current tracker state")

	sprintCmd.AddCommand(sprintLoadCmd)

	// Subverbs migrated from top-level commands:
	//   estimate (was `hero cost`)     — effort estimation
	//   status   (was `hero pulse`)    — sprint narrative
	//   retro    (was `hero replay`)   — post-spec post-mortem
	//   report   (was `hero report`)   — manager snapshot HTML
	//   velocity (was `hero velocity`) — agent contribution metrics
	sprintCmd.AddCommand(costCmd)
	sprintCmd.AddCommand(pulseCmd)
	sprintCmd.AddCommand(replayCmd)
	sprintCmd.AddCommand(reportCmd)
	sprintCmd.AddCommand(velocityCmd)
}

func runSprintLoad(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Determine tracker config — CLI flag overrides hero.json
	trackerCfg := cfg.Tracker
	if trackerCfg == nil {
		trackerCfg = &config.TrackerConfig{}
	}
	if sprintLoadTracker != "" {
		trackerCfg = &config.TrackerConfig{
			Type:     sprintLoadTracker,
			Project:  trackerCfg.Project,
			TokenEnv: trackerCfg.TokenEnv,
			BaseURL:  trackerCfg.BaseURL,
		}
	}

	if trackerCfg.Type == "" || trackerCfg.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json or use --tracker")
	}

	loader, err := tracker.NewSprintLoader(trackerCfg, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing sprint loader: %w", err)
	}

	// Validate that we have at least one sprint reference
	if sprintLoadSprint == "" && sprintLoadBoard == "" && sprintLoadIteration == "" {
		return fmt.Errorf("provide --sprint, --board, or --iteration to specify which sprint to load")
	}

	fmt.Printf("Loading sprint from %s...\n", trackerCfg.Type)
	var items []tracker.SprintItem
	var sprintInfo *tracker.SprintInfo

	switch {
	case sprintLoadIteration != "" || trackerCfg.Type == "linear":
		items, sprintInfo, err = loader.LoadIteration(sprintLoadIteration)
	case sprintLoadBoard != "":
		items, sprintInfo, err = loader.LoadActiveSprint(sprintLoadBoard)
	default:
		items, sprintInfo, err = loader.LoadSprint(sprintLoadSprint)
	}

	if err != nil {
		return fmt.Errorf("loading sprint: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No items found in sprint.")
		return nil
	}

	fmt.Printf("Found %d items in %q\n", len(items), sprintInfo.Name)

	// Find existing specs with tracker_ids
	existingIDs := findLinkedTrackerIDs(heroDir)

	var created, updated, skipped int
	var createdSlugs []string

	for _, item := range items {
		slug := sprintItemToSlug(item)
		specType := item.Type
		if specType == "" {
			specType = "feature"
		}

		targetDir, err := specTargetDir(heroDir, specType, slug)
		if err != nil {
			return err
		}
		specPath := filepath.Join(targetDir, "spec.md")

		// Check for duplicate by tracker_id
		if existingIDs[item.ID] {
			if sprintLoadUpdate {
				if err := updateSprintStub(specPath, item); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not update %s: %v\n", slug, err)
				} else {
					fmt.Printf("  Updated %s (%s)\n", slug, item.ID)
					updated++
				}
			} else {
				skipped++
			}
			continue
		}

		if sprintLoadDryRun {
			fmt.Printf("  [dry-run] would create %s: %s (%s)\n", specType, slug, item.ID)
			created++
			createdSlugs = append(createdSlugs, slug)
			continue
		}

		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", slug, err)
		}

		// Skip if spec already exists by slug (and not in our ID map — edge case)
		if _, err := os.Stat(specPath); err == nil && !sprintLoadUpdate {
			skipped++
			continue
		}

		content := generateSprintStub(item, sprintInfo.Name, trackerCfg.Type)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing spec %s: %w", slug, err)
		}

		fmt.Printf("  Created %s: %s (%s)\n", specType, slug, item.ID)
		created++
		createdSlugs = append(createdSlugs, slug)
	}

	fmt.Printf("\nCreated: %d, Updated: %d, Skipped (already imported): %d\n", created, updated, skipped)

	if !sprintLoadDryRun && created > 0 && sprintInfo != nil {
		if err := writeSprintNote(heroDir, sprintInfo, createdSlugs, items); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not write sprint note: %v\n", err)
		} else {
			noteSlug := sprintNoteSlug(sprintInfo)
			fmt.Printf("\nSprint plan note written: .hero/knowledge/notes/%s/spec.md\n", noteSlug)
			fmt.Printf("Run: hero index   (to make all imported specs searchable)\n")
			fmt.Printf("Then: /sprint initiative: %s   (to sequence and plan)\n", noteSlug)
		}
	}

	// Ingest the sprint into the knowledge graph (Issue/Sprint/Person
	// nodes plus belongs_to / assigned_to / blocks edges). Best-effort
	// — never fails the sprint load on a graph error.
	if !sprintLoadDryRun && len(items) > 0 {
		if err := writeSprintToGraph(heroDir, projectRoot, items, sprintInfo); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: graph ingest failed: %v\n", err)
		}
	}

	return nil
}

// writeSprintToGraph opens the graph and writes the sprint subgraph.
// Separated from runSprintLoad so it can be unit-tested independently
// and so a graph failure can't break the sprint-load workflow.
func writeSprintToGraph(heroDir, projectRoot string, items []tracker.SprintItem, info *tracker.SprintInfo) error {
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := filepath.Base(projectRoot)
	summary, err := tracker.WriteSprintGraph(items, info, repoKey, store)
	if err != nil {
		return fmt.Errorf("writing sprint graph: %w", err)
	}
	fmt.Printf("Graph: %d sprints, %d issues, %d persons, %d edges\n",
		summary.Sprints, summary.Issues, summary.Persons, summary.Edges)
	return nil
}

// generateSprintStub creates spec content for a sprint item.
func generateSprintStub(item tracker.SprintItem, sprintName, trackerName string) string {
	date := time.Now().Format("2006-01-02")
	prefix := trackerName // e.g. "jira", "github", "linear"

	tags := []string{}
	if item.SprintName != "" {
		tags = append(tags, slugifyTag(item.SprintName))
	}
	for _, l := range item.Labels {
		if l != "" {
			tags = append(tags, slugifyTag(l))
		}
	}
	// Deduplicate
	seen := map[string]bool{}
	var dedupTags []string
	for _, t := range tags {
		if !seen[t] {
			seen[t] = true
			dedupTags = append(dedupTags, t)
		}
	}

	// --- Hero section ---
	var fm strings.Builder
	fm.WriteString("---\n")
	fmt.Fprintf(&fm, "title: %q\n", item.Title)
	fmt.Fprintf(&fm, "type: %s\n", item.Type)
	fm.WriteString("status: planning\n")
	fmt.Fprintf(&fm, "tracker_id: %s\n", item.ID)
	if item.Assignee != "" {
		fmt.Fprintf(&fm, "claimed_by: %s\n", item.Assignee)
	}
	if len(dedupTags) > 0 {
		fmt.Fprintf(&fm, "tags: [%s]\n", strings.Join(dedupTags, ", "))
	}
	if sprintName != "" {
		fmt.Fprintf(&fm, "sprint: %q\n", sprintName)
	}
	fmt.Fprintf(&fm, "created: %s\n", date)
	if item.Priority != "" {
		fmt.Fprintf(&fm, "priority: %s\n", strings.ToLower(item.Priority))
	}
	if item.Severity != "" {
		fmt.Fprintf(&fm, "severity: %s\n", strings.ToLower(item.Severity))
	}

	// --- Tracker section ---
	fmt.Fprintf(&fm, "\n# %s\n", capitalizeFieldName(prefix))
	fmt.Fprintf(&fm, "%s_id: %s\n", prefix, item.ID)
	if item.Status != "" {
		fmt.Fprintf(&fm, "%s_status: %s\n", prefix, item.Status)
	}
	if item.Priority != "" {
		fmt.Fprintf(&fm, "%s_priority: %s\n", prefix, item.Priority)
	}
	if item.Severity != "" {
		fmt.Fprintf(&fm, "%s_severity: %s\n", prefix, item.Severity)
	}
	if item.Type != "" {
		fmt.Fprintf(&fm, "%s_type: %s\n", prefix, item.Type)
	}
	if item.Assignee != "" {
		fmt.Fprintf(&fm, "%s_assignee: %s\n", prefix, item.Assignee)
	}
	if item.URL != "" {
		fmt.Fprintf(&fm, "%s_url: %s\n", prefix, item.URL)
	}

	// Add epic/parent relation
	if item.EpicID != "" {
		fm.WriteString("relations:\n")
		epicSlug := issueToSlug(tracker.Issue{ID: item.EpicID, Title: item.EpicTitle})
		fmt.Fprintf(&fm, "  - target: %s\n    kind: parent\n", epicSlug)
		for _, link := range item.LinkedIDs {
			linkSlug := link.ID
			fmt.Fprintf(&fm, "  - target: %s\n    kind: %s\n", linkSlug, link.LinkType)
		}
	} else if len(item.LinkedIDs) > 0 {
		fm.WriteString("relations:\n")
		for _, link := range item.LinkedIDs {
			fmt.Fprintf(&fm, "  - target: %s\n    kind: %s\n", link.ID, link.LinkType)
		}
	}
	fm.WriteString("---\n\n")

	// --- Body ---
	var body strings.Builder

	// Description summary from tracker
	if item.Description != "" {
		desc := summarizeDescription(item.Description, 400)
		fmt.Fprintf(&body, "> %s\n\n", desc)
	}

	fmt.Fprintf(&body, "## Goal\n\n")
	if item.Description != "" {
		body.WriteString(item.Description)
		body.WriteString("\n\n")
	} else {
		body.WriteString("<!-- Imported from tracker. Describe the goal. -->\n\n")
	}

	if item.AcceptanceCriteria != "" {
		body.WriteString("## Acceptance Criteria\n\n")
		body.WriteString(item.AcceptanceCriteria)
		body.WriteString("\n\n")
	}

	body.WriteString("## Changes\n\n<!-- Files to modify. -->\n\n")

	body.WriteString("## Notes\n\n")
	fmt.Fprintf(&body, "*Spec stub imported from %s. Run `/design` to flesh this out into a full Hero spec before delivery.*\n", item.ID)

	return fm.String() + body.String()
}

// updateSprintStub refreshes only the tracker-sourced frontmatter fields on an existing spec stub.
func updateSprintStub(specPath string, item tracker.SprintItem) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Update Hero-level fields
	if item.Assignee != "" {
		content = spec.SetFrontmatterField(content, "claimed_by", item.Assignee)
	}

	// Update tracker-prefixed fields if they exist in the spec.
	// Detect which tracker prefix is used by scanning frontmatter.
	if prefix := detectTrackerPrefix(content); prefix != "" {
		if item.Priority != "" {
			content = spec.SetFrontmatterField(content, prefix+"_priority", item.Priority)
		}
		if item.Severity != "" {
			content = spec.SetFrontmatterField(content, prefix+"_severity", item.Severity)
		}
		if item.Status != "" {
			content = spec.SetFrontmatterField(content, prefix+"_status", item.Status)
		}
		if item.Assignee != "" {
			content = spec.SetFrontmatterField(content, prefix+"_assignee", item.Assignee)
		}
	} else {
		// Legacy format — update old-style fields
		if item.Priority != "" {
			content = spec.SetFrontmatterField(content, "priority", item.Priority)
		}
	}

	return os.WriteFile(specPath, []byte(content), 0o644)
}

// detectTrackerPrefix scans spec frontmatter for a tracker-prefixed field
// (jira_*, github_*, linear_*) and returns the prefix, or empty if none found.
func detectTrackerPrefix(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		for _, prefix := range []string{"jira_", "github_", "linear_"} {
			if strings.HasPrefix(trimmed, prefix) {
				return strings.TrimSuffix(prefix, "_")
			}
		}
	}
	return ""
}

// writeSprintNote writes a sprint plan note to the knowledge/notes directory.
func writeSprintNote(heroDir string, info *tracker.SprintInfo, createdSlugs []string, allItems []tracker.SprintItem) error {
	noteSlug := sprintNoteSlug(info)
	noteDir := filepath.Join(heroDir, "knowledge", "notes", noteSlug)
	if err := os.MkdirAll(noteDir, 0o755); err != nil {
		return err
	}

	date := time.Now().Format("2006-01-02")
	sprintTag := slugifyTag(info.Name)

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %q\n", info.Name+" — "+date)
	sb.WriteString("type: note\n")
	fmt.Fprintf(&sb, "tags: [sprint, %s]\n", sprintTag)
	fmt.Fprintf(&sb, "created: %s\n", date)
	sb.WriteString("---\n\n")

	fmt.Fprintf(&sb, "# %s\n\n", info.Name)
	fmt.Fprintf(&sb, "Imported from tracker on %s.\n\n", date)

	if info.Goal != "" {
		fmt.Fprintf(&sb, "## Sprint Goal\n\n%s\n\n", info.Goal)
	}

	sb.WriteString("## Items\n\n")
	sb.WriteString("| Spec | Type | Assignee | Priority | Tracker |\n")
	sb.WriteString("|---|---|---|---|---|\n")

	slugMap := map[string]string{}
	for _, item := range allItems {
		slugMap[item.ID] = sprintItemToSlug(item)
	}

	for _, item := range allItems {
		assignee := item.Assignee
		if assignee == "" {
			assignee = "—"
		}
		priority := item.Priority
		if priority == "" {
			priority = "—"
		}
		slug := slugMap[item.ID]
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n", slug, item.Type, assignee, priority, item.ID)
	}

	sb.WriteString("\n## Ready for Delivery\n\n")
	sb.WriteString("Specs that already have a `/design` pass: none — run `/design` on each before `/deliver`.\n\n")
	if len(createdSlugs) > 0 {
		sb.WriteString("Newly created stubs:\n")
		for _, s := range createdSlugs {
			fmt.Fprintf(&sb, "- %s\n", s)
		}
	}

	notePath := filepath.Join(noteDir, "spec.md")
	return os.WriteFile(notePath, []byte(sb.String()), 0o644)
}

// sprintItemToSlug converts a SprintItem to a kebab-case slug.
func sprintItemToSlug(item tracker.SprintItem) string {
	return issueToSlug(tracker.Issue{ID: item.ID, Title: item.Title})
}

// sprintNoteSlug creates a slug for the sprint plan note.
func sprintNoteSlug(info *tracker.SprintInfo) string {
	date := time.Now().Format("2006-01-02")
	if info == nil || info.Name == "" {
		return "sprint-" + date
	}
	name := slugifyTag(info.Name)
	return "sprint-" + name + "-" + date
}

// slugifyTag converts a display string to a lowercase slug suitable for a tag or slug.
func slugifyTag(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' || r == '-' {
			result.WriteByte('-')
		}
	}
	out := result.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}
