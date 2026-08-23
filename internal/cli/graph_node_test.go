package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
	"github.com/hero-engine/hero/internal/graph"
)

func TestGraphNodeAddHandlerOwnerStampsDurableProvenance(t *testing.T) {
	env := newTestEnv(t)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainEngineering, []domains.DomainID{domains.DomainQA}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd("graph", "node", "add",
		"--type", "TestPlan", "--key", "checkout-qa", "--handler-owner", "qa", "--json")
	if err != nil {
		t.Fatalf("handler-owned write: %v", err)
	}
	var written nodeAddResult
	if err := json.Unmarshal([]byte(out), &written); err != nil {
		t.Fatal(err)
	}
	if written.Domain != "qa" {
		t.Fatalf("written domain = %q, want qa", written.Domain)
	}

	// Primary/focus are retrieval context, not write ownership. Rewriting the
	// same artifact through the same selected handler remains QA-owned.
	cfg, err = config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainQA, nil); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := runCmd("graph", "node", "add",
		"--type", "TestPlan", "--key", "checkout-qa", "--handler-owner", "qa", "--json"); err != nil {
		t.Fatalf("same-owner mutation after primary change: %v", err)
	}
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	node, err := store.GetNode("TestPlan", "checkout-qa", "")
	if err != nil {
		t.Fatal(err)
	}
	if node.Domain != "qa" {
		t.Fatalf("durable node domain = %q, want qa", node.Domain)
	}

	if _, err := runCmd("graph", "node", "add",
		"--type", "TestPlan", "--key", "disabled-pm", "--handler-owner", "pm", "--json"); err == nil || !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("disabled handler owner error = %v", err)
	}
}

func TestGraphNodeAddRejectsDomainAndHandlerOwner(t *testing.T) {
	_ = newTestEnv(t)
	_, err := runCmd("graph", "node", "add", "--type", "Feature", "--key", "x", "--domain", "engineering", "--handler-owner", "engineering")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error = %v", err)
	}
}

func TestGraphNodeAddHappyPathStoryPM(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "reduce-checkout-abandonment",
		"--title", "Reduce checkout abandonment",
		"--json",
	)
	if err != nil {
		t.Fatalf("graph node add: %v", err)
	}

	var got nodeAddResult
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decoding json output %q: %v", output, err)
	}
	if got.NodeID == 0 {
		t.Errorf("node_id should be non-zero, got %+v", got)
	}
	if got.Type != "Story" {
		t.Errorf("type = %q, want %q", got.Type, "Story")
	}
	if got.Key != "reduce-checkout-abandonment" {
		t.Errorf("key = %q, want %q", got.Key, "reduce-checkout-abandonment")
	}
	if got.Domain != "pm" {
		t.Errorf("domain should default to pm for Story, got %q", got.Domain)
	}
	if got.Title != "Reduce checkout abandonment" {
		t.Errorf("title = %q, want %q", got.Title, "Reduce checkout abandonment")
	}
	if got.CreatedAt == "" {
		t.Errorf("created_at should be populated")
	}
}

func TestGraphNodeAddFeatureEngineering(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("graph", "node", "add",
		"--type", "Feature",
		"--key", "checkout-bandwidth",
		"--json",
	)
	if err != nil {
		t.Fatalf("graph node add: %v", err)
	}
	var got nodeAddResult
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Domain != "engineering" {
		t.Errorf("domain should default to engineering for Feature, got %q", got.Domain)
	}
	if got.Title != "checkout-bandwidth" {
		t.Errorf("title should fall back to key when --title omitted, got %q", got.Title)
	}
}

func TestGraphNodeAddMissingType(t *testing.T) {
	_ = newTestEnv(t)
	_, err := runCmd("graph", "node", "add", "--key", "x")
	if err == nil {
		t.Fatal("expected error when --type missing")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("error should mention --type: %v", err)
	}
}

func TestGraphNodeAddMissingKey(t *testing.T) {
	_ = newTestEnv(t)
	_, err := runCmd("graph", "node", "add", "--type", "Story")
	if err == nil {
		t.Fatal("expected error when --key missing")
	}
	if !strings.Contains(err.Error(), "--key is required") {
		t.Errorf("error should mention --key: %v", err)
	}
}

func TestGraphNodeAddUnknownTypeWithoutDomain(t *testing.T) {
	_ = newTestEnv(t)
	_, err := runCmd("graph", "node", "add",
		"--type", "Widget",
		"--key", "foo",
	)
	if err == nil {
		t.Fatal("expected error for unknown type without --domain")
	}
	if !strings.Contains(err.Error(), "--domain") {
		t.Errorf("error should mention --domain: %v", err)
	}
	if !strings.Contains(err.Error(), "Widget") {
		t.Errorf("error should name the unmapped type: %v", err)
	}
}

func TestGraphNodeAddUnknownTypeWithExplicitDomain(t *testing.T) {
	_ = newTestEnv(t)
	// Unknown type is fine when caller supplies --domain explicitly —
	// the CLI is forward-compatible with whatever the graph allows.
	output, err := runCmd("graph", "node", "add",
		"--type", "Widget",
		"--key", "foo",
		"--domain", "engineering",
		"--json",
	)
	if err != nil {
		t.Fatalf("explicit --domain should let unknown types through: %v", err)
	}
	var got nodeAddResult
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Domain != "engineering" {
		t.Errorf("domain = %q, want %q", got.Domain, "engineering")
	}
}

func TestGraphNodeAddIdempotent(t *testing.T) {
	_ = newTestEnv(t)

	out1, err := runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "same-story",
		"--json",
	)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	out2, err := runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "same-story",
		"--json",
	)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}

	var a, b nodeAddResult
	if err := json.Unmarshal([]byte(out1), &a); err != nil {
		t.Fatalf("decode 1: %v", err)
	}
	if err := json.Unmarshal([]byte(out2), &b); err != nil {
		t.Fatalf("decode 2: %v", err)
	}
	if a.NodeID == 0 || a.NodeID != b.NodeID {
		t.Errorf("idempotent run should return same node_id; got %d then %d", a.NodeID, b.NodeID)
	}
}

// TestGraphNodeAddCrossDomainMutationRejected guards graph.UpsertNode's
// "first write wins" invariant from the CLI surface: once Story:foo is
// stamped pm, a follow-up upsert with --domain engineering must
// surface graph.ErrDomainMutation.
func TestGraphNodeAddCrossDomainMutationRejected(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "domain-conflict",
		"--domain", "pm",
		"--json",
	)
	if err != nil {
		t.Fatalf("first write (pm): %v", err)
	}
	_, err = runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "domain-conflict",
		"--domain", "engineering",
		"--json",
	)
	if err == nil {
		t.Fatal("expected ErrDomainMutation on second write with different domain")
	}
	if !strings.Contains(err.Error(), "cannot change a node's domain") {
		t.Errorf("error should explain domain mutation: %v", err)
	}
}

// TestGraphNodeAddTextOutput sanity-checks the human-readable mode —
// useful when developers run the command interactively without --json.
func TestGraphNodeAddTextOutput(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("graph", "node", "add",
		"--type", "Story",
		"--key", "human-mode",
		"--title", "Human mode test",
	)
	if err != nil {
		t.Fatalf("graph node add: %v", err)
	}
	for _, want := range []string{"node ", "Story:human-mode", "domain:", "pm", "Human mode test"} {
		if !strings.Contains(output, want) {
			t.Errorf("text output missing %q: %s", want, output)
		}
	}
}
