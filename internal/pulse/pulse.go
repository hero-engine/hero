package pulse

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Period represents a time window for pulse analysis.
type Period struct {
	From time.Time
	To   time.Time
}

// SpecSummary holds pulse data about a single spec.
type SpecSummary struct {
	Slug       string
	Title      string
	Status     string
	ClaimedBy  string
	LastCommit time.Time
	DaysStale  int
	TrackerID  string
}

// KnowledgeUpdate represents a knowledge entry updated during the period.
type KnowledgeUpdate struct {
	Slug       string
	Title      string
	ModifiedAt time.Time
}

// DriftEntry represents a spec with drift warnings in the pulse report.
type DriftEntry struct {
	Slug         string
	Title        string
	Warnings     int
	HasViolation bool
}

// AmbientSizeDrift is the count+hint summary emitted by
// sizing.AmbientDrift, mirrored into the pulse response shape so MCP
// clients (and renderers) consume it without reaching into the sizing
// package. Nil → no drift to surface (the quiet state). See spec
// roadmap-review-ambient-surfacing.
type AmbientSizeDrift struct {
	Count int    `json:"count"`
	Hint  string `json:"hint"`
}

// PulseData is the complete pulse report for a period.
type PulseData struct {
	Period           Period
	Done             []SpecSummary
	InFlight         []SpecSummary
	AtRisk           []SpecSummary
	Drift            []DriftEntry
	KnowledgeUpdates []KnowledgeUpdate
	Blockers         []string
	// SizeDrift carries the workspace-wide ambient size-drift summary
	// (count + hint). Nil when the AmbientDrift helper reports Quiet
	// (no drift, or stop-nagging window active). Surfaces are
	// instructed to emit count+hint only — never per-spec rows.
	SizeDrift *AmbientSizeDrift
}

// CalcPeriod returns the period based on flags.
// If since is set, use since..now. If week is true, use 7 days ago..now.
// Otherwise use sprintDays (default 14) back from now.
func CalcPeriod(since string, week bool, sprintDays int) Period {
	now := time.Now()
	if since != "" {
		t, err := time.Parse("2006-01-02", since)
		if err == nil {
			return Period{From: t, To: now}
		}
		// Try RFC3339
		t, err = time.Parse(time.RFC3339, since)
		if err == nil {
			return Period{From: t, To: now}
		}
	}
	if week {
		return Period{From: now.AddDate(0, 0, -7), To: now}
	}
	if sprintDays <= 0 {
		sprintDays = 14
	}
	return Period{From: now.AddDate(0, 0, -sprintDays), To: now}
}

// BuildPulse assembles PulseData from the spec corpus and git log.
// heroDir is the .hero/ directory. period defines the time window.
func BuildPulse(heroDir string, period Period, staleDeliveringDays, stalePlanningDays int) (*PulseData, error) {
	if staleDeliveringDays <= 0 {
		staleDeliveringDays = 3
	}
	if stalePlanningDays <= 0 {
		stalePlanningDays = 7
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	// Find project root (parent of heroDir)
	projectRoot := filepath.Dir(heroDir)

	pulse := &PulseData{
		Period: period,
	}

	now := time.Now()

	for _, s := range specs {
		if !s.IsWorkSpec() {
			// Check for knowledge updates
			if s.IsKnowledge() && s.ModifiedAt.After(period.From) && s.ModifiedAt.Before(period.To.Add(24*time.Hour)) {
				pulse.KnowledgeUpdates = append(pulse.KnowledgeUpdates, KnowledgeUpdate{
					Slug:       s.Slug,
					Title:      s.Title,
					ModifiedAt: s.ModifiedAt,
				})
			}
			continue
		}

		summary := SpecSummary{
			Slug:      s.Slug,
			Title:     s.Title,
			Status:    string(s.Status),
			ClaimedBy: s.ClaimedBy,
			TrackerID: s.TrackerID,
		}

		switch s.Status {
		case spec.StatusCompleted:
			// Include in Done if modified within the period
			if s.ModifiedAt.After(period.From) && s.ModifiedAt.Before(period.To.Add(24*time.Hour)) {
				pulse.Done = append(pulse.Done, summary)
			}

		case spec.StatusDelivering:
			// Check staleness via git log
			lastCommit := gitLastCommit(projectRoot, s.Path)
			summary.LastCommit = lastCommit

			if !lastCommit.IsZero() {
				summary.DaysStale = int(now.Sub(lastCommit).Hours() / 24)
			} else {
				// No commit info — use file mod time as proxy
				summary.DaysStale = int(now.Sub(s.ModifiedAt).Hours() / 24)
			}

			if summary.DaysStale >= staleDeliveringDays {
				pulse.AtRisk = append(pulse.AtRisk, summary)
			} else {
				pulse.InFlight = append(pulse.InFlight, summary)
			}

		case spec.StatusPlanning, spec.StatusInReview:
			// Planning specs that are old go to AtRisk
			age := int(now.Sub(s.CreatedAt).Hours() / 24)
			if age >= stalePlanningDays {
				summary.DaysStale = age
				pulse.AtRisk = append(pulse.AtRisk, summary)
			} else {
				pulse.InFlight = append(pulse.InFlight, summary)
			}
		}
	}

	return pulse, nil
}

// PopulateDrift fills the Drift field on a PulseData by running drift detection
// on all delivering specs. Separated from BuildPulse to avoid a circular import
// — the caller (CLI/MCP layer) calls this after BuildPulse.
func PopulateDrift(p *PulseData, driftEntries []DriftEntry) {
	p.Drift = driftEntries
}

// gitLastCommit returns the time of the most recent git commit touching the given file path.
// Returns zero time if git is unavailable or no commits found.
func gitLastCommit(projectRoot, filePath string) time.Time {
	// Use --follow to handle renames; limit to 1 result
	cmd := exec.Command("git", "-C", projectRoot, "log", "-1", "--pretty=format:%ai", "--", filePath)
	cmd.Stderr = nil // suppress git errors
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return time.Time{}
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return time.Time{}
	}

	// git --pretty=format:%ai outputs "2006-01-02 15:04:05 -0700"
	t, err := time.Parse("2006-01-02 15:04:05 -0700", line)
	if err != nil {
		// Try alternate format
		t, err = time.Parse(time.RFC3339, line)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// FilterBySlug returns a PulseData containing only entries matching the given slug.
func FilterBySlug(p *PulseData, slug string) *PulseData {
	filtered := &PulseData{
		Period: p.Period,
	}

	for _, s := range p.Done {
		if s.Slug == slug {
			filtered.Done = append(filtered.Done, s)
		}
	}
	for _, s := range p.InFlight {
		if s.Slug == slug {
			filtered.InFlight = append(filtered.InFlight, s)
		}
	}
	for _, s := range p.AtRisk {
		if s.Slug == slug {
			filtered.AtRisk = append(filtered.AtRisk, s)
		}
	}
	for _, u := range p.KnowledgeUpdates {
		if u.Slug == slug {
			filtered.KnowledgeUpdates = append(filtered.KnowledgeUpdates, u)
		}
	}
	filtered.Blockers = p.Blockers

	return filtered
}


