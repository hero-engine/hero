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
	NodeUserAsk            = "UserAsk"
	NodeNextSuggestion     = "NextSuggestion"
	NodeSessionReflection  = "SessionReflection"

	// SourceKind tags the Source.kind field on every node this
	// package emits, so ingest paths can recognise round-trip data.
	SourceKind = "handoff"
)

// UserAsk is the most recent prompt the user sent to the agent in
// this user's working session. Singleton per user — each new ask
// supersedes the prior via bitemporal valid_to.
type UserAsk struct {
	User      string // user slug — same shape as nextUserSlug() returns
	Text      string // verbatim or paraphrased prompt
	SessionID string // optional — links to a Session node when known
	UpdatedAt string // RFC3339; populated by Record on write
}

// NextSuggestion is the agent's recommended next prompt for this
// user. Singleton per user — each new suggestion supersedes prior.
type NextSuggestion struct {
	User      string
	Text      string // suggested prompt (one paragraph, agent's voice)
	Rationale string // optional — why this is the right next move
	SessionID string
	UpdatedAt string
}

// SessionReflection is a mid-session lesson worth surfacing in the
// next handoff but not yet promoted to a durable Note. Multiple co-
// existing rows per user; query latest N within a session.
type SessionReflection struct {
	User      string
	Text      string
	Tags      []string
	SessionID string
	UpdatedAt string
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
	if ask.Text == "" {
		// Clearing the ask: invalidate the current row if any.
		return store.InvalidateNode(NodeUserAsk, ask.User)
	}
	props := map[string]any{
		"text":       ask.Text,
		"session_id": ask.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeUserAsk,
		Key:         ask.User,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(ask.Text, ask.SessionID),
		Source:      map[string]any{"kind": SourceKind, "type": NodeUserAsk},
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
	if sug.Text == "" {
		return store.InvalidateNode(NodeNextSuggestion, sug.User)
	}
	props := map[string]any{
		"text":       sug.Text,
		"rationale":  strings.TrimSpace(sug.Rationale),
		"session_id": sug.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeNextSuggestion,
		Key:         sug.User,
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
	key := ref.User + ":" + stamp
	props := map[string]any{
		"text":       ref.Text,
		"tags":       ref.Tags,
		"session_id": ref.SessionID,
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        NodeSessionReflection,
		Key:         key,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashFields(ref.Text, ref.SessionID, strings.Join(ref.Tags, ",")),
		Source:      map[string]any{"kind": SourceKind, "type": NodeSessionReflection},
	})
	return err
}

// LatestAsk returns the current UserAsk for the user, or nil if
// none. Returns (nil, nil) on a clean miss.
func LatestAsk(store *graph.Store, user string) (*UserAsk, error) {
	n, err := store.GetNode(NodeUserAsk, user)
	if err != nil {
		if err == graph.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return askFromNode(n), nil
}

// LatestSuggestion returns the current NextSuggestion for the user,
// or nil if none.
func LatestSuggestion(store *graph.Store, user string) (*NextSuggestion, error) {
	n, err := store.GetNode(NodeNextSuggestion, user)
	if err != nil {
		if err == graph.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return suggestionFromNode(n), nil
}

// RecentReflections returns up to limit most-recent reflections for
// the user, newest first. Pass limit <= 0 to return all.
func RecentReflections(store *graph.Store, user string, limit int) ([]SessionReflection, error) {
	all, err := store.ListNodesByType(NodeSessionReflection)
	if err != nil {
		return nil, err
	}
	prefix := user + ":"
	var out []SessionReflection
	for _, n := range all {
		if !strings.HasPrefix(n.Key, prefix) {
			continue
		}
		out = append(out, *reflectionFromNode(n))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func askFromNode(n *graph.Node) *UserAsk {
	if n == nil {
		return nil
	}
	return &UserAsk{
		User:      n.Key,
		Text:      stringProp(n.Props, "text"),
		SessionID: stringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func suggestionFromNode(n *graph.Node) *NextSuggestion {
	if n == nil {
		return nil
	}
	return &NextSuggestion{
		User:      n.Key,
		Text:      stringProp(n.Props, "text"),
		Rationale: stringProp(n.Props, "rationale"),
		SessionID: stringProp(n.Props, "session_id"),
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
		Text:      stringProp(n.Props, "text"),
		Tags:      stringSliceProp(n.Props, "tags"),
		SessionID: stringProp(n.Props, "session_id"),
		UpdatedAt: n.ValidFrom,
	}
}

func stringProp(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
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
