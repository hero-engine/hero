package extract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// decisionsSystemPrompt is sent with cache_control on every call.
// Keep it stable across calls so the cache hit rate stays high.
const decisionsSystemPrompt = `You read engineering notes and specs and extract structured decisions.

A "decision" is any concrete choice the author made or proposes:
choosing technology X over Y, locking a design pattern, accepting a
trade-off, agreeing on a process, or rejecting an approach.

Output ONLY a JSON array. Each element has:
  - "title":       short noun phrase (≤ 10 words) naming the decision
  - "rationale":   1-2 sentences on why
  - "alternatives": array of strings — other options considered (may be empty)
  - "concepts":    array of normalized lowercase keywords (≤ 8)
  - "verdict":     one of: "accepted" | "proposed" | "rejected" | "deferred"

If no decisions are present, output [].

Do NOT include any text outside the JSON array. Do NOT wrap in
code fences. Do NOT add explanations.

Example output:
[
  {
    "title": "Use SQLite for the local graph",
    "rationale": "Embedded, zero-install, plays nicely with Go.",
    "alternatives": ["Neo4j", "DuckDB"],
    "concepts": ["sqlite", "graph", "embedded-db", "local-storage"],
    "verdict": "accepted"
  }
]`

// ExtractedDecision is what the LLM produces. Decoded from JSON output.
type ExtractedDecision struct {
	Title        string   `json:"title"`
	Rationale    string   `json:"rationale"`
	Alternatives []string `json:"alternatives"`
	Concepts     []string `json:"concepts"`
	Verdict      string   `json:"verdict"`
}

// DecisionExtractor pulls Decision nodes (and Concept nodes + edges)
// out of prose attached to a source node — typically a Note, Spec,
// or Document.
type DecisionExtractor struct {
	Client *Client
}

// ExtractFromSource reads the prose of a source graph node, calls
// the LLM, and writes the extracted Decision/Concept nodes plus
// edges back into the graph.
//
// Idempotency: the source's content_hash is used as the cache key.
// If a prior extraction wrote nodes whose source.extracted_from_hash
// equals the current source's content_hash, the call is skipped.
//
// Returns the count of new Decision/Concept/edge rows written.
func (x *DecisionExtractor) ExtractFromSource(
	ctx context.Context,
	store *graph.Store,
	sourceType, sourceKey, prose, repoKey string,
) (*ExtractSummary, error) {
	if !x.Client.HasKey() {
		return nil, ErrNoAPIKey
	}
	if strings.TrimSpace(prose) == "" {
		return &ExtractSummary{}, nil
	}

	// Resolve the source node so we can attach edges.
	sourceID, err := store.GetNodeID(sourceType, sourceKey, repoKey)
	if err != nil {
		return nil, fmt.Errorf("source %s/%s not in graph: %w", sourceType, sourceKey, err)
	}

	hash := contentHash(prose)
	if alreadyExtracted(store, sourceType, sourceKey, hash) {
		return &ExtractSummary{Skipped: true}, nil
	}

	resp, err := x.Client.Run(ctx, Request{
		System: decisionsSystemPrompt,
		User:   fmt.Sprintf("Source: %s `%s`\n\n%s", sourceType, sourceKey, prose),
		MaxOut: 1500,
	})
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	decisions, err := parseDecisionsJSON(resp.Text)
	if err != nil {
		return nil, fmt.Errorf("parse llm output: %w (raw: %q)", err, truncatePreview(resp.Text, 240))
	}

	summary := &ExtractSummary{
		InputTokens:   resp.InputTokens,
		OutputTokens:  resp.OutputTokens,
		CacheReads:    resp.CacheReads,
		CacheCreates:  resp.CacheCreates,
	}

	source := map[string]any{
		"kind":                 "tier2-extraction",
		"extracted_from_type":  sourceType,
		"extracted_from_key":   sourceKey,
		"extracted_from_hash":  hash,
	}

	for i, d := range decisions {
		// Decision key is stable: source-key + index. Re-running on the
		// same source content produces the same keys, so re-extraction
		// is idempotent at the node level too.
		decisionKey := fmt.Sprintf("%s:decision-%d", sourceKey, i+1)

		props := map[string]any{
			"title":     d.Title,
			"rationale": d.Rationale,
			"verdict":   d.Verdict,
		}
		if len(d.Alternatives) > 0 {
			props["alternatives"] = d.Alternatives
		}

		decisionID, err := store.UpsertNode(&graph.Node{
			Type:        "Decision",
			Domain:      "engineering",
			Key:         decisionKey,
			Props:       props,
			Repo:        repoKey,
			ContentHash: contentHash(d.Title + d.Rationale + d.Verdict),
			Source:      source,
		})
		if err != nil {
			return summary, fmt.Errorf("upsert Decision: %w", err)
		}
		summary.Decisions++

		// Decision originated_in Source.
		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: decisionID, ToID: sourceID, Type: "originated_in",
			Repo:   repoKey,
			Source: source,
		}); err != nil {
			return summary, fmt.Errorf("upsert originated_in: %w", err)
		}
		summary.Edges++

		// Concepts mentioned in this decision.
		for _, term := range d.Concepts {
			term = normalizeConcept(term)
			if term == "" {
				continue
			}
			conceptID, err := store.UpsertNode(&graph.Node{
				Type:        "Concept",
				Domain:      "engineering",
				Key:         term,
				Props:       map[string]any{"term": term},
				ContentHash: contentHash("concept:" + term),
				Source:      map[string]any{"kind": "tier2-extraction"},
			})
			if err != nil {
				return summary, fmt.Errorf("upsert Concept: %w", err)
			}
			summary.Concepts++

			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: decisionID, ToID: conceptID, Type: "mentions",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return summary, fmt.Errorf("upsert mentions: %w", err)
			}
			summary.Edges++
		}
	}

	return summary, nil
}

// ExtractSummary reports counts + token usage for diagnostics.
type ExtractSummary struct {
	Skipped       bool
	Decisions     int
	Concepts      int
	Edges         int
	InputTokens   int
	OutputTokens  int
	CacheReads    int
	CacheCreates  int
}

// alreadyExtracted checks whether a prior extraction with the same
// content_hash produced any Decision rows for this source.
func alreadyExtracted(store *graph.Store, sourceType, sourceKey, hash string) bool {
	var n int
	err := store.DB().QueryRow(
		`SELECT COUNT(*) FROM nodes
		  WHERE type = 'Decision' AND valid_to IS NULL
		    AND json_extract(source, '$.extracted_from_type') = ?
		    AND json_extract(source, '$.extracted_from_key')  = ?
		    AND json_extract(source, '$.extracted_from_hash') = ?`,
		sourceType, sourceKey, hash,
	).Scan(&n)
	if err != nil {
		return false
	}
	return n > 0
}

// parseDecisionsJSON tolerates a few common LLM quirks: leading/
// trailing whitespace, accidental code fences, conversational preamble.
func parseDecisionsJSON(raw string) ([]ExtractedDecision, error) {
	s := strings.TrimSpace(raw)
	// Strip code fences if the model added them.
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	// Find the first '[' to skip preamble.
	if i := strings.Index(s, "["); i > 0 {
		s = s[i:]
	}

	var out []ExtractedDecision
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeConcept(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	// Collapse whitespace; replace word separators with hyphens.
	s = strings.ReplaceAll(s, "_", "-")
	fields := strings.Fields(s)
	return strings.Join(fields, "-")
}

func contentHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func truncatePreview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
