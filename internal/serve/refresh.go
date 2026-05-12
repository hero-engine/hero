package serve

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
)

// ImportRefresher handles periodic import refresh for a project.
type ImportRefresher struct {
	projectRoot string
	heroDir     string
	slug        string
	bus         *EventBus
	interval    time.Duration
	cancel      context.CancelFunc
}

// StartImportRefresher starts a background goroutine that periodically runs
// the import refresh cycle for a project. Returns nil if auto_refresh is disabled.
func StartImportRefresher(projectRoot, heroDir, slug string, bus *EventBus) *ImportRefresher {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil
	}

	if cfg.Import == nil || !cfg.Import.AutoRefresh {
		return nil
	}

	interval := 30 * time.Minute
	if cfg.Import.RefreshInterval != "" {
		if d, err := time.ParseDuration(cfg.Import.RefreshInterval); err == nil && d > 0 {
			interval = d
		}
	}

	// Don't allow intervals shorter than 5 minutes
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &ImportRefresher{
		projectRoot: projectRoot,
		heroDir:     heroDir,
		slug:        slug,
		bus:         bus,
		interval:    interval,
		cancel:      cancel,
	}

	go r.run(ctx)
	return r
}

// Stop cancels the refresh loop.
func (r *ImportRefresher) Stop() {
	if r != nil && r.cancel != nil {
		r.cancel()
	}
}

func (r *ImportRefresher) run(ctx context.Context) {
	fmt.Fprintf(os.Stderr, "hero serve: auto-refresh enabled for %s (interval: %s)\n", r.slug, r.interval)

	// Run once immediately on startup
	r.refresh()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh()
		}
	}
}

func (r *ImportRefresher) refresh() {
	cfg, err := config.Load(r.projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: refresh config error for %s: %v\n", r.slug, err)
		return
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "none" || cfg.Tracker.Type == "" {
		return
	}

	t, err := tracker.New(cfg.Tracker)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: refresh tracker error for %s: %v\n", r.slug, err)
		return
	}

	behavior := &config.RefreshBehavior{}
	if cfg.Import != nil && cfg.Import.OnRefresh != nil {
		behavior = cfg.Import.OnRefresh
	}

	specs, err := spec.Discover(r.heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: refresh discover error for %s: %v\n", r.slug, err)
		return
	}

	var updated, resolved int
	for _, s := range specs {
		if s.TrackerID == "" || !s.IsWorkSpec() {
			continue
		}
		if s.Status == spec.StatusCompleted || s.Status == spec.StatusSuperseded {
			continue
		}

		issue, err := t.GetIssue(s.TrackerID)
		if err != nil {
			continue
		}

		mappedStatus := mapTrackerStatusForRefresh(issue.Status, t.Name())

		// Check for resolved
		if mappedStatus == string(spec.StatusCompleted) || mappedStatus == string(spec.StatusSuperseded) {
			if behavior.ResolvedAction() == "mark" || behavior.ResolvedAction() == "archive" {
				updateSpecFrontmatterField(s.Path, "status", string(spec.StatusCompleted))
				resolved++
			}
			continue
		}

		// Check for reassignment
		if behavior.ShouldMarkReassigned() && issue.Assignee != "" && s.ClaimedBy != "" {
			if !strings.EqualFold(issue.Assignee, s.ClaimedBy) {
				updateSpecFrontmatterField(s.Path, "claimed_by", issue.Assignee)
				addTagToSpec(s.Path, "reassigned")
				updated++
				continue
			}
		}

		// Sync status
		if behavior.ShouldUpdateStatus() && mappedStatus != "" && spec.Status(mappedStatus) != s.Status {
			updateSpecFrontmatterField(s.Path, "status", mappedStatus)
			updated++
		}
	}

	if updated > 0 || resolved > 0 {
		fmt.Fprintf(os.Stderr, "hero serve: auto-refresh for %s — %d updated, %d resolved\n",
			r.slug, updated, resolved)

		r.bus.Publish(Event{
			Type:    EventIndexRebuilt,
			Project: r.slug,
			Message: fmt.Sprintf("auto-refresh: %d updated, %d resolved", updated, resolved),
		})
	}
}

// mapTrackerStatusForRefresh maps tracker status to spec status (same logic as CLI pull).
func mapTrackerStatusForRefresh(trackerStatus, trackerType string) string {
	normalized := strings.ToLower(strings.TrimSpace(trackerStatus))

	switch normalized {
	case "open", "to do", "todo", "backlog", "new":
		return string(spec.StatusPlanning)
	case "in progress", "in_progress", "started", "doing":
		return string(spec.StatusDelivering)
	case "in review", "in_review", "review":
		return string(spec.StatusInReview)
	case "closed", "done", "resolved", "completed", "complete":
		return string(spec.StatusCompleted)
	}

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

// updateSpecFrontmatterField is a lightweight wrapper for use in the server package.
func updateSpecFrontmatterField(path, key, value string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	updated := spec.SetFrontmatterField(content, key, value)
	_ = os.WriteFile(path, []byte(updated), 0o644)
}

// addTagToSpec adds a tag to a spec's frontmatter.
func addTagToSpec(path, tag string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "tags:") {
			if strings.Contains(trimmed, tag) {
				return // already has tag
			}
			if idx := strings.LastIndex(lines[i], "]"); idx >= 0 {
				lines[i] = lines[i][:idx] + ", " + tag + "]"
			}
			_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
			return
		}
	}
}
