package tasks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteSummary reports counts written by Write.
type WriteSummary struct {
	Tasks             int // Task nodes upserted (current run)
	StatusFlips       int // Tasks whose status changed since the prior current row
	BelongsTo         int // Task → Spec edges upserted
	DiscoveredAgainst int // Task → Spec (discovered_against) edges upserted
	AssignedTo        int // Task → Person edges upserted (when assignee present)
	NoOps             int // Tasks where nothing changed since the prior current row
}

// Write upserts Task nodes for one spec's parsed tasks into the graph
// and wires the canonical edge kinds:
//
//   - belongs_to        Task → Spec
//   - discovered_against Task → Spec (when the task's metadata names a sibling spec)
//   - assigned_to       Task → Person (when assignee is set)
//
// repoKey scopes federation per the same contract WriteGraph uses.
// parentType is the graph node type of the parent spec (e.g. "Feature",
// "Bug", "Initiative") — the caller supplies it so this package stays
// independent of the spec type registry.
//
// Idempotent: re-running on unchanged tasks produces no new bitemporal
// rows. Status flips (todo → doing → done) trigger fresh rows because
// the content hash includes status.
func Write(parentType, parentSlug string, parentID int64, parsed []Task, repoKey string, store *graph.Store) (*WriteSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("tasks: Write requires non-nil Store")
	}
	if parentSlug == "" {
		return nil, fmt.Errorf("tasks: Write requires non-empty parentSlug")
	}
	summary := &WriteSummary{}
	source := map[string]any{"kind": "spec-tasks"}

	for _, t := range parsed {
		key := parentSlug + ":" + t.ID
		props := map[string]any{
			"task_id": t.ID,
			"text":    t.Text,
			"status":  t.Status,
			"parent":  parentSlug,
		}
		if t.Kind != "" {
			props["kind"] = t.Kind
		}
		if t.Assignee != "" {
			props["assignee"] = t.Assignee
		}
		if t.DiscoveredAgainst != "" {
			props["discovered_against"] = t.DiscoveredAgainst
		}
		if t.Started != "" {
			props["started"] = t.Started
		}
		if t.Done != "" {
			props["done"] = t.Done
		}

		// Detect "was there a flip" before the upsert so we can report
		// it accurately. Re-upserting the same hash is a no-op.
		var prevStatus string
		if existing, err := store.GetNode("Task", key); err == nil && existing != nil {
			if s, ok := existing.Props["status"].(string); ok {
				prevStatus = s
			}
		}

		taskID, err := store.UpsertNode(&graph.Node{
			Type:        "Task",
			Domain:      "engineering",
			Key:         key,
			Props:       props,
			Repo:        repoKey,
			ContentHash: hashTask(key, t.Text, t.Status, t.Kind, t.Assignee, t.DiscoveredAgainst, t.Started, t.Done),
			Source:      source,
		})
		if err != nil {
			return summary, fmt.Errorf("upsert Task %s: %w", key, err)
		}
		summary.Tasks++
		if prevStatus != "" && prevStatus != t.Status {
			summary.StatusFlips++
		} else if prevStatus == t.Status && prevStatus != "" {
			summary.NoOps++
		}

		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: taskID,
			ToID:   parentID,
			Type:   "belongs_to",
			Repo:   repoKey,
			Source: source,
		}); err != nil {
			return summary, fmt.Errorf("upsert belongs_to: %w", err)
		}
		summary.BelongsTo++

		if t.DiscoveredAgainst != "" {
			targetID, ok := resolveSpecNodeID(store, t.DiscoveredAgainst)
			if ok {
				if _, err := store.UpsertEdge(&graph.Edge{
					FromID: taskID,
					ToID:   targetID,
					Type:   "discovered_against",
					Props:  map[string]any{"target_slug": t.DiscoveredAgainst},
					Repo:   repoKey,
					Source: source,
				}); err != nil {
					return summary, fmt.Errorf("upsert discovered_against: %w", err)
				}
				summary.DiscoveredAgainst++
			}
			// Silently skip when the target slug doesn't exist yet — a
			// later scan will land the edge once the spec is created.
		}

		if t.Assignee != "" {
			personID, err := upsertPerson(store, t.Assignee, repoKey)
			if err != nil {
				return summary, fmt.Errorf("upsert Person %s: %w", t.Assignee, err)
			}
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: taskID,
				ToID:   personID,
				Type:   "assigned_to",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return summary, fmt.Errorf("upsert assigned_to: %w", err)
			}
			summary.AssignedTo++
		}
	}
	return summary, nil
}

// resolveSpecNodeID tries each work-shaped spec node type to find one
// keyed by slug. Tasks discovered_against another spec don't know the
// target's type, only its slug — mirrors how spec.WriteGraph resolves
// relation targets.
func resolveSpecNodeID(store *graph.Store, slug string) (int64, bool) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return 0, false
	}
	// Strip a cross-repo qualifier (e.g. "alias/slug") same way the
	// spec resolver does.
	if idx := strings.LastIndex(slug, "/"); idx > 0 {
		slug = slug[idx+1:]
	}
	for _, t := range []string{"Feature", "Bug", "Initiative", "Decision", "Convention"} {
		if id, err := store.GetNodeID(t, slug); err == nil {
			return id, true
		}
	}
	return 0, false
}

// upsertPerson resolves (or creates) a Person node for an assignee
// string. Keyed by the assignee literal (handle / email / display
// name). Stays lightweight — no schema beyond {name: <key>}.
func upsertPerson(store *graph.Store, assignee, repoKey string) (int64, error) {
	if id, err := store.GetNodeID("Person", assignee); err == nil {
		return id, nil
	}
	return store.UpsertNode(&graph.Node{
		// Person is in globalNodeTypes — Domain stays empty.
		Type:        "Person",
		Key:         assignee,
		Props:       map[string]any{"name": assignee},
		Repo:        repoKey,
		ContentHash: "person-" + assignee,
		Source:      map[string]any{"kind": "task-assignee"},
	})
}

// hashTask hashes a Task's stable identity + status + metadata so
// idempotent re-ingest is a no-op when nothing changed. Any of these
// fields flipping produces a fresh bitemporal row — same contract AC
// follows via hashCriterion.
func hashTask(key, text, status, kind, assignee, discoveredAgainst, started, done string) string {
	parts := []string{key, text, status, kind, assignee, discoveredAgainst, started, done}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

// nowRFC returns wall-clock time formatted RFC3339-UTC. Single helper
// so test fakes can shadow time if needed (none today; kept simple).
func nowRFC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
