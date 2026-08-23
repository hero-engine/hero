package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteGraph maps parsed specs into the unified knowledge graph.
//
// Each spec becomes a node whose Type is its spec Type capitalized
// (Feature, Initiative, Decision, ...), keyed by slug. Frontmatter
// `relations:` entries become typed edges between the spec nodes.
//
// repoKey stamps the partition column on every node and edge written
// (federation contract — see graph-memory-federation/spec.md).
//
// fallbackDomain is the workspace-level domain (typically from
// the resolved primary domain via graph.DomainFor) used when a spec's frontmatter
// does not declare its own `domain:` field. Per-spec frontmatter
// wins when set; empty falls back to fallbackDomain; empty
// fallbackDomain falls back to "engineering" to keep pre-migration
// workspaces a no-op (DSKG Phase 4 contract).
//
// Idempotent: re-running on unchanged specs produces no new history.
// Updates to a spec invalidate the prior row and reassert its edges.
func WriteGraph(specs []*Spec, repoKey, fallbackDomain string, store *graph.Store) (*GraphWriteSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("spec: WriteGraph requires non-nil Store")
	}
	if fallbackDomain == "" {
		fallbackDomain = "engineering"
	}

	source := map[string]any{"kind": "spec"}
	summary := &GraphWriteSummary{}

	// First pass: upsert all spec nodes so relations can resolve target IDs.
	idByTypeKey := map[string]int64{} // key: "<NodeType>:<slug>"
	for _, s := range specs {
		nodeType := graphTypeFor(s.Type)
		if nodeType == "" {
			continue // unsupported type — skip
		}
		props := specProps(s)
		hash := hashSpec(s)
		domain := s.Domain
		if domain == "" {
			domain = fallbackDomain
		}

		id, err := store.UpsertNode(&graph.Node{
			Type:        nodeType,
			Domain:      domain,
			Key:         s.Slug,
			Props:       props,
			Repo:        repoKey,
			ContentHash: hash,
			Source: map[string]any{
				"kind": "spec",
				"path": s.Path,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("upsert %s/%s: %w", nodeType, s.Slug, err)
		}
		idByTypeKey[nodeType+":"+s.Slug] = id
		summary.Specs++
	}

	// Second pass: acceptance criteria → Criterion nodes + belongs_to
	// edges from each parsed AC back to its parent spec node. Tier-1
	// deterministic: the parser only extracts what's structurally
	// present in the spec body. Status is left as "unknown" — Phase 2
	// of acceptance-criteria-graph wires run-result ingest to flip it.
	for _, sp := range specs {
		fromType := graphTypeFor(sp.Type)
		if fromType == "" {
			continue
		}
		fromID, ok := idByTypeKey[fromType+":"+sp.Slug]
		if !ok {
			continue
		}
		critDomain := sp.Domain
		if critDomain == "" {
			critDomain = fallbackDomain
		}
		for _, ac := range sp.ParseAcceptanceCriteria() {
			critKey := sp.Slug + ":" + ac.ID

			// Preserve any status that prior runs (e.g. `hero ac
			// record`) have flipped. Scan should never clobber a
			// recorded pass/fail back to "proposed". The "unknown"
			// value was a Phase-1 placeholder; treat it as default
			// so existing nodes graduate to "proposed" on next scan.
			status := "proposed"
			if existing, err := store.GetNode("Criterion", critKey, repoKey); err == nil && existing != nil {
				if s, ok := existing.Props["status"].(string); ok && isRecordedStatus(s) {
					status = s
				}
			}

			critID, err := store.UpsertNode(&graph.Node{
				Type:        "Criterion",
				Domain:      critDomain,
				Key:         critKey,
				Props: map[string]any{
					"ac_id":     ac.ID,
					"statement": ac.Statement,
					"status":    status,
					"parent":    sp.Slug,
				},
				Repo:        repoKey,
				ContentHash: hashCriterion(critKey, ac.Statement, status),
				Source: map[string]any{
					"kind": "spec-acceptance",
					"path": sp.Path,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Criterion %s: %w", critKey, err)
			}
			summary.Criteria++
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: critID, ToID: fromID, Type: "belongs_to",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert Criterion→%s edge: %w", sp.Slug, err)
			}
			summary.Edges++
		}
	}

	// Third pass: relations → edges. Resolve target by trying each spec
	// type until we find a match (relation targets carry slug, not type).
	for _, s := range specs {
		fromType := graphTypeFor(s.Type)
		if fromType == "" {
			continue
		}
		fromID, ok := idByTypeKey[fromType+":"+s.Slug]
		if !ok {
			continue
		}
		for _, rel := range s.Relations {
			edgeType := graphEdgeForRelation(rel.Kind)
			if edgeType == "" {
				continue
			}
			toID, ok := resolveTargetID(rel.Target, idByTypeKey, fromID)
			if !ok {
				continue // target not in this batch — skipped, will resolve on a later run
			}
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: fromID, ToID: toID, Type: edgeType,
				Props: map[string]any{
					"declared_kind": rel.Kind,
				},
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert edge %s→%s: %w", s.Slug, rel.Target, err)
			}
			summary.Edges++
		}
	}

	// Fourth pass: typed external sources owned by specs. Mail bodies remain
	// private; only stable routing/provenance identity enters the graph.
	for _, s := range specs {
		if s.Type != TypeIntake || s.Source == nil || s.Source.Kind != "mail" || s.Source.ID == "" {
			continue
		}
		fromID, ok := idByTypeKey["Intake:"+s.Slug]
		if !ok {
			continue
		}
		domain := s.Domain
		if domain == "" {
			domain = fallbackDomain
		}
		sourceID, err := store.UpsertNode(&graph.Node{
			Type: "MailSource", Key: s.Source.ID, Domain: domain, Repo: repoKey,
			Props: map[string]any{
				"message_id":        s.Source.ID,
				"thread_id":         s.Source.ThreadID,
				"sender_peer_id":    s.Source.SenderPeerID,
				"recipient_peer_id": s.Source.RecipientPeerID,
			},
			Source: map[string]any{"kind": "mail", "path": s.Path},
		})
		if err != nil {
			return nil, fmt.Errorf("upsert MailSource/%s: %w", s.Source.ID, err)
		}
		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: fromID, ToID: sourceID, Type: "mail_source", Repo: repoKey,
			Source: map[string]any{"kind": "mail", "path": s.Path},
		}); err != nil {
			return nil, fmt.Errorf("upsert Intake→MailSource edge: %w", err)
		}
		summary.Edges++
	}

	return summary, nil
}

// GraphWriteSummary reports counts written by WriteGraph.
type GraphWriteSummary struct {
	Specs    int
	Criteria int
	Edges    int
}

// graphTypeFor maps a spec.Type to its node type name, or "" if the
// type is not represented in the graph yet.
func graphTypeFor(t Type) string {
	switch t {
	case TypeFeature:
		return "Feature"
	case TypeInitiative:
		return "Initiative"
	case TypeBug:
		return "Bug"
	case TypeConvention:
		return "Convention"
	case TypeDecision:
		return "Decision"
	case TypeRule:
		return "Rule"
	case TypeContext:
		return "ContextDoc"
	case TypeNote:
		return "Note"
	case TypeExternal:
		return "External"
	case TypeTripwire:
		return "Tripwire"
	case TypeExplainer:
		return "Explainer"
	case TypeIntake:
		return "Intake"
	default:
		return ""
	}
}

// graphEdgeForRelation maps a frontmatter relation kind to an edge
// type. Returns "" for relations we choose not to materialize as edges
// (e.g. child relations are the inverse of parent, recorded from the
// parent side).
func graphEdgeForRelation(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "parent":
		return "belongs_to"
	case "depends-on", "depends_on":
		return "depends_on"
	case "conflicts-with", "conflicts_with":
		// Soft mutex consumed by the /drive judge: two specs must not
		// deliver concurrently. Distinct from related_to so the gate can
		// find it.
		return "conflicts_with"
	case "blocks":
		return "blocks"
	case "supersedes":
		return "supersedes"
	case "related", "relates-to", "relates_to", "sibling":
		return "related_to"
	case "derived_from", "derived-from":
		// Promoted roadmap spec → originating intake. `hero why` walks
		// this in reverse to surface an intake's provenance.
		return "derived_from"
	case "promotes_to", "promotes-to":
		// Intake → promoted roadmap spec. This is the forward half of
		// the promotion provenance contract; derived_from is its reverse.
		return "promotes_to"
	case "child":
		// Inverse of parent — emitted from the child's side, so skip here
		// to avoid duplicate edges.
		return ""
	default:
		return ""
	}
}

// normalizeRelTarget converts a relation target to a slug.
// Relative paths like "../../initiatives/hero-cli/spec.md" become "hero-cli".
// Cross-repo qualifiers like "remote-org/feature" become "feature".
// Plain slugs pass through unchanged.
func normalizeRelTarget(target string) string {
	if !strings.Contains(target, "/") {
		return target
	}
	if strings.HasSuffix(target, ".md") {
		return filepath.Base(filepath.Dir(target))
	}
	// Cross-repo qualifier: keep the segment after the last slash.
	return target[strings.LastIndex(target, "/")+1:]
}

// resolveTargetID looks up a relation target slug across all spec
// types, since relations carry only the slug. fromID is the node that
// declared the relation; a match equal to fromID is skipped so a spec
// never edges to itself. This also disambiguates the promote case where
// an intake and its promoted spec share a slug: a derived_from relation
// declared by the promoted spec resolves to the (other-typed) intake
// node rather than self-looping back to the promoted spec.
func resolveTargetID(target string, idByTypeKey map[string]int64, fromID int64) (int64, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, false
	}
	target = normalizeRelTarget(target)
	for _, t := range []string{
		"Feature", "Initiative", "Bug", "Convention", "Decision",
		"Rule", "ContextDoc", "Note", "External", "Intake", "Criterion",
	} {
		if id, ok := idByTypeKey[t+":"+target]; ok && id != fromID {
			return id, true
		}
	}
	return 0, false
}

func specProps(s *Spec) map[string]any {
	props := map[string]any{
		"title":  s.Title,
		"status": string(s.Status),
	}
	if s.Priority != "" {
		props["priority"] = s.Priority
	}
	if s.Severity != "" {
		props["severity"] = s.Severity
	}
	if s.ClaimedBy != "" {
		props["claimed_by"] = s.ClaimedBy
	}
	if len(s.Tags) > 0 {
		props["tags"] = s.Tags
	}
	if s.TrackerID != "" {
		props["tracker_id"] = s.TrackerID
	}
	if s.TrackerStatus != "" {
		props["tracker_status"] = s.TrackerStatus
	}
	if s.URL != "" {
		props["url"] = s.URL
	}
	if !s.CreatedAt.IsZero() {
		props["created_at"] = s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if len(s.Triggers) > 0 {
		props["triggers"] = s.Triggers
	}
	if len(s.Scope) > 0 {
		props["scope"] = s.Scope
	}
	if s.Subproject != "" {
		props["subproject"] = s.Subproject
	}
	return props
}

// isRecordedStatus reports whether s is one of the run-result-driven
// statuses that scan must preserve. "proposed" and "unknown" are
// scan-default placeholders that should yield to a fresh default on
// re-ingest.
func isRecordedStatus(s string) bool {
	switch s {
	case "passing", "failing", "regressed", "retired":
		return true
	}
	return false
}

// hashCriterion hashes a Criterion's stable identity + statement +
// status so idempotent re-ingest is a no-op when nothing changed,
// and any of the three changing produces a fresh bitemporal row.
//
// Status is part of the hash so `hero ac record` flipping a Criterion
// from "proposed" → "passing" invalidates the prior row and inserts a
// new current one — that's the bitemporal contract per the spec.
func hashCriterion(key, statement, status string) string {
	sum := sha256.Sum256([]byte(key + "|" + statement + "|" + status))
	return hex.EncodeToString(sum[:])
}

// hashSpec hashes the structural fields and modtime of a spec so
// unchanged specs produce unchanged hashes. Body content is hashed
// indirectly via RawContent length + modtime; we keep this cheap on
// the assumption that mtime is a good change signal.
func hashSpec(s *Spec) string {
	type sigSpec struct {
		Slug, Title, Type, Status string
		Priority, ClaimedBy       string
		Tags, FilesTouched        []string
		Relations                 []Relation
		ModTime                   string
		RawLen                    int
	}
	sig := sigSpec{
		Slug: s.Slug, Title: s.Title,
		Type: string(s.Type), Status: string(s.Status),
		Priority: s.Priority, ClaimedBy: s.ClaimedBy,
		Tags: s.Tags, FilesTouched: s.FilesTouched,
		Relations: s.Relations,
		ModTime:   s.ModifiedAt.UTC().Format("2006-01-02T15:04:05Z"),
		RawLen:    len(s.RawContent),
	}
	b, _ := json.Marshal(sig)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
