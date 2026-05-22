package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

// seedEdgeNodes writes two nodes directly into the test workspace's
// graph.db and returns their IDs. Bypasses the spec-ingest path so
// these tests focus on the CLI surface, not on the graph populating
// pipeline.
func seedEdgeNodes(t *testing.T, env *testEnv) {
	t.Helper()

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer store.Close()

	nodes := []graph.Node{
		{Type: "story", Key: "reduce-churn", Domain: "pm", ContentHash: "h-story"},
		{Type: "feature", Key: "checkout-bandwidth", Domain: "engineering", ContentHash: "h-feat"},
		{Type: "feature", Key: "same-domain-target", Domain: "pm", ContentHash: "h-same"},
	}
	for i := range nodes {
		if _, err := store.UpsertNode(&nodes[i]); err != nil {
			t.Fatalf("UpsertNode %s:%s: %v", nodes[i].Type, nodes[i].Key, err)
		}
	}
}

func TestGraphEdgeAddHappyPath(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	output, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
		"--from-domain", "pm",
		"--to-domain", "engineering",
		"--json",
	)
	if err != nil {
		t.Fatalf("graph edge add: %v", err)
	}

	var got edgeAddResult
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decoding json output %q: %v", output, err)
	}
	if got.EdgeID == 0 {
		t.Errorf("edge_id should be non-zero, got %+v", got)
	}
	if got.From != "story:reduce-churn" {
		t.Errorf("from = %q, want %q", got.From, "story:reduce-churn")
	}
	if got.To != "feature:checkout-bandwidth" {
		t.Errorf("to = %q, want %q", got.To, "feature:checkout-bandwidth")
	}
	if got.Kind != "handoff" {
		t.Errorf("kind = %q, want %q", got.Kind, "handoff")
	}
	if got.FromDomain != "pm" {
		t.Errorf("from_domain = %q, want %q", got.FromDomain, "pm")
	}
	if got.ToDomain != "engineering" {
		t.Errorf("to_domain = %q, want %q", got.ToDomain, "engineering")
	}
	if got.CreatedAt == "" {
		t.Errorf("created_at should be populated")
	}
}

func TestGraphEdgeAddMissingFromNode(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	_, err := runCmd("graph", "edge", "add",
		"--from", "story:does-not-exist",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
	)
	if err == nil {
		t.Fatal("expected error for missing from-node")
	}
	if !strings.Contains(err.Error(), "from-node not found") {
		t.Errorf("error should name the missing from-node: %v", err)
	}
	if !strings.Contains(err.Error(), "story:does-not-exist") {
		t.Errorf("error should include the missing identifier: %v", err)
	}
}

func TestGraphEdgeAddMissingToNode(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	_, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:nope",
		"--kind", "handoff",
	)
	if err == nil {
		t.Fatal("expected error for missing to-node")
	}
	if !strings.Contains(err.Error(), "to-node not found") {
		t.Errorf("error should name the missing to-node: %v", err)
	}
}

// TestGraphEdgeAddCrossDomainAllowedKind verifies the primary use case:
// pm story → engineering feature, kind=handoff. handoff is in the v1
// cross-domain allow-list, so this must succeed cleanly.
func TestGraphEdgeAddCrossDomainAllowedKind(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	output, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
	)
	if err != nil {
		t.Fatalf("cross-domain allowed-kind should succeed: %v", err)
	}
	if !strings.Contains(output, "handoff") {
		t.Errorf("output should mention edge kind: %q", output)
	}
}

// TestGraphEdgeAddCrossDomainDisallowedKindStillWrites mirrors
// internal/graph's existing "permissive on intent" semantics: a
// cross-domain edge with a kind NOT in the v1 allow-list still
// writes, and is surfaced later by `hero warnings`. This test
// guards against regressing to a hard reject.
func TestGraphEdgeAddCrossDomainDisallowedKindStillWrites(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	_, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:checkout-bandwidth",
		"--kind", "mentions",
		"--json",
	)
	if err != nil {
		t.Fatalf("cross-domain non-allow-listed kind should still write: %v", err)
	}
}

func TestGraphEdgeAddInvalidFromFormat(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	_, err := runCmd("graph", "edge", "add",
		"--from", "no-colon-here",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
	)
	if err == nil {
		t.Fatal("expected parse error for bad --from")
	}
	if !strings.Contains(err.Error(), "<type>:<key>") {
		t.Errorf("error should explain the format: %v", err)
	}
}

func TestGraphEdgeAddIdempotent(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	out1, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
		"--from-domain", "pm",
		"--json",
	)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	out2, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:checkout-bandwidth",
		"--kind", "handoff",
		"--from-domain", "pm",
		"--json",
	)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}

	var a, b edgeAddResult
	if err := json.Unmarshal([]byte(out1), &a); err != nil {
		t.Fatalf("decode 1: %v", err)
	}
	if err := json.Unmarshal([]byte(out2), &b); err != nil {
		t.Fatalf("decode 2: %v", err)
	}
	if a.EdgeID == 0 || a.EdgeID != b.EdgeID {
		t.Errorf("idempotent run should return same edge_id; got %d then %d", a.EdgeID, b.EdgeID)
	}
}

func TestGraphEdgeAddMissingRequiredFlags(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("graph", "edge", "add", "--from", "story:x")
	if err == nil {
		t.Fatal("expected error when --to and --kind missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("error should mention required flags: %v", err)
	}
}

func TestGraphEdgeAddSameDomain(t *testing.T) {
	env := newTestEnv(t)
	seedEdgeNodes(t, env)

	output, err := runCmd("graph", "edge", "add",
		"--from", "story:reduce-churn",
		"--to", "feature:same-domain-target",
		"--kind", "depends_on",
		"--json",
	)
	if err != nil {
		t.Fatalf("same-domain edge: %v", err)
	}
	var got edgeAddResult
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.FromDomain != "pm" || got.ToDomain != "pm" {
		t.Errorf("same-domain edge should have pm/pm domains, got from=%q to=%q",
			got.FromDomain, got.ToDomain)
	}
}
