// Package projection renders graph state into the user-visible
// markdown surfaces hero already exposes: NEXT.md, sprint views,
// future GitHub Pages output, etc.
//
// Projections are deterministic and fast (<100ms target). They run
// pure SQL queries against the graph and format the results as
// markdown — no LLM calls. The expensive work (entity extraction,
// pattern detection, narrative synthesis) happens in ingest; reads
// stay snappy.
//
// Projection is read-only. The graph is the source of truth; the
// rendered markdown is regenerated on demand. If a user hand-edits
// a projected file, the next render replaces it.
package projection

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/methodology"
	"github.com/hero-engine/hero/internal/vocabulary"
)

// NextMDOptions tunes the NEXT.md projection.
type NextMDOptions struct {
	RepoKey   string // partition filter; required
	Branch    string // current branch (frontmatter only)
	SessionID string // anchors "Tried and failed" to a session
	NextN     int    // open features to surface under "## Next" (default 1)
	// Vocab is the active vocabulary preset used to render type / kind
	// display names (e.g. "feature" → "Story" under agile-scrum). Nil
	// preserves the canonical literal — engineering / legacy workspaces
	// render identically to today.
	Vocab *vocabulary.Vocabulary
	// Methodology is the active methodology profile. Nil falls through
	// to the methodology-neutral phrasing used today.
	Methodology *methodology.Methodology

	// HeroDir is the absolute path to `.hero/`. Required to surface the
	// ambient `## Roadmap shape` section (the helper reads specs and
	// the session-record directory under HeroDir). Empty disables the
	// surfacing — the section is omitted entirely. Older callers
	// that haven't been updated stay quiet by default.
	HeroDir string
	// ProjectRoot is the absolute project root. Required alongside
	// HeroDir for the ambient surfacing (used for git-mtime lookups).
	// Empty disables the surfacing.
	ProjectRoot string
	// ActiveSpec is the slug the session is currently touching, when
	// known. Empty (the commit-time path) skips the active-spec rule
	// of the noise filter — rule 2 (recency) and rule 3 (high-impact)
	// still apply.
	ActiveSpec string
	// RoadmapRecencyDays overrides the default 7-day recency window
	// for the ambient noise filter. <=0 → use the helper default.
	RoadmapRecencyDays int
	// RoadmapStopNaggingHours overrides the default 24-hour
	// suppression window after a `/roadmap-review` session record.
	// <=0 → use the helper default.
	RoadmapStopNaggingHours int
}

// NextMD renders the contents of .hero/NEXT.md from the graph. The
// returned string starts with a frontmatter block and ends with a
// trailing newline.
//
// Projections are best-effort: if the graph is sparse (fresh repo),
// individual sections render as "Nothing yet." rather than failing.
func NextMD(store *graph.Store, opts NextMDOptions) (string, error) {
	if store == nil {
		return "", fmt.Errorf("projection: NextMD requires non-nil Store")
	}
	if opts.RepoKey == "" {
		return "", fmt.Errorf("projection: RepoKey is required")
	}
	if opts.NextN == 0 {
		opts.NextN = 1
	}

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "updated: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "repo: %s\n", opts.RepoKey)
	if opts.SessionID != "" {
		fmt.Fprintf(&b, "session: %s\n", opts.SessionID)
	}
	if opts.Branch != "" {
		fmt.Fprintf(&b, "branch: %s\n", opts.Branch)
	}
	b.WriteString("---\n\n")

	// Just finished — pointer to git log rather than a frozen copy.
	// Commit lists embedded in NEXT.md are always stale (pre-commit
	// can't see its own SHA) and duplicate what git log already provides.
	b.WriteString("## Just finished\n\n")
	b.WriteString("Run `git log --oneline -10` for recent commits.\n")
	b.WriteString("\n")

	// Next
	b.WriteString("## Next\n\n")
	openFeatures, err := openFeaturesByPriority(store, opts.RepoKey, opts.NextN)
	if err != nil {
		return "", fmt.Errorf("next: %w", err)
	}
	featurePlural := pluralizeWorkType(opts.Vocab, "feature")
	if len(openFeatures) == 0 {
		fmt.Fprintf(&b, "No open %s in this repo.\n", featurePlural)
	} else {
		for _, f := range openFeatures {
			fmt.Fprintf(&b, "- **%s** (`%s`", f.title, f.slug)
			if f.priority != "" {
				fmt.Fprintf(&b, ", %s", f.priority)
			}
			if f.status != "" {
				fmt.Fprintf(&b, ", %s", f.status)
			}
			b.WriteString(")\n")
		}
		fmt.Fprintf(&b, "\n→ `/deliver %s`\n", openFeatures[0].slug)
	}
	b.WriteString("\n")

	// NOTE: the ambient `## Roadmap shape` size-drift line was removed from the
	// NEXT.md projection — it embedded a corpus-derived count that is stale by
	// construction in the committed file (the pre-commit hook computes it
	// against the pre-commit index; a clean rebuild computes a different value),
	// which made the CI byte-exact drift gate structurally unwinnable. The
	// authoritative count still lives in `hero size --check`, `hero pulse`, and
	// the MCP surfaces. Spec: next-drift-gate-unwinnable.

	// Blocked on
	b.WriteString("## Blocked on\n\n")
	blockers, err := blockedFeatures(store, opts.RepoKey)
	if err != nil {
		return "", fmt.Errorf("blocked on: %w", err)
	}
	if len(blockers) == 0 {
		b.WriteString("Nothing.\n")
	} else {
		for _, blk := range blockers {
			fmt.Fprintf(&b, "- **%s** ← waiting on `%s` (%s)\n",
				blk.featureSlug, blk.blockerSlug, blk.blockerStatus)
		}
	}
	b.WriteString("\n")

	// Tried and failed
	b.WriteString("## Tried and failed\n\n")
	attempts, err := attemptsForSession(store, opts.SessionID)
	if err != nil {
		return "", fmt.Errorf("tried and failed: %w", err)
	}
	if len(attempts) == 0 {
		b.WriteString("Nothing this session.\n")
	} else {
		for _, a := range attempts {
			fmt.Fprintf(&b, "- %s\n", oneLine(a))
		}
	}
	b.WriteString("\n")

	// Context to carry forward
	b.WriteString("## Context to carry forward\n\n")
	carry, err := contextToCarry(store, opts.RepoKey)
	if err != nil {
		return "", fmt.Errorf("context: %w", err)
	}
	if len(carry) == 0 {
		b.WriteString("Nothing pinned.\n")
	} else {
		for _, c := range carry {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	b.WriteString("\n")

	return b.String(), nil
}

// --- queries ---------------------------------------------------------------

type featureRow struct {
	slug, title, status, priority string
}

// openFeaturesByPriority returns up to `limit` Feature nodes whose
// status is anything but completed/superseded, ranked by priority
// (P0 > P1 > P2 ...) then by recency.
func openFeaturesByPriority(store *graph.Store, repoKey string, limit int) ([]featureRow, error) {
	rows, err := store.DB().Query(
		`SELECT key,
		        json_extract(props, '$.title') AS title,
		        json_extract(props, '$.status') AS status,
		        json_extract(props, '$.priority') AS priority,
		        ingested_at
		   FROM nodes
		  WHERE type = 'Feature' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed', 'superseded')
		  ORDER BY COALESCE(json_extract(props, '$.priority'), 'P9'),
		           ingested_at DESC
		  LIMIT ?`,
		repoKey, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []featureRow
	for rows.Next() {
		var f featureRow
		var title, status, priority sql.NullString
		var ingested string
		if err := rows.Scan(&f.slug, &title, &status, &priority, &ingested); err != nil {
			return nil, err
		}
		f.title, f.status, f.priority = title.String, status.String, priority.String
		out = append(out, f)
	}
	return out, rows.Err()
}

type blockerRow struct {
	featureSlug, blockerSlug, blockerStatus string
}

// blockedFeatures finds Features that depend_on something which is
// not yet completed.
func blockedFeatures(store *graph.Store, repoKey string) ([]blockerRow, error) {
	rows, err := store.DB().Query(
		`SELECT f.key AS feature_slug,
		        b.key AS blocker_slug,
		        COALESCE(json_extract(b.props, '$.status'), '') AS blocker_status
		   FROM nodes f
		   JOIN edges e ON e.from_id = f.id AND e.type IN ('depends_on', 'blocks') AND e.valid_to IS NULL
		   JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
		  WHERE f.type = 'Feature' AND f.repo = ? AND f.valid_to IS NULL
		    AND COALESCE(json_extract(f.props, '$.status'), '') NOT IN ('completed', 'superseded')
		    AND COALESCE(json_extract(b.props, '$.status'), '') NOT IN ('completed', 'accepted')
		  ORDER BY f.key, b.key`,
		repoKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []blockerRow
	for rows.Next() {
		var b blockerRow
		if err := rows.Scan(&b.featureSlug, &b.blockerSlug, &b.blockerStatus); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// attemptsForSession returns Attempt body strings linked to the given
// session via attempted_in. If sessionID is empty, returns nothing.
func attemptsForSession(store *graph.Store, sessionID string) ([]string, error) {
	if sessionID == "" {
		return nil, nil
	}
	rows, err := store.DB().Query(
		`SELECT json_extract(a.props, '$.body') AS body
		   FROM nodes a
		   JOIN edges e ON e.from_id = a.id AND e.type = 'attempted_in' AND e.valid_to IS NULL
		   JOIN nodes s ON e.to_id = s.id AND s.type = 'Session' AND s.valid_to IS NULL
		  WHERE a.type = 'Attempt' AND a.valid_to IS NULL
		    AND s.key = ?
		  ORDER BY a.ingested_at`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var body sql.NullString
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		if body.Valid && body.String != "" {
			out = append(out, body.String)
		}
	}
	return out, rows.Err()
}

// contextToCarry returns short bullets summarizing pinned context for
// the next session. v1: top-priority recent Decision nodes in the
// current repo, plus claimed-by ownership notes. Phase 5 will
// personalize this per-developer.
func contextToCarry(store *graph.Store, repoKey string) ([]string, error) {
	rows, err := store.DB().Query(
		`SELECT type, key,
		        COALESCE(json_extract(props, '$.title'), key) AS title,
		        COALESCE(json_extract(props, '$.claimed_by'), '') AS claimed_by
		   FROM nodes
		  WHERE type IN ('Decision', 'Initiative')
		    AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed', 'superseded')
		  ORDER BY ingested_at DESC
		  LIMIT 5`,
		repoKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var typ, key, title, claimedBy string
		if err := rows.Scan(&typ, &key, &title, &claimedBy); err != nil {
			return nil, err
		}
		bullet := fmt.Sprintf("%s — `%s`", title, key)
		if claimedBy != "" {
			bullet += " (claimed by " + claimedBy + ")"
		}
		out = append(out, bullet)
	}
	return out, rows.Err()
}

// --- helpers ---------------------------------------------------------------

func shortSha(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

// pluralizeWorkType returns a methodology-aware plural noun for the
// canonical work type (e.g. "feature" → "features", or "stories" under
// agile-scrum). When the active vocabulary has no override, falls
// through to the canonical type literal with a trailing "s". When vocab
// is nil (engineering / legacy), preserves the canonical phrasing —
// "features" — exactly as the previous code path emitted.
func pluralizeWorkType(v *vocabulary.Vocabulary, canonicalType string) string {
	if v == nil {
		return canonicalType + "s"
	}
	// The canonical engineering type "feature" maps to spec.feature
	// in the vocabulary kinds table. Try the kind-level mapping first
	// since it is the most specific.
	display := v.Display("spec", canonicalType)
	if display == "" || strings.EqualFold(display, "spec") {
		return canonicalType + "s"
	}
	// Cheap English pluralization. Vocabularies pick clean singular
	// nouns (Story, Scope, Card) so naive +s is correct in practice;
	// the special case for words ending in 'y' covers "Story" → "Stories".
	if strings.HasSuffix(display, "y") && !strings.HasSuffix(display, "ay") && !strings.HasSuffix(display, "ey") && !strings.HasSuffix(display, "oy") && !strings.HasSuffix(display, "uy") {
		return strings.ToLower(display[:len(display)-1]) + "ies"
	}
	return strings.ToLower(display) + "s"
}
