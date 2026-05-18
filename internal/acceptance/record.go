// Package acceptance ingests run-result payloads into the unified
// knowledge graph, flipping Criterion status bitemporally and wiring
// satisfied_by / breaks edges to the Commit nodes that earned them.
//
// This is Phase 2 of acceptance-criteria-graph. Phase 1 (the parser
// and Criterion-node ingest path) lives in internal/spec/.
package acceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

// RunResult is one entry in the run-result JSON payload. The schema
// is intentionally tiny — `ac` and `status` are the contract; the
// rest is best-effort metadata that improves traceability.
//
//	[
//	  {"ac": "master-ingest-restore:AC-2", "status": "pass",
//	   "ts": "2026-04-28T22:30:00Z", "sha": "0dce2d1"},
//	  {"ac": "master-ingest-restore:AC-3", "status": "fail",
//	   "ts": "2026-04-28T22:30:00Z", "sha": "0dce2d1"}
//	]
type RunResult struct {
	AC        string `json:"ac"`             // <spec-slug>:AC-N
	Status    string `json:"status"`         // pass | fail | skip
	Timestamp string `json:"ts,omitempty"`   // RFC3339; defaults to now
	SHA       string `json:"sha,omitempty"`  // git commit; produces edge
	RunID     string `json:"run_id,omitempty"`
}

// RecordSummary reports counts written by Record.
type RecordSummary struct {
	Criteria      int // status flips applied (no-op flips not counted)
	NoOps         int // results where status was already correct
	Unknown       int // results referencing an AC key that doesn't exist
	SatisfiedBy   int // Criterion → Commit edges added
	Breaks        int // Commit → Criterion edges added (regressions)
}

// Record walks results, flips Criterion status (bitemporal), and
// emits satisfied_by / breaks edges keyed off SHA when present.
//
// Status mapping:
//   - "pass" → "passing" (or "regressed-passing" if recovering from a fail)
//   - "fail" → "failing" if previously proposed/passing
//             "regressed" if it was previously passing
//   - "skip" → no-op, recorded as NoOps
//
// Unknown ACs (the key doesn't match any Criterion in the graph) are
// counted in summary.Unknown rather than erroring — so a stale run
// payload doesn't fail an entire batch.
func Record(results []RunResult, repoKey string, store *graph.Store) (*RecordSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("acceptance: Record requires non-nil Store")
	}
	summary := &RecordSummary{}
	source := map[string]any{"kind": "run-result"}

	for _, r := range results {
		key := strings.TrimSpace(r.AC)
		if key == "" {
			continue
		}
		newStatus, ok := mapRunStatus(r.Status)
		if !ok {
			// Unknown status string — skip silently.
			summary.NoOps++
			continue
		}

		existing, err := store.GetNode("Criterion", key)
		if err != nil || existing == nil {
			summary.Unknown++
			continue
		}
		prevStatus, _ := existing.Props["status"].(string)

		// Promote pass-after-fail to "passing" (the eventually-clean
		// state). Promote fail-after-pass to "regressed".
		switch {
		case newStatus == "passing" && prevStatus == "failing":
			newStatus = "passing"
		case newStatus == "failing" && prevStatus == "passing":
			newStatus = "regressed"
		}

		if prevStatus == newStatus {
			summary.NoOps++
			continue
		}

		newProps := graph.CloneProps(existing.Props)
		newProps["status"] = newStatus
		newProps["last_run_at"] = orNow(r.Timestamp)
		if newStatus == "passing" {
			newProps["last_pass_at"] = orNow(r.Timestamp)
		}
		if r.RunID != "" {
			newProps["last_run_id"] = r.RunID
		}

		// Re-upsert with a hash that includes the new status. Because
		// scan-path hashCriterion folds status in, the bitemporal
		// model invalidates the prior row and inserts a new current
		// one — exactly the contract per the spec.
		critID, err := store.UpsertNode(&graph.Node{
			Type:        "Criterion",
			Key:         key,
			Props:       newProps,
			Repo:        existing.Repo,
			ContentHash: hashCriterionStatus(key, statementOf(newProps), newStatus),
			Source:      source,
		})
		if err != nil {
			return summary, fmt.Errorf("upsert Criterion %s: %w", key, err)
		}
		summary.Criteria++

		if r.SHA == "" {
			continue
		}
		// Resolve the Commit node by SHA. Commit nodes are keyed by
		// full SHA in gitutil.WriteGitLogGraph; tolerate short SHAs by
		// upserting a stub if not found, so the edge always lands.
		commitID, err := resolveOrStubCommit(store, r.SHA)
		if err != nil {
			return summary, fmt.Errorf("resolve commit %s: %w", r.SHA, err)
		}
		switch newStatus {
		case "passing":
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: critID, ToID: commitID, Type: "satisfied_by",
				Props:  map[string]any{"recorded_at": orNow(r.Timestamp)},
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return summary, fmt.Errorf("upsert satisfied_by: %w", err)
			}
			summary.SatisfiedBy++
		case "regressed":
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: commitID, ToID: critID, Type: "breaks",
				Props:  map[string]any{"recorded_at": orNow(r.Timestamp)},
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return summary, fmt.Errorf("upsert breaks: %w", err)
			}
			summary.Breaks++
		}
	}
	return summary, nil
}

// LoadRunResults reads and decodes a run-result JSON file. Returns an
// empty slice (not an error) for an empty / missing file so callers
// can drive batch ingest from a glob.
func LoadRunResults(path string) ([]RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var out []RunResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

// mapRunStatus converts the run-result `status` string to the
// canonical Criterion status. Returns ok=false for unknown values.
func mapRunStatus(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pass", "passed", "passing", "ok", "green":
		return "passing", true
	case "fail", "failed", "failing", "red":
		return "failing", true
	case "skip", "skipped":
		return "", false
	default:
		return "", false
	}
}

// resolveOrStubCommit returns the row id of the Commit node keyed by
// sha. Tries three lookups before stubbing:
//
//  1. Exact key match (covers full-SHA inputs).
//  2. Prefix match against existing Commit keys (covers short-SHA
//     inputs landing on an already-ingested full-SHA node).
//
// Falls back to creating a stub when nothing matches. This used to
// stub eagerly on any non-exact match, producing parallel "abc1234"
// and "abc1234...full" nodes — which broke the AC participation join
// because the stub had no touches edges.
func resolveOrStubCommit(store *graph.Store, sha string) (int64, error) {
	if id, err := store.GetNodeID("Commit", sha); err == nil {
		return id, nil
	}
	// Prefix match: a short SHA in the run-result file lands on the
	// existing full-SHA Commit ingested by git scan.
	if len(sha) >= 4 && len(sha) < 40 {
		row := store.DB().QueryRow(
			`SELECT id FROM nodes
			  WHERE type = 'Commit' AND valid_to IS NULL AND key LIKE ?
			  ORDER BY length(key) DESC
			  LIMIT 1`,
			sha+"%",
		)
		var id int64
		if err := row.Scan(&id); err == nil {
			return id, nil
		}
	}
	return store.UpsertNode(&graph.Node{
		Type: "Commit", Key: sha,
		Props:       map[string]any{"sha": sha, "stub": true},
		ContentHash: "stub-" + sha,
		Source:      map[string]any{"kind": "run-result-stub"},
	})
}

// hashCriterionStatus produces the same hash spec.WriteGraph uses for
// Criterion nodes (key|statement|status), so a recorded status is
// bit-for-bit comparable to a scan-time write. Duplicated here to
// avoid importing internal/spec from internal/acceptance.
//
// IMPORTANT: keep this in sync with internal/spec.hashCriterion. If
// one changes, the other must change identically — otherwise scan
// will invalidate every recorded row on its next pass.
func hashCriterionStatus(key, statement, status string) string {
	sum := sha256.Sum256([]byte(key + "|" + statement + "|" + status))
	return hex.EncodeToString(sum[:])
}

func orNow(ts string) string {
	if ts != "" {
		return ts
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func statementOf(props map[string]any) string {
	if s, ok := props["statement"].(string); ok {
		return s
	}
	return ""
}
