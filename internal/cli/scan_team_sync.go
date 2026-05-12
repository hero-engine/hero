package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
)

// teamSyncResult mirrors the other scan-time best-effort results: a
// `Skipped` boolean + `Reason` for the skip path, counters for the
// happy path. Mostly cosmetic — runOpportunisticTeamSync is the only
// caller and just prints.
type teamSyncResult struct {
	Skipped       bool
	Reason        string
	Pushed        int
	NodesPulled   int
	EdgesPulled   int
	EdgesDeferred int
}

// pullStaleAfter is how long we wait between opportunistic pulls. The
// scan path is best-effort enrichment, not the canonical sync verb —
// users still run `hero sync graph pull` for an immediate pull. Five
// minutes matches the rate-limit captured in master-ingest-restore.
const pullStaleAfter = 5 * time.Minute

// runOpportunisticTeamSync is the scan-time wrapper around
// `hero sync graph push|pull`. It pushes pending non-local nodes
// always (cheap), and pulls deltas only when the last-pull cursor is
// stale or absent. Skips silently when the user isn't logged in or
// no team server is configured.
//
// Best-effort: any network/auth error is collapsed into Skipped+Reason
// rather than bubbling up — never breaks the scan.
func runOpportunisticTeamSync(cfg config.Config, repoKey string, store *graph.Store) (*teamSyncResult, error) {
	creds, err := loadCredentials()
	if err != nil || creds == nil || creds.AccessToken == "" {
		return &teamSyncResult{Skipped: true, Reason: "not logged in to Hero Cloud"}, nil
	}
	if cfg.Cloud == nil || cfg.Cloud.OrgID == "" {
		return &teamSyncResult{Skipped: true, Reason: "no cloud.org_id in hero.json"}, nil
	}

	cloudURL := creds.CloudURL
	if cloudURL == "" {
		cloudURL = cloudBaseURL()
	}
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			token: creds.AccessToken,
			base:  http.DefaultTransport,
		},
	}
	c := graph.NewSyncClient(
		fmt.Sprintf("%s/api/v1/orgs/%s", cloudURL, cfg.Cloud.OrgID),
		repoKey,
		cfg.Cloud.OrgID,
	)
	c.HTTP = httpClient

	result := &teamSyncResult{}

	// Push always — only sends rows newer than last successful push so
	// the cost is proportional to the local delta.
	pushResp, err := store.Push(c)
	if err != nil {
		return &teamSyncResult{Skipped: true, Reason: fmt.Sprintf("push failed: %v", err)}, nil
	}
	if pushResp != nil {
		result.Pushed = pushResp.Accepted
	}

	// Pull only when the last-pull timestamp is stale. Avoids
	// hammering the server when the user runs `hero scan` repeatedly
	// during a session.
	lastAt, _ := store.LastPullAt(c.ServerURL)
	if shouldPull(lastAt) {
		_, nodesApplied, edgesApplied, edgesDeferred, err := store.Pull(c)
		if err != nil {
			// Push already succeeded — surface a partial result rather
			// than swallowing the push counters.
			result.Skipped = true
			result.Reason = fmt.Sprintf("pull failed: %v", err)
			return result, nil
		}
		result.NodesPulled = nodesApplied
		result.EdgesPulled = edgesApplied
		result.EdgesDeferred = edgesDeferred
	}
	return result, nil
}

// shouldPull returns true when last-pull-at is unset or older than
// pullStaleAfter. Unparseable values fall through as "pull anyway"
// since the alternative is wedging on a corrupted local cursor.
func shouldPull(lastAt string) bool {
	if lastAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, lastAt)
	if err != nil {
		return true
	}
	return time.Since(t) > pullStaleAfter
}
