package acceptance

import (
	"database/sql"
	"fmt"

	"github.com/hero-engine/hero/internal/graph"
)

// ParticipationSummary reports what ComputeParticipation did.
type ParticipationSummary struct {
	Edges   int // participates_in edges upserted (current run)
	Touched int // distinct File nodes that gained an edge
	Skipped int // (file, criterion) pairs where the edge already existed
}

// ComputeParticipation derives File --participates_in--> Criterion
// edges from the join Criterion --satisfied_by--> Commit
// --touches--> File. Phase 4 of acceptance-criteria-graph.
//
// "If a file was touched by a commit that satisfied an AC, then
// changing that file likely affects that AC." This is the structural
// signal that lets `hero relevant <file>` surface AC-awareness:
// editors get told which acceptance criteria they're potentially
// breaking before the run actually fails.
//
// Idempotent — UpsertEdge is content-addressed by (from_id, type,
// to_id), so re-running on an unchanged graph produces no new rows.
// The Skipped counter reflects pairs we'd have written but that
// already exist as current edges.
//
// repoKey scopes which Commit nodes count, so cross-repo joins don't
// pollute another repo's File→Criterion mapping.
func ComputeParticipation(store *graph.Store, repoKey string) (ParticipationSummary, error) {
	var summary ParticipationSummary
	if store == nil {
		return summary, fmt.Errorf("acceptance: nil store")
	}

	rows, err := store.DB().Query(
		`SELECT f.id        AS file_id,
		        c.id        AS crit_id,
		        commit_n.key AS commit_sha
		   FROM nodes c
		   JOIN edges sb ON sb.from_id = c.id
		                AND sb.type = 'satisfied_by'
		                AND sb.valid_to IS NULL
		   JOIN nodes commit_n ON commit_n.id = sb.to_id
		                      AND commit_n.type = 'Commit'
		                      AND commit_n.valid_to IS NULL
		                      AND (commit_n.repo = ? OR COALESCE(commit_n.repo,'') = '')
		   JOIN edges tch ON tch.from_id = commit_n.id
		                AND tch.type = 'touches'
		                AND tch.valid_to IS NULL
		   JOIN nodes f ON f.id = tch.to_id
		               AND f.type = 'File'
		               AND f.valid_to IS NULL
		  WHERE c.type = 'Criterion' AND c.valid_to IS NULL`,
		repoKey,
	)
	if err != nil {
		return summary, fmt.Errorf("query participation: %w", err)
	}
	defer rows.Close()

	type pair struct{ fileID, critID int64 }
	seen := make(map[pair]struct{})
	files := make(map[int64]struct{})

	for rows.Next() {
		var fileID, critID int64
		var commitSHA sql.NullString
		if err := rows.Scan(&fileID, &critID, &commitSHA); err != nil {
			return summary, err
		}
		key := pair{fileID, critID}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		// Pre-check existence for accurate Skipped count. UpsertEdge
		// is idempotent, but it doesn't tell us whether the edge was
		// new or already there — so we ask the index directly.
		var existing int64
		row := store.DB().QueryRow(
			`SELECT id FROM edges
			  WHERE from_id = ? AND to_id = ? AND type = 'participates_in'
			    AND valid_to IS NULL
			  LIMIT 1`,
			fileID, critID,
		)
		if err := row.Scan(&existing); err == nil {
			summary.Skipped++
			continue
		}

		props := map[string]any{}
		if commitSHA.Valid && commitSHA.String != "" {
			props["via_commit"] = commitSHA.String
		}
		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: fileID,
			ToID:   critID,
			Type:   "participates_in",
			Props:  props,
			Repo:   repoKey,
			Source: map[string]any{"kind": "ac-participation-join"},
		}); err != nil {
			return summary, fmt.Errorf("upsert participates_in: %w", err)
		}
		summary.Edges++
		files[fileID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return summary, err
	}
	summary.Touched = len(files)
	return summary, nil
}
