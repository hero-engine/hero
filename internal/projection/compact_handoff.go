package projection

import (
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

// CompactHandoffOptions narrows graph queries to a single session for
// the post-compaction handoff projector. SessionID + SessionStart bound
// the "this session only" filter; ActiveSpecSlug, when non-empty, also
// pulls in cross-session events anchored to the same spec (Decisions,
// AC flips, commits) so collaborators on the same work see each other's
// activity carry forward intentionally.
type CompactHandoffOptions struct {
	SessionID      string
	SessionStart   time.Time // session start time; events before this are excluded
	ActiveSpecSlug string    // optional carryover anchor
	RepoKey        string    // optional partition for spec-anchored events
	FilesLimit     int       // default 10
	DecisionsLimit int       // default 5
}

// CompactDecision is a Decision-node text bullet for the handoff.
type CompactDecision struct {
	Title string
	Body  string
	When  string // RFC3339 from valid_from
}

// CompactFileTouch summarises a file that received activity this session.
type CompactFileTouch struct {
	Path  string
	Count int
}

// SessionEvents holds the session-scoped + spec-anchored event slices
// the compact handoff needs from the graph. All slices are nil-safe;
// callers should handle empty as "no events".
type SessionEvents struct {
	Decisions    []CompactDecision
	FilesTouched []CompactFileTouch
	// TotalEvents is the count of session-tagged graph rows visited
	// while building the slices above. Callers use this to detect the
	// "fewer than 3 events for this session — go to fallback path" edge
	// case described in the spec.
	TotalEvents int
}

// CollectSessionEvents runs the session-filtered query described in
// the next-compact-handoff spec:
//
//   - Events tagged with this session_id since session_start (UserAsk,
//     NextSuggestion, SessionReflection, Attempt — handoff package
//     nodes whose props.session_id matches the current session).
//   - UNION events with spec_slug = ActiveSpecSlug since session_start
//     where node type is in {Decision, Criterion (status flip),
//     Commit-on-spec} — the cross-session collaboration carryover.
//
// Returns an empty SessionEvents (not nil) on any non-fatal query
// shortfall so callers can render the fallback path without nil checks.
func CollectSessionEvents(store *graph.Store, opts CompactHandoffOptions) (*SessionEvents, error) {
	out := &SessionEvents{}
	if store == nil || opts.SessionID == "" {
		return out, nil
	}
	if opts.FilesLimit <= 0 {
		opts.FilesLimit = 10
	}
	if opts.DecisionsLimit <= 0 {
		opts.DecisionsLimit = 5
	}

	since := opts.SessionStart.UTC().Format(time.RFC3339)

	decisions, err := queryCompactDecisions(store, opts.SessionID, opts.ActiveSpecSlug, since, opts.DecisionsLimit)
	if err == nil {
		out.Decisions = decisions
	}

	files, total, err := queryCompactFilesTouched(store, opts.SessionID, opts.FilesLimit)
	if err == nil {
		out.FilesTouched = files
		out.TotalEvents = total + len(decisions)
	}

	return out, nil
}

func queryCompactDecisions(store *graph.Store, sessionID, specSlug, since string, limit int) ([]CompactDecision, error) {
	q := `
		SELECT key,
		       COALESCE(json_extract(props, '$.title'), key) AS title,
		       COALESCE(json_extract(props, '$.body'),
		                json_extract(props, '$.text'), '') AS body,
		       valid_from
		  FROM nodes
		 WHERE type = 'Decision' AND valid_to IS NULL
		   AND valid_from >= ?
		   AND (
		     COALESCE(json_extract(props, '$.session_id'), '') = ?
		     OR (
		       ? <> '' AND
		       COALESCE(json_extract(props, '$.spec_slug'), '') = ?
		     )
		   )
		 ORDER BY valid_from DESC
		 LIMIT ?`
	rows, err := store.DB().Query(q, since, sessionID, specSlug, specSlug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompactDecision
	for rows.Next() {
		var key string
		var title, body, validFrom sql.NullString
		if err := rows.Scan(&key, &title, &body, &validFrom); err != nil {
			return nil, err
		}
		t := title.String
		if t == "" {
			t = key
		}
		out = append(out, CompactDecision{
			Title: t,
			Body:  body.String,
			When:  validFrom.String,
		})
	}
	return out, rows.Err()
}

// queryCompactFilesTouched returns the top-N file paths mentioned in
// session-tagged Attempt / Reflection / Ask / Suggestion bodies, plus
// the total event count visited so callers can detect the "fewer than
// 3 events" fallback condition.
func queryCompactFilesTouched(store *graph.Store, sessionID string, limit int) ([]CompactFileTouch, int, error) {
	q := `
		SELECT COALESCE(json_extract(props, '$.body'),
		                json_extract(props, '$.text'), '') AS body
		  FROM nodes
		 WHERE valid_to IS NULL
		   AND type IN ('Attempt', 'SessionReflection', 'UserAsk', 'NextSuggestion')
		   AND COALESCE(json_extract(props, '$.session_id'), '') = ?`
	rows, err := store.DB().Query(q, sessionID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	total := 0
	for rows.Next() {
		total++
		var body sql.NullString
		if err := rows.Scan(&body); err != nil {
			return nil, 0, err
		}
		for _, tok := range extractCompactPathTokens(body.String) {
			counts[tok]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	type pair struct {
		path  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for p, c := range counts {
		pairs = append(pairs, pair{p, c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].path < pairs[j].path
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]CompactFileTouch, len(pairs))
	for i, p := range pairs {
		out[i] = CompactFileTouch{Path: p.path, Count: p.count}
	}
	return out, total, nil
}

// extractCompactPathTokens returns file-path-shaped tokens found in s.
// Conservative shape rule: token must contain a slash and a known
// code/text extension.
func extractCompactPathTokens(s string) []string {
	if s == "" {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, raw := range strings.Fields(s) {
		tok := strings.Trim(raw, "`\"'(),;:.")
		if !compactLooksLikePath(tok) {
			continue
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

func compactLooksLikePath(s string) bool {
	if !strings.Contains(s, "/") {
		return false
	}
	ext := filepath.Ext(s)
	if ext == "" {
		return false
	}
	switch ext {
	case ".go", ".java", ".groovy", ".gradle",
		".rs", ".js", ".ts", ".jsx", ".tsx",
		".mjs", ".cjs", ".mts",
		".py", ".rb", ".sql", ".md",
		".json", ".yaml", ".yml", ".toml",
		".xml", ".html", ".css", ".scss",
		".sh", ".bash",
		".kt", ".kts", ".swift", ".c", ".h",
		".cpp", ".hpp", ".cs", ".php":
		return true
	}
	return false
}
