package projection

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
)

// UserHandoffOptions tunes the per-user handoff projection.
type UserHandoffOptions struct {
	User        string // user slug — required
	RepoKey     string // partition for "your recent activity" filtering
	SessionID   string // anchors session-scoped sections
	ReflectionN int    // most recent reflections (default 5)
	CommitsN    int    // your recent commits (default 6)
}

// UserHandoffMD renders .hero/next/<user>.md from the user-graph
// nodes in the handoff package (UserAsk, NextSuggestion,
// SessionReflection) plus user-attributed activity from the project
// graph (your commits, your AC flips, your failed attempts).
//
// The result is stable: same graph state in → same markdown out.
// Empty sections render explicitly so the projection is always a
// complete handoff document, not a sparse one.
func UserHandoffMD(store *graph.Store, opts UserHandoffOptions) (string, error) {
	if store == nil {
		return "", fmt.Errorf("projection: UserHandoffMD requires non-nil Store")
	}
	if opts.User == "" {
		return "", fmt.Errorf("projection: UserHandoffMD requires User")
	}
	if opts.ReflectionN == 0 {
		opts.ReflectionN = 5
	}
	if opts.CommitsN == 0 {
		opts.CommitsN = 6
	}

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "user: %s\n", opts.User)
	fmt.Fprintf(&b, "updated: %s\n", time.Now().UTC().Format(time.RFC3339))
	if opts.SessionID != "" {
		fmt.Fprintf(&b, "session: %s\n", opts.SessionID)
	}
	if opts.RepoKey != "" {
		fmt.Fprintf(&b, "repo: %s\n", opts.RepoKey)
	}
	b.WriteString("---\n\n")

	fmt.Fprintf(&b, "# %s's handoff\n\n", opts.User)

	// Last user ask. Same staleness model as NextSuggestion: a commit
	// landing after the ask suggests the ask's premise has shifted.
	// We don't auto-derive a fallback (you can't synthesize a user's
	// voice from project state), but we do flag staleness inline so
	// the reader can judge whether to act on it or wait for a refresh.
	b.WriteString("## Last user ask\n\n")
	if ask, _ := handoff.LatestAsk(store, opts.User); ask != nil && ask.Text != "" {
		fmt.Fprintf(&b, "> %s\n", indentQuote(ask.Text))
		if note := stalenessNote(store, opts.RepoKey, ask.UpdatedAt); note != "" {
			fmt.Fprintf(&b, "\n_%s_\n", note)
		}
	} else {
		b.WriteString("_(none recorded — `hero next ask \"...\"` to set)_\n")
	}
	b.WriteString("\n")

	// Suggested next prompt — agent-emitted suggestion when fresh,
	// auto-derived from project state when stale or missing. Always
	// renders something current; never shows a suggestion superseded
	// by commits the agent didn't refresh against.
	b.WriteString("## Suggested next prompt\n\n")
	text, rationale, source := PickUserSuggestion(store, opts.User, opts.RepoKey)
	if text != "" {
		fmt.Fprintf(&b, "> %s\n", indentQuote(text))
		if rationale != "" {
			fmt.Fprintf(&b, "\n_Rationale: %s_\n", oneLine(rationale))
		}
		if source != SuggestionFromAgent {
			fmt.Fprintf(&b, "\n_Source: %s — `hero next suggest \"...\"` to override._\n", source)
		}
	} else {
		b.WriteString("_(none — `hero next suggest \"...\"` to set, or open a Feature to derive one)_\n")
	}
	b.WriteString("\n")

	// Recent reflections
	b.WriteString("## Recent reflections\n\n")
	if refs, _ := handoff.RecentReflections(store, opts.User, opts.ReflectionN); len(refs) > 0 {
		for _, r := range refs {
			fmt.Fprintf(&b, "- %s\n", oneLine(r.Text))
		}
	} else {
		b.WriteString("_(none yet)_\n")
	}
	b.WriteString("\n")

	// Tried and failed (this session) — kept here because it's
	// session-scoped user state, not project state.
	b.WriteString("## Tried and failed (this session)\n\n")
	attempts, err := attemptsForSession(store, opts.SessionID)
	if err != nil {
		return "", fmt.Errorf("attempts: %w", err)
	}
	if len(attempts) == 0 {
		b.WriteString("Nothing this session.\n")
	} else {
		for _, a := range attempts {
			fmt.Fprintf(&b, "- %s\n", oneLine(a))
		}
	}
	b.WriteString("\n")

	// Your recent activity — author-attributed commits in this repo.
	b.WriteString("## Your recent activity\n\n")
	mine, err := userRecentCommits(store, opts.RepoKey, opts.User, opts.CommitsN)
	if err != nil {
		return "", fmt.Errorf("recent activity: %w", err)
	}
	if len(mine) == 0 {
		b.WriteString("_(no commits attributed to you in this repo's graph)_\n")
	} else {
		for _, c := range mine {
			fmt.Fprintf(&b, "- `%s` — %s\n", shortSha(c.sha), oneLine(c.subject))
		}
	}
	b.WriteString("\n")

	return b.String(), nil
}

// SuggestionSource describes where a suggestion came from. Lets the
// projection mark derived ones so the user knows they're auto-filled
// and the agent knows to override with a real one when possible.
type SuggestionSource string

const (
	SuggestionFromAgent       SuggestionSource = "agent"
	SuggestionFromOpenFeature SuggestionSource = "auto-derived from open feature"
	SuggestionFromInitiative  SuggestionSource = "auto-derived from active initiative"
)

// PickUserSuggestion returns the best-current suggestion text for
// the user, with three priorities:
//
//  1. Agent-emitted NextSuggestion when fresh (no commits landed
//     since it was set).
//  2. Highest-priority open Feature title (user-voice) when the
//     agent suggestion is stale or missing.
//  3. Open Initiative title when no open Features exist.
//
// Returns (text, rationale, source). text is empty when nothing
// useful can be derived. The CLI and the projection both call this
// so `hero next suggest` and the rendered .hero/next/<user>.md
// always show the same answer.
func PickUserSuggestion(store *graph.Store, user, repoKey string) (text, rationale string, source SuggestionSource) {
	if store == nil {
		return "", "", ""
	}
	sug, _ := handoff.LatestSuggestion(store, user)
	if sug != nil && sug.Text != "" && !suggestionStale(store, repoKey, sug.UpdatedAt) {
		return sug.Text, sug.Rationale, SuggestionFromAgent
	}
	// Agent suggestion is stale or missing — derive from project state.
	if features, err := openFeaturesByPriority(store, repoKey, 1); err == nil && len(features) > 0 {
		f := features[0]
		title := f.title
		if title == "" {
			title = f.slug
		}
		return fmt.Sprintf("let's tackle %s", title),
			fmt.Sprintf("highest-priority open feature: %s (`%s`)", title, f.slug),
			SuggestionFromOpenFeature
	}
	if init := topOpenInitiative(store, repoKey); init != "" {
		return fmt.Sprintf("pick the next phase of %s", init), "", SuggestionFromInitiative
	}
	return "", "", ""
}

// stalenessNote returns a one-line "(possibly stale — N commits since,
// last set X ago)" footer when the entry's timestamp is older than the
// most recent commit in this repo. Empty string when fresh. Caller
// chooses how to present (italic footer for asks, "Source:" footer for
// suggestions).
func stalenessNote(store *graph.Store, repoKey, entryTime string) string {
	if entryTime == "" {
		return ""
	}
	entryAt, err := time.Parse(time.RFC3339, entryTime)
	if err != nil {
		return ""
	}
	var (
		latestCommit sql.NullString
		commitCount  sql.NullInt64
	)
	err = store.DB().QueryRow(
		`SELECT MAX(datetime(json_extract(props, '$.date'))),
		        SUM(CASE WHEN datetime(json_extract(props, '$.date')) > datetime(?) THEN 1 ELSE 0 END)
		   FROM nodes
		  WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL`,
		entryAt.UTC().Format("2006-01-02 15:04:05"),
		repoKey,
	).Scan(&latestCommit, &commitCount)
	if err != nil || !latestCommit.Valid || latestCommit.String == "" {
		return ""
	}
	commitAt, err := time.Parse("2006-01-02 15:04:05", latestCommit.String)
	if err != nil {
		return ""
	}
	if !commitAt.After(entryAt) {
		return ""
	}
	since := time.Since(entryAt)
	n := int64(0)
	if commitCount.Valid {
		n = commitCount.Int64
	}
	return fmt.Sprintf("possibly stale — %d commit(s) since, last set %s ago", n, humanDuration(since))
}

// humanDuration renders a coarse "Xh Ym" / "Xd Yh" age. Mirrors the
// helper in checkpoint.go but lives here so the projection package
// stays self-contained.
func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		if m := int(d.Minutes()) - h*60; m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	default:
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) - days*24
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
}

// suggestionStale returns true if a Commit node has landed in this
// repo since the suggestion's valid_from. Agent suggestions emitted
// before a commit landed are treated as superseded — the agent
// predicted a next move and the work moved past it.
//
// Commit dates land in two formats from git: "2026-04-29T03:55:07Z"
// (UTC) and "2026-04-28T21:55:07-06:00" (with TZ offset). Both
// represent the same instants but compare as strings differently.
// SQLite's datetime() normalises both to local-naive form, so MAX(...)
// returns the latest commit regardless of how git wrote the date.
func suggestionStale(store *graph.Store, repoKey, suggestionTime string) bool {
	if suggestionTime == "" {
		return true
	}
	var latest sql.NullString
	err := store.DB().QueryRow(
		`SELECT MAX(datetime(json_extract(props, '$.date')))
		   FROM nodes
		  WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL`,
		repoKey,
	).Scan(&latest)
	if err != nil || !latest.Valid || latest.String == "" {
		return false
	}
	suggestionAt, err := time.Parse(time.RFC3339, suggestionTime)
	if err != nil {
		return false
	}
	// SQLite's datetime() returns "YYYY-MM-DD HH:MM:SS" with no zone.
	// Treat as UTC (which is what datetime() does when given a Z or
	// offset input — it normalises to UTC).
	commitAt, err := time.Parse("2006-01-02 15:04:05", latest.String)
	if err != nil {
		return false
	}
	return commitAt.After(suggestionAt)
}

// topOpenInitiative returns the title of the most recently-ingested
// open Initiative for the repo, or empty string if none.
func topOpenInitiative(store *graph.Store, repoKey string) string {
	row := store.DB().QueryRow(
		`SELECT COALESCE(json_extract(props, '$.title'), key)
		   FROM nodes
		  WHERE type = 'Initiative' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed', 'superseded')
		  ORDER BY ingested_at DESC
		  LIMIT 1`,
		repoKey,
	)
	var title sql.NullString
	if err := row.Scan(&title); err != nil {
		return ""
	}
	return title.String
}

// userRecentCommits filters Commit nodes by author. Author matching
// is loose: matches if the user slug appears in author_email or
// author_name (case-insensitive), so "chet-bellows" matches both
// "user@example.com" and "username" git user.name configurations.
func userRecentCommits(store *graph.Store, repoKey, user string, limit int) ([]commitRow, error) {
	if user == "" {
		return nil, nil
	}
	pattern := "%" + strings.ToLower(user) + "%"
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.sha') AS sha,
		        json_extract(props, '$.subject') AS subject,
		        json_extract(props, '$.date') AS date
		   FROM nodes
		  WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL
		    AND (LOWER(COALESCE(json_extract(props, '$.author_email'), '')) LIKE ?
		      OR LOWER(COALESCE(json_extract(props, '$.author_name'),  '')) LIKE ?)
		  ORDER BY json_extract(props, '$.date') DESC
		  LIMIT ?`,
		repoKey, pattern, pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []commitRow
	for rows.Next() {
		var c commitRow
		var sha, subject, date sql.NullString
		if err := rows.Scan(&sha, &subject, &date); err != nil {
			return nil, err
		}
		c.sha, c.subject, c.date = sha.String, subject.String, date.String
		out = append(out, c)
	}
	return out, rows.Err()
}

// indentQuote prefixes every newline with "> " so multi-line ask /
// suggestion text renders as a coherent blockquote.
func indentQuote(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "\n") {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}
