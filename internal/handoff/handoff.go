// Package handoff is the read/write layer for the per-user durable
// handoff state — Last user ask, Suggested next prompt, in-flight
// Session reflections.
//
// These are the agent-narrative pieces of NEXT.md that previously
// lived as hand-written markdown bullets at the top of the file and
// drifted as a result. Promoting them to structured graph nodes
// gives them three things:
//
//   - Federation via existing Cloud sync (when configured).
//   - Bitemporal time travel — "what was suggested-next on Tuesday?"
//     is a query, not an archaeology dig.
//   - Cross-machine continuity through the graph (with Cloud) or via
//     round-trip through .hero/next/<user>.md (without Cloud).
//
// Phase 2 of next-as-projection. Phase 3 wires the field-grab CLI
// (hero next suggest / ask / reflection) on top of this package.
// Phase 4 wires the projection that renders these into NEXT.md and
// .hero/next/<user>.md.
package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

// Graph node type names. Kept as exported constants so projections,
// CLI, and round-trip ingest can reference them without re-typing
// the strings.
const (
	NodeUserAsk           = "UserAsk"
	NodeNextSuggestion    = "NextSuggestion"
	NodeSessionReflection = "SessionReflection"
	NodeSessionGoal       = "SessionGoal"

	// SourceKind tags the Source.kind field on every node this
	// package emits, so ingest paths can recognise round-trip data.
	SourceKind = "handoff"
)

// Goal source identifiers. They form a strict priority ladder
// (auto-window < auto-embed < marker < manual): an incoming write only
// overwrites the current goal when its priority is >= the existing
// one, so an automatic pass refreshes a lower/equal source but never
// clobbers a deliberately-set higher one.
const (
	GoalSourceAutoWindow = "auto-window"
	GoalSourceAutoEmbed  = "auto-embed"
	GoalSourceMarker     = "marker"
	GoalSourceManual     = "manual"
)

// goalSourcePriority maps a goal source to its rung on the priority
// ladder. Unknown sources sort as auto-window (the lowest rung) so a
// malformed write can never clobber a real marker/manual goal.
func goalSourcePriority(source string) int {
	switch source {
	case GoalSourceManual:
		return 3
	case GoalSourceMarker:
		return 2
	case GoalSourceAutoEmbed:
		return 1
	case GoalSourceAutoWindow:
		return 0
	default:
		return 0
	}
}

// UserAsk is the most recent prompt the user sent to the agent in
// this user's working session. Singleton per (user, repo, domain) —
// each new ask within that triple supersedes the prior via bitemporal
// valid_to. Different active domains in the same workspace keep their
// own singletons so a PM agent's ask and an engineering agent's ask
// don't overwrite each other.
type UserAsk struct {
	User      string // user slug — same shape as nextUserSlug() returns
	Domain    string // domain partition; "" defaults to "engineering"
	Text      string // verbatim or paraphrased prompt
	SessionID string // optional — links to a Session node when known
	UpdatedAt string // RFC3339; populated by Record on write
}

// NextSuggestion is the agent's recommended next prompt for this
// user. Singleton per (user, repo, domain) — each new suggestion
// within that triple supersedes prior.
type NextSuggestion struct {
	User      string
	Domain    string // domain partition; "" defaults to "engineering"
	Text      string // suggested prompt (one paragraph, agent's voice)
	Rationale string // optional — why this is the right next move
	SessionID string
	UpdatedAt string
}

// SessionReflection is a mid-session lesson worth surfacing in the
// next handoff but not yet promoted to a durable Note. Multiple co-
// existing rows per (user, repo, domain); query latest N within a
// session.
type SessionReflection struct {
	User      string
	Domain    string // domain partition; "" defaults to "engineering"
	Text      string
	Tags      []string
	SessionID string
	UpdatedAt string
}

// SessionGoal is the session's durable intent — the WHY the work is
// happening — captured automatically from the transcript's opening
// window and refined by optional better signals (marker, manual).
// Singleton per (user, repo, domain), same class as UserAsk: a distinct
// node so the goal and the volatile last-ask can coexist instead of
// clobbering each other on one singleton. Source records which rung of
// the priority ladder set the current value, and drives both the render
// framing (soft for guesses, asserted for statements) and the priority
// guard in RecordGoal.
type SessionGoal struct {
	User      string
	Domain    string // domain partition; "" defaults to "engineering"
	Text      string // verbatim/window goal candidate (truncated)
	Source    string // one of GoalSource* — drives priority + framing
	SessionID string
	UpdatedAt string
}

// resolveDomain provides the "engineering" fallback for handoff
// records that don't carry an explicit domain. Mirrors the same
// rule as graph.DomainFor's IntrinsicActive branch.
func resolveDomain(d string) string {
	if d == "" {
		return "engineering"
	}
	return d
}

// singletonKey builds the per-(user, domain) singleton key for the
// UserAsk and NextSuggestion node types per DSKG AC #11.
func singletonKey(user, domain string) string {
	return user + ":" + resolveDomain(domain)
}

// RecordAsk upserts the UserAsk singleton for the given user. Empty
// text clears the prior ask (no-op if none exists).
func RecordAsk(store *graph.Store, repoKey string, ask UserAsk) error {
	if store == nil {
		return fmt.Errorf("handoff: nil store")
	}
	if ask.User == "" {
		return fmt.Errorf("handoff: UserAsk requires User")
	}
	ask.Text = strings.TrimSpace(ask.Text)
	domain := resolveDomain(ask.Domain)
	key := singletonKey(ask.User, domain)
	if ask.Text == "" {
		// Clearing the ask: invalidate the current row if any.
		return store.InvalidateNode(NodeUserAsk, key, repoKey)
	}
	props := map[string]any{
		"text":       ask.Text,
		"session_id": ask.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeUserAsk,
		Domain:      domain,
		Key:         key,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(ask.Text, ask.SessionID),
		Source:      map[string]any{"kind": SourceKind, "type": NodeUserAsk},
	})
	return err
}

// RecordGoal upserts the SessionGoal singleton for the given user,
// enforcing the priority ladder. Empty text clears the prior goal.
//
// Priority guard — the one rule that implements the whole ladder: read
// the current row; write only when priority(incoming) >= priority(existing)
// (or no row exists). So an auto-window pass refreshes a window goal but
// never clobbers a marker/manual one; a marker overrides window/embed but
// not manual; manual wins over all. A same-source rewrite with identical
// content is a no-op via the content hash (UpsertNode skips no-op writes),
// so repeated checkpoints don't churn.
func RecordGoal(store *graph.Store, repoKey string, goal SessionGoal) error {
	if store == nil {
		return fmt.Errorf("handoff: nil store")
	}
	if goal.User == "" {
		return fmt.Errorf("handoff: SessionGoal requires User")
	}
	goal.Text = strings.TrimSpace(goal.Text)
	domain := resolveDomain(goal.Domain)
	key := singletonKey(goal.User, domain)
	if goal.Text == "" {
		// Clearing the goal: invalidate the current row if any.
		return store.InvalidateNode(NodeSessionGoal, key, repoKey)
	}
	if goal.Source == "" {
		goal.Source = GoalSourceAutoWindow
	}

	// Priority guard: never let a lower-priority source clobber a
	// higher one. Same priority is allowed through so an equal-source
	// refresh (and the content-hash no-op) works as expected.
	if existing, err := scanLatestSingleton(store, NodeSessionGoal, key, repoKey); err == nil && existing != nil {
		existingSource := graph.StringProp(existing.Props, "source")
		if goalSourcePriority(goal.Source) < goalSourcePriority(existingSource) {
			return nil
		}
	}

	props := map[string]any{
		"text":       goal.Text,
		"source":     goal.Source,
		"session_id": goal.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeSessionGoal,
		Domain:      domain,
		Key:         key,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(goal.Text, goal.Source, goal.SessionID),
		Source:      map[string]any{"kind": SourceKind, "type": NodeSessionGoal},
	})
	return err
}

// RecordSuggestion upserts the NextSuggestion singleton for the
// given user. Empty text clears the prior suggestion.
func RecordSuggestion(store *graph.Store, repoKey string, sug NextSuggestion) error {
	if store == nil {
		return fmt.Errorf("handoff: nil store")
	}
	if sug.User == "" {
		return fmt.Errorf("handoff: NextSuggestion requires User")
	}
	sug.Text = strings.TrimSpace(sug.Text)
	domain := resolveDomain(sug.Domain)
	key := singletonKey(sug.User, domain)
	if sug.Text == "" {
		return store.InvalidateNode(NodeNextSuggestion, key, repoKey)
	}
	props := map[string]any{
		"text":       sug.Text,
		"rationale":  strings.TrimSpace(sug.Rationale),
		"session_id": sug.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeNextSuggestion,
		Domain:      domain,
		Key:         key,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(sug.Text, sug.Rationale, sug.SessionID),
		Source:      map[string]any{"kind": SourceKind, "type": NodeNextSuggestion},
	})
	return err
}

// RecordReflection inserts a new SessionReflection for the given
// user. Each reflection is its own node (unique key with timestamp
// suffix), so multiple co-exist within a session.
func RecordReflection(store *graph.Store, repoKey string, ref SessionReflection) error {
	if store == nil {
		return fmt.Errorf("handoff: nil store")
	}
	if ref.User == "" {
		return fmt.Errorf("handoff: SessionReflection requires User")
	}
	ref.Text = strings.TrimSpace(ref.Text)
	if ref.Text == "" {
		return fmt.Errorf("handoff: SessionReflection requires Text")
	}
	// Sub-second precision avoids key collisions when multiple
	// reflections are recorded in the same second (e.g. during a
	// round-trip ingest of an existing handoff file).
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	domain := resolveDomain(ref.Domain)
	key := ref.User + ":" + domain + ":" + stamp
	props := map[string]any{
		"text":       ref.Text,
		"tags":       ref.Tags,
		"session_id": ref.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeSessionReflection,
		Domain:      domain,
		Key:         key,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(ref.Text, ref.SessionID, strings.Join(ref.Tags, ",")),
		Source:      map[string]any{"kind": SourceKind, "type": NodeSessionReflection},
	})
	return err
}

// LatestAsk returns the current UserAsk for the (user, repo, domain)
// triple, or nil if none. Returns (nil, nil) on a clean miss. The
// repoKey filter is what keeps an ask recorded in repo A from
// surfacing in repo B's handoff projection; the domain filter keeps
// PM and engineering singletons distinct.
//
// An empty `domain` arg resolves to "engineering" — pre-domain
// callers see no behavior change.
func LatestAsk(store *graph.Store, user, repoKey, domain string) (*UserAsk, error) {
	if store == nil {
		return nil, fmt.Errorf("handoff: nil store")
	}
	n, err := scanLatestSingleton(store, NodeUserAsk, singletonKey(user, domain), repoKey)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return askFromNode(n), nil
}

// LatestGoal returns the current SessionGoal for the (user, repo,
// domain) triple, or nil if none. Mirrors LatestAsk; returns the text
// plus the source so renderers can pick soft-vs-asserted framing.
func LatestGoal(store *graph.Store, user, repoKey, domain string) (*SessionGoal, error) {
	if store == nil {
		return nil, fmt.Errorf("handoff: nil store")
	}
	n, err := scanLatestSingleton(store, NodeSessionGoal, singletonKey(user, domain), repoKey)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return goalFromNode(n), nil
}

// LatestSuggestion returns the current NextSuggestion for the
// (user, repo, domain) triple, or nil if none.
func LatestSuggestion(store *graph.Store, user, repoKey, domain string) (*NextSuggestion, error) {
	if store == nil {
		return nil, fmt.Errorf("handoff: nil store")
	}
	n, err := scanLatestSingleton(store, NodeNextSuggestion, singletonKey(user, domain), repoKey)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return suggestionFromNode(n), nil
}

// RecentReflections returns up to limit most-recent reflections for
// the (user, repo, domain) triple, newest first. Pass limit <= 0 to
// return all.
//
// An empty `domain` arg resolves to "engineering".
func RecentReflections(store *graph.Store, user, repoKey, domain string, limit int) ([]SessionReflection, error) {
	if store == nil {
		return nil, fmt.Errorf("handoff: nil store")
	}
	prefix := user + ":" + resolveDomain(domain) + ":"
	rows, err := store.DB().Query(
		`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes
		  WHERE type = ? AND repo = ? AND valid_to IS NULL AND key LIKE ?
		  ORDER BY valid_from DESC`,
		NodeSessionReflection, repoKey, prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionReflection
	for rows.Next() {
		n, err := graph.ScanNode(rows)
		if err != nil {
			return nil, err
		}
		// Defensive: the LIKE prefix matches "user:..." but also any
		// key whose prefix happens to share the same leading bytes.
		// Re-check the exact "user:" prefix in Go.
		if !strings.HasPrefix(n.Key, prefix) {
			continue
		}
		out = append(out, *reflectionFromNode(n))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// scanLatestSingleton returns the current (valid_to IS NULL) row for
// (type, key, repo) — used by LatestAsk and LatestSuggestion. Returns
// (nil, nil) when no row matches so callers can treat absence as a
// clean miss rather than an error.
func scanLatestSingleton(store *graph.Store, typ, key, repoKey string) (*graph.Node, error) {
	row := store.DB().QueryRow(
		`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes
		  WHERE type = ? AND key = ? AND repo = ? AND valid_to IS NULL`,
		typ, key, repoKey,
	)
	n, err := graph.ScanNode(row)
	if err == graph.ErrNotFound {
		return nil, nil
	}
	return n, err
}

func askFromNode(n *graph.Node) *UserAsk {
	if n == nil {
		return nil
	}
	user, _, _ := strings.Cut(n.Key, ":")
	return &UserAsk{
		User:      user,
		Domain:    n.Domain,
		Text:      graph.StringProp(n.Props, "text"),
		SessionID: graph.StringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func goalFromNode(n *graph.Node) *SessionGoal {
	if n == nil {
		return nil
	}
	user, _, _ := strings.Cut(n.Key, ":")
	source := graph.StringProp(n.Props, "source")
	if source == "" {
		source = GoalSourceAutoWindow
	}
	return &SessionGoal{
		User:      user,
		Domain:    n.Domain,
		Text:      graph.StringProp(n.Props, "text"),
		Source:    source,
		SessionID: graph.StringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func suggestionFromNode(n *graph.Node) *NextSuggestion {
	if n == nil {
		return nil
	}
	user, _, _ := strings.Cut(n.Key, ":")
	return &NextSuggestion{
		User:      user,
		Domain:    n.Domain,
		Text:      graph.StringProp(n.Props, "text"),
		Rationale: graph.StringProp(n.Props, "rationale"),
		SessionID: graph.StringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func reflectionFromNode(n *graph.Node) *SessionReflection {
	if n == nil {
		return nil
	}
	user, _, _ := strings.Cut(n.Key, ":")
	return &SessionReflection{
		User:      user,
		Domain:    n.Domain,
		Text:      graph.StringProp(n.Props, "text"),
		Tags:      stringSliceProp(n.Props, "tags"),
		SessionID: graph.StringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func stringSliceProp(m map[string]any, k string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[k].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func hashFields(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
