package tracker

import (
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
)

// PullResult describes the outcome of a scan-time tracker pull.
//
// When Skipped is true, Reason carries a one-line explanation suitable
// for printing alongside the scan output.
type PullResult struct {
	Skipped bool
	Reason  string
	Issues  int
	Persons int
	Edges   int
}

// defaultPullLimit caps how many issues a scan-time tracker pull will
// fetch in one go. Conservative — scan is best-effort enrichment, not
// the bulk-import path; use `hero sync import` for that.
const defaultPullLimit = 100

// PullAndWriteGraph fetches open issues from the configured tracker
// and upserts Issue (+ Person) nodes into store. Best-effort: returns
// (result with Skipped=true, nil) when no tracker is configured, the
// token isn't available, or the tracker call fails — never errors out
// of the scan.
//
// trackerKnowledgeDir is the path to .hero/knowledge/tracker/ for the
// Jira field cache (same argument NewWithJiraConfig takes).
func PullAndWriteGraph(cfg *config.TrackerConfig, jiraCfg *config.JiraConfig, trackerKnowledgeDir, repoKey string, store *graph.Store) (*PullResult, error) {
	if cfg == nil || cfg.Type == "" || cfg.Type == "none" {
		return &PullResult{Skipped: true, Reason: "tracker not configured"}, nil
	}
	if _, err := cfg.ResolveToken(); err != nil {
		return &PullResult{Skipped: true, Reason: err.Error()}, nil
	}

	t, err := NewWithJiraConfig(cfg, jiraCfg, trackerKnowledgeDir)
	if err != nil {
		return &PullResult{Skipped: true, Reason: err.Error()}, nil
	}

	issues, err := t.ListIssues("", defaultPullLimit)
	if err != nil {
		return &PullResult{Skipped: true, Reason: err.Error()}, nil
	}
	if len(issues) == 0 {
		return &PullResult{}, nil
	}

	summary, err := WriteIssuesGraph(issues, repoKey, store)
	if err != nil {
		return nil, err
	}
	return &PullResult{
		Issues:  summary.Issues,
		Persons: summary.Persons,
		Edges:   summary.Edges,
	}, nil
}
