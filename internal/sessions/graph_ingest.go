package sessions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteGraph maps sessions into the unified knowledge graph.
//
// Each session becomes a Session node keyed by its ID. If the session
// has a SpecSlug, an edge `mentions` is emitted to whichever spec node
// matches that slug across the supported spec types.
//
// repoKey stamps the partition column. Sessions anchor to the primary
// repo a developer is working in; cross-repo session tracking is
// opt-in (phase 8+).
//
// Spec resolution is best-effort: if the target spec hasn't been
// ingested yet, the edge is skipped — a later WriteGraph after spec
// ingest will fill it in.
func WriteGraph(sess []*Session, repoKey string, store *graph.Store) (*GraphWriteSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("sessions: WriteGraph requires non-nil Store")
	}
	source := map[string]any{"kind": "session"}

	summary := &GraphWriteSummary{}
	for _, s := range sess {
		if s == nil || s.ID == "" {
			continue
		}
		props := map[string]any{
			"hero_calls":      s.HeroCalls,
			"specs_completed": s.SpecsDone,
		}
		if s.Name != "" {
			props["name"] = s.Name
		}
		if s.Agent != "" {
			props["agent"] = s.Agent
		}
		if !s.Start.IsZero() {
			props["start"] = s.Start.UTC().Format("2006-01-02T15:04:05Z")
		}
		if s.End != nil {
			props["end"] = s.End.UTC().Format("2006-01-02T15:04:05Z")
		}
		if s.SpecSlug != "" {
			props["spec_slug"] = s.SpecSlug
		}

		hash := hashSession(s)
		sessID, err := store.UpsertNode(&graph.Node{
			Type:        "Session",
			Domain:      "engineering",
			Key:         s.ID,
			Props:       props,
			Repo:        repoKey,
			ContentHash: hash,
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Session %s: %w", s.ID, err)
		}
		summary.Sessions++

		if s.SpecSlug != "" {
			if specID, ok := resolveSpecBySlug(store, s.SpecSlug); ok {
				if _, err := store.UpsertEdge(&graph.Edge{
					FromID: sessID, ToID: specID, Type: "mentions",
					Repo:   repoKey,
					Source: source,
				}); err != nil {
					return nil, fmt.Errorf("upsert edge Session→Spec: %w", err)
				}
				summary.Edges++
			}
		}
	}
	return summary, nil
}

// GraphWriteSummary reports counts written by WriteGraph.
type GraphWriteSummary struct {
	Sessions int
	Edges    int
}

// resolveSpecBySlug looks up any spec-typed node with the given slug.
func resolveSpecBySlug(store *graph.Store, slug string) (int64, bool) {
	for _, t := range []string{"Feature", "Initiative", "Bug", "Decision",
		"Convention", "Rule", "ContextDoc", "Note", "External"} {
		if id, err := store.GetNodeID(t, slug); err == nil {
			return id, true
		}
	}
	return 0, false
}

func hashSession(s *Session) string {
	parts := []byte(s.ID + "|" + s.Name + "|" + s.Agent + "|" + s.SpecSlug)
	parts = append(parts, []byte(s.Start.UTC().Format("2006-01-02T15:04:05Z"))...)
	if s.End != nil {
		parts = append(parts, []byte(s.End.UTC().Format("2006-01-02T15:04:05Z"))...)
	}
	parts = append(parts, []byte(fmt.Sprintf("|%d|%d", s.HeroCalls, s.SpecsDone))...)
	sum := sha256.Sum256(parts)
	return hex.EncodeToString(sum[:])
}
