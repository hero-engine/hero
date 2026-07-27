package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/async"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/version"
	"github.com/hero-engine/hero/internal/vocabulary"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	statusAll     bool
	statusHorizon string
	statusJSON    bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current state of the hero workspace",
	Long: `Displays specs in planning as a compact operational briefing: work
counts, everything in progress, up to ten priority-ranked upcoming and waiting
items, and the five most recent timestamped completions.

By default shows only horizon ∈ {now, next} — the actionable set.
Use --all to include someday/parking, or --horizon someday/parking
to view those specifically. Use hero list for complete filtered views,
including completed history, intake, and knowledge entries.`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "include someday/parking specs in the listing")
	statusCmd.Flags().StringVar(&statusHorizon, "horizon", "", "show only specs at this horizon (now/next/someday/parking)")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "emit typed JSON")

	RegisterSmoke(statusCmd, func(cmd *cobra.Command) error {
		return runStatus(cmd, nil)
	})
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	if statusJSON {
		return renderStatusJSON(projectRoot, heroDir)
	}

	// Surface workspace location and active scope when running from a
	// satellite or subfolder. Quiet when at root with no scope.
	printWorkspaceContext(projectRoot, heroDir)

	// Surface active vocabulary / methodology when either is configured
	// (per Risk mitigation in pm-foundation-delivery spec). Quiet for
	// engineering / legacy workspaces.
	if line := dialectLine(&cfg); line != "" {
		fmt.Println(line)
		fmt.Println()
	}
	printProjectMailSummary()

	// Auto-fire peer-side completion: any awaiting_peer spec whose
	// peer counterpart has reached completed is flipped to handed_back
	// before we render. Best-effort — silent on per-spec errors.
	if transitioned, err := peering.ReconcileAwaitingPeer(projectRoot, nil); err == nil && len(transitioned) > 0 {
		fmt.Printf("Peer-side completion detected for %d spec(s): %v — moved to handed_back.\n\n", len(transitioned), transitioned)
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	view := collectStatusView(specs, heroDir, statusAll, statusHorizon)
	vocab := activeVocab(&cfg)
	renderStatusView(view, vocab, statusAll, statusHorizon)

	// Smoke failures — surface prominently so regressions aren't missed.
	// Best-effort: missing last-run.json is treated as no failures.
	if smokeRecords, serr := loadSmokeResults(heroDir); serr == nil {
		var failed []SmokeRunRecord
		for _, r := range smokeRecords {
			if r.Status == "fail" {
				failed = append(failed, r)
			}
		}
		if len(failed) > 0 {
			fmt.Printf("\nSmoke failures (%d) — run `hero smoke status` for details:\n", len(failed))
			for _, r := range failed {
				fmt.Printf("  🔴 %s\n", r.Slug)
			}
		}
	}

	// Async delivery jobs
	printAsyncJobs()

	// Connection health
	fmt.Println()
	printConnectionHealth(cfg)

	// Version info
	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}
	wsVersion := version.WorkspaceVersion(heroDir)

	versionStr := fmt.Sprintf("Hero %s", binaryVersion)
	if wsVersion != "unknown" && wsVersion != binaryVersion {
		cmp := version.CompareVersions(binaryVersion, wsVersion)
		if cmp > 0 {
			versionStr += fmt.Sprintf(" (workspace %s — upgrade available)", wsVersion)
		}
	}

	totalInFlight := len(view.inProgress) + len(view.upcomingReady) +
		len(view.upcomingBlocked) + len(view.waiting)
	fmt.Printf("\n%s — %d in-flight, %d completed, %d knowledge entries\n",
		versionStr, totalInFlight, view.completedCount, view.knowledgeCount)

	return nil
}

const (
	statusUpcomingLimit  = 10
	statusWaitingLimit   = 10
	statusCompletedLimit = 5
)

type statusView struct {
	inProgress           []*spec.Spec
	upcomingReady        []*spec.Spec
	upcomingBlocked      []*spec.Spec
	waiting              []*spec.Spec
	timestampedCompleted []*spec.Spec
	completedCount       int
	intakeCount          int
	knowledgeCount       int
	autoContextCount     int
	hiddenByHorizon      int
	archiveInconsistency int
}

func collectStatusView(all []*spec.Spec, heroDir string, includeAll bool, horizon string) statusView {
	var view statusView
	bySlug := make(map[string]*spec.Spec, len(all))
	for _, item := range all {
		bySlug[item.Slug] = item
	}

	var handedBack, delivering, inReview []*spec.Spec
	for _, item := range all {
		switch {
		case item.IsPreCommitment():
			view.intakeCount++
			continue
		case item.IsKnowledge():
			view.knowledgeCount++
			if isAutoGenerated(item) {
				view.autoContextCount++
			}
			continue
		case !item.IsWorkSpec() && item.Type != spec.TypeInitiative:
			continue
		}

		inPlanning := pathWithin(filepath.Join(heroDir, "planning"), item.Path)
		inArchive := pathWithin(filepath.Join(heroDir, "specs"), item.Path)
		if item.Status == spec.StatusCompleted {
			view.completedCount++
			if inPlanning {
				view.archiveInconsistency++
			}
			if !item.CompletedAt.IsZero() {
				view.timestampedCompleted = append(view.timestampedCompleted, item)
			}
			continue
		}
		if inArchive {
			view.archiveInconsistency++
			continue
		}
		if !inPlanning {
			continue
		}
		if !statusHorizonMatches(item, includeAll, horizon) {
			view.hiddenByHorizon++
			continue
		}

		switch item.Status {
		case spec.StatusHandedBack:
			handedBack = append(handedBack, item)
		case spec.StatusDelivering:
			delivering = append(delivering, item)
		case spec.StatusInReview:
			inReview = append(inReview, item)
		case spec.StatusPlanning:
			if spec.IsBlocked(item, bySlug) {
				view.upcomingBlocked = append(view.upcomingBlocked, item)
			} else {
				view.upcomingReady = append(view.upcomingReady, item)
			}
		case spec.StatusHandedOff, spec.StatusAwaitingPeer:
			view.waiting = append(view.waiting, item)
		}
	}

	spec.SortByPriority(handedBack)
	spec.SortByPriority(delivering)
	spec.SortByPriority(inReview)
	spec.SortByPriority(view.upcomingReady)
	spec.SortByPriority(view.upcomingBlocked)
	spec.SortByPriority(view.waiting)
	view.inProgress = append(view.inProgress, handedBack...)
	view.inProgress = append(view.inProgress, delivering...)
	view.inProgress = append(view.inProgress, inReview...)
	sort.SliceStable(view.timestampedCompleted, func(i, j int) bool {
		left, right := view.timestampedCompleted[i], view.timestampedCompleted[j]
		if !left.CompletedAt.Equal(right.CompletedAt) {
			return left.CompletedAt.After(right.CompletedAt)
		}
		return left.Slug < right.Slug
	})
	return view
}

func statusHorizonMatches(item *spec.Spec, includeAll bool, horizon string) bool {
	if horizon != "" {
		return item.EffectiveHorizon() == spec.Horizon(horizon)
	}
	return includeAll || item.IsActiveHorizon()
}

func pathWithin(root, path string) bool {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) &&
		(len(rel) < 3 || rel[:3] != ".."+string(filepath.Separator))
}

func renderStatusView(view statusView, vocab *vocabulary.Vocabulary, includeAll bool, horizon string) {
	upcomingCount := len(view.upcomingReady) + len(view.upcomingBlocked)
	fmt.Printf("Work: %d in progress · %d upcoming (%d ready, %d blocked) · %d waiting · %d completed\n",
		len(view.inProgress), upcomingCount, len(view.upcomingReady), len(view.upcomingBlocked),
		len(view.waiting), view.completedCount)
	fmt.Printf("Other: %d intake · %d knowledge", view.intakeCount, view.knowledgeCount)
	if horizon == "" && !includeAll {
		fmt.Printf(" · %d hidden by horizon", view.hiddenByHorizon)
	}
	fmt.Println()
	if view.hiddenByHorizon > 0 && horizon == "" && !includeAll {
		fmt.Println("  Show hidden work with `hero status --all` or `hero status --horizon someday|parking`.")
	}
	if view.archiveInconsistency > 0 {
		fmt.Printf("Archive inconsistencies: %d — run `hero check` for details\n", view.archiveInconsistency)
	}

	if len(view.inProgress) == 0 && upcomingCount == 0 && len(view.waiting) == 0 && view.completedCount == 0 {
		fmt.Println("\nNo operational work. Specs: (none)")
	} else {
		printStatusGroup("In progress", view.inProgress, 0, "", vocab, nil)
		upcoming := append(append([]*spec.Spec{}, view.upcomingReady...), view.upcomingBlocked...)
		blocked := make(map[string]bool, len(view.upcomingBlocked))
		for _, item := range view.upcomingBlocked {
			blocked[item.Slug] = true
		}
		printStatusGroup("Upcoming", upcoming, statusUpcomingLimit,
			"hero list --status planning --sort priority", vocab, blocked)
		printStatusGroup("Waiting", view.waiting, statusWaitingLimit,
			"hero list --status handed_off,awaiting_peer --sort priority", vocab, nil)
		printRecentlyCompleted(view, vocab)
	}

	if view.intakeCount > 0 || view.knowledgeCount > 0 {
		fmt.Println("\nBrowse full corpus:")
		if view.intakeCount > 0 {
			fmt.Printf("  Intake — pre-commitment: %d total — `hero list --type intake`\n", view.intakeCount)
		}
		if view.knowledgeCount > 0 {
			fmt.Println("  Knowledge: `hero list --type convention,decision,rule,external,context,note`")
		}
		if view.autoContextCount > 0 {
			fmt.Printf("  %d knowledge entries are auto-generated code-scan stubs.\n", view.autoContextCount)
		}
	}
}

func printStatusGroup(label string, items []*spec.Spec, limit int, hint string, vocab *vocabulary.Vocabulary, blocked map[string]bool) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("\n%s (%d):\n", label, len(items))
	shown := len(items)
	if limit > 0 && shown > limit {
		shown = limit
	}
	for _, item := range items[:shown] {
		blockedLabel := ""
		if blocked[item.Slug] {
			blockedLabel = "  [blocked]"
		}
		claim := ""
		if item.ClaimedBy != "" {
			claim = fmt.Sprintf("  [%s]", item.ClaimedBy)
		}
		fmt.Printf("  %-30s  %-10s  %-13s  %s%s%s\n",
			item.Slug, displayType(vocab, string(item.Type)), item.Status, item.Title, blockedLabel, claim)
	}
	if shown < len(items) {
		fmt.Printf("  … %d more — `%s`\n", len(items)-shown, hint)
	}
}

func printRecentlyCompleted(view statusView, vocab *vocabulary.Vocabulary) {
	if view.completedCount == 0 {
		return
	}
	fmt.Printf("\nRecently completed (%d):\n", view.completedCount)
	shown := len(view.timestampedCompleted)
	if shown > statusCompletedLimit {
		shown = statusCompletedLimit
	}
	for _, item := range view.timestampedCompleted[:shown] {
		age := time.Since(item.CompletedAt).Truncate(time.Hour)
		fmt.Printf("  %-30s  %-10s  %-12s  %s\n",
			item.Slug, displayType(vocab, string(item.Type)), formatAge(age), item.Title)
	}
	if shown < view.completedCount {
		fmt.Printf("  … %d more — `hero list --status completed --sort recency`\n", view.completedCount-shown)
	}
}

type statusJSONSpec struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Horizon string `json:"horizon"`
}

func renderStatusJSON(projectRoot, heroDir string) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	items := make([]statusJSONSpec, 0, len(specs))
	for _, item := range specs {
		if item.IsWorkSpec() {
			if statusHorizon != "" && item.EffectiveHorizon() != spec.Horizon(statusHorizon) {
				continue
			}
			if statusHorizon == "" && !statusAll && !item.IsActiveHorizon() {
				continue
			}
		}
		items = append(items, statusJSONSpec{Slug: item.Slug, Title: item.Title, Type: string(item.Type), Status: string(item.Status), Horizon: string(item.EffectiveHorizon())})
	}
	payload := map[string]any{
		"workspace": projectRoot,
		"hero_dir":  heroDir,
		"specs":     items,
	}
	if summary, summaryErr := projectMailSummary(); summaryErr == nil {
		payload["mail"] = summary
	}
	return json.NewEncoder(os.Stdout).Encode(payload)
}

func formatAge(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	days := hours / 24
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// printConnectionHealth shows tracker and wiki connection status.
func printConnectionHealth(cfg config.Config) {
	fmt.Println("Connections:")

	// Tracker
	if cfg.Tracker != nil && cfg.Tracker.Type != "" && cfg.Tracker.Type != "none" {
		token, err := cfg.Tracker.ResolveToken()
		if err != nil || token == "" {
			fmt.Printf("  tracker    %-10s  %-30s  no token (run 'hero connect %s')\n",
				cfg.Tracker.Type, cfg.Tracker.Project, cfg.Tracker.Type)
		} else {
			fmt.Printf("  tracker    %-10s  %-30s  token set\n",
				cfg.Tracker.Type, cfg.Tracker.Project)
		}
	} else {
		fmt.Println("  tracker    (none)  — run 'hero connect github|jira|linear' to set up")
	}

	// Confluence
	if cfg.Confluence != nil && cfg.Confluence.BaseURL != "" {
		token, err := cfg.Confluence.ResolveToken()
		if err != nil || token == "" {
			fmt.Printf("  confluence %-10s  %-30s  no token (run 'hero connect confluence')\n",
				"wiki", cfg.Confluence.SpaceKey)
		} else {
			fmt.Printf("  confluence %-10s  %-30s  token set\n",
				"wiki", cfg.Confluence.SpaceKey)
		}
	}
}

// printAsyncJobs shows active and recent async delivery jobs.
func printAsyncJobs() {
	store := async.DefaultStore()
	jobs, err := store.Load()
	if err != nil || len(jobs) == 0 {
		return
	}

	var active, recent []async.Job
	for _, j := range jobs {
		switch j.Status {
		case async.StatusPending, async.StatusRunning:
			active = append(active, j)
		case async.StatusCompleted, async.StatusFailed:
			if time.Since(j.CompletedAt) < 24*time.Hour {
				recent = append(recent, j)
			}
		}
	}

	if len(active) == 0 && len(recent) == 0 {
		return
	}

	fmt.Println("\nAsync Delivery:")
	for _, j := range active {
		elapsed := time.Since(j.StartedAt).Truncate(time.Second)
		fmt.Printf("  %-30s  %-10s  %s  branch:%s\n", j.Slug, string(j.Status), elapsed, j.Branch)
	}
	for _, j := range recent {
		status := string(j.Status)
		extra := ""
		if j.Status == async.StatusFailed && j.Error != "" {
			extra = fmt.Sprintf("  error:%s", j.Error)
		}
		if j.PRURL != "" {
			extra = fmt.Sprintf("  PR:%s", j.PRURL)
		}
		fmt.Printf("  %-30s  %-10s  %s%s\n", j.Slug, status, formatAge(time.Since(j.CompletedAt)), extra)
	}
}

// isAutoGenerated reports whether a spec was scaffolded by hero scan
// rather than authored by a human. Used to filter the per-package
// "Package: X" stubs out of the headline status listing.
func isAutoGenerated(s *spec.Spec) bool {
	for _, t := range s.Tags {
		if t == "auto-generated" || t == "code-scan" {
			return true
		}
	}
	return false
}

// printWorkspaceContext shows where the hero workspace is rooted and
// what scope is active for the current cwd. It is silent when cwd ==
// root and no subproject scope applies — keeps the default `hero
// status` output unchanged for the common case.
func printWorkspaceContext(projectRoot, heroDir string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	ws, err := workspace.Locate(cwd)
	if err != nil {
		return
	}
	subs, _ := install.LoadSubprojects(heroDir)
	var declared []string
	if subs != nil {
		declared = subs.DeclaredPaths()
	}
	scope := ws.Scope(declared)

	if ws.CWD == ws.Root && scope == workspace.RootScope {
		return // at root, no scope, no need to print anything
	}

	if ws.IsSatellite {
		fmt.Printf("Workspace: %s (satellite at %s)\n", ws.Root, ws.SatellitePath)
	} else {
		fmt.Printf("Workspace: %s (cwd: %s)\n", ws.Root, ws.CWD)
	}
	if scope != workspace.RootScope {
		fmt.Printf("Scope:     %s\n", scope)
	}
	fmt.Println()
}
