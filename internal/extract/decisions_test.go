package extract

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/runner"
)

// fakeLLM implements the LLM interface (== a runner.LLMProvider)
// without making any network calls. It tracks call count and
// returns the configured reply text. Used in extraction tests
// to verify everything except real model output.
type fakeLLM struct {
	name    string
	reply   string
	calls   int
	lastReq runner.ChatRequest
}

func (f *fakeLLM) Chat(ctx context.Context, req runner.ChatRequest) (*runner.ChatResponse, error) {
	f.calls++
	f.lastReq = req
	return &runner.ChatResponse{
		Text:                     f.reply,
		InputTokens:              1234,
		OutputTokens:             200,
		CacheReadInputTokens:     1100, // verifies cache stats round-trip
		CacheCreationInputTokens: 0,
	}, nil
}

func (f *fakeLLM) Name() string {
	if f.name == "" {
		return "fake"
	}
	return f.name
}

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	s, err := graph.Open(filepath.Join(t.TempDir(), "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

const sampleDecisionsJSON = `[
  {
    "title": "Use SQLite for the local graph",
    "rationale": "Embedded, zero-install, plays well with Go.",
    "alternatives": ["Neo4j", "DuckDB"],
    "concepts": ["sqlite", "graph", "embedded-db"],
    "verdict": "accepted"
  },
  {
    "title": "Defer fine-tuning",
    "rationale": "Cost prohibitive, retraining cadence wrong.",
    "alternatives": ["Fine-tune small model", "Bigger context"],
    "concepts": ["llm", "fine-tuning", "rag"],
    "verdict": "rejected"
  }
]`

func TestExtractFromSource_HappyPath(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Note", Key: "buddy-model", Repo: "test", ContentHash: "h",
		Domain:      "engineering",
	}); err != nil {
		t.Fatal(err)
	}

	llm := &fakeLLM{name: "anthropic", reply: sampleDecisionsJSON}
	x := &DecisionExtractor{Client: NewClient(llm, "test-model")}

	summary, err := x.ExtractFromSource(
		context.Background(), store,
		"Note", "buddy-model",
		"... rich prose about model architecture choices ...",
		"test",
	)
	if err != nil {
		t.Fatalf("ExtractFromSource: %v", err)
	}

	if summary.Decisions != 2 {
		t.Errorf("Decisions = %d, want 2", summary.Decisions)
	}
	if summary.Concepts != 6 {
		t.Errorf("Concepts = %d, want 6 (3 per decision)", summary.Concepts)
	}
	if summary.Edges != 8 {
		t.Errorf("Edges = %d, want 8 (2 originated_in + 6 mentions)", summary.Edges)
	}

	stats, _ := store.Stats()
	if stats.NodesByType["Decision"] != 2 {
		t.Errorf("Decision nodes = %d, want 2", stats.NodesByType["Decision"])
	}
	if stats.NodesByType["Concept"] != 6 {
		t.Errorf("Concept nodes = %d, want 6", stats.NodesByType["Concept"])
	}
}

func TestExtractFromSource_AsksForCachedSystem(t *testing.T) {
	store := openTestStore(t)
	store.UpsertNode(&graph.Node{Type: "Note", Domain: "engineering", Key: "x", ContentHash: "h"})

	llm := &fakeLLM{reply: sampleDecisionsJSON}
	x := &DecisionExtractor{Client: NewClient(llm, "")}
	_, err := x.ExtractFromSource(context.Background(), store, "Note", "x", "prose", "test")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !llm.lastReq.CacheSystem {
		t.Error("extract should set CacheSystem=true so the provider caches when supported")
	}
}

func TestExtractFromSource_SkipsOnUnchangedHash(t *testing.T) {
	store := openTestStore(t)
	store.UpsertNode(&graph.Node{Type: "Note", Domain: "engineering", Key: "x", ContentHash: "h"})

	llm := &fakeLLM{reply: sampleDecisionsJSON}
	x := &DecisionExtractor{Client: NewClient(llm, "")}

	prose := "same prose"
	if _, err := x.ExtractFromSource(context.Background(), store, "Note", "x", prose, "test"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if llm.calls != 1 {
		t.Fatalf("first should call LLM once, got %d", llm.calls)
	}

	summary, err := x.ExtractFromSource(context.Background(), store, "Note", "x", prose, "test")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !summary.Skipped {
		t.Errorf("second call should skip on unchanged content")
	}
	if llm.calls != 1 {
		t.Errorf("second call should NOT hit LLM (cache by content_hash); got %d", llm.calls)
	}
}

func TestExtractFromSource_RerunsOnChangedContent(t *testing.T) {
	store := openTestStore(t)
	store.UpsertNode(&graph.Node{Type: "Note", Domain: "engineering", Key: "x", ContentHash: "h"})

	llm := &fakeLLM{reply: sampleDecisionsJSON}
	x := &DecisionExtractor{Client: NewClient(llm, "")}

	if _, err := x.ExtractFromSource(context.Background(), store, "Note", "x", "first prose", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := x.ExtractFromSource(context.Background(), store, "Note", "x", "DIFFERENT prose", "test"); err != nil {
		t.Fatal(err)
	}
	if llm.calls != 2 {
		t.Errorf("changed content should re-hit LLM, calls = %d (want 2)", llm.calls)
	}
}

func TestExtractFromSource_NoProviderFailsOpen(t *testing.T) {
	store := openTestStore(t)
	store.UpsertNode(&graph.Node{Type: "Note", Domain: "engineering", Key: "x", ContentHash: "h"})

	c := &Client{provider: nil} // simulate no key
	x := &DecisionExtractor{Client: c}
	_, err := x.ExtractFromSource(context.Background(), store, "Note", "x", "prose", "test")
	if err != ErrNoAPIKey {
		t.Errorf("got %v, want ErrNoAPIKey", err)
	}
}

func TestParseDecisionsJSON_ToleratesCodeFences(t *testing.T) {
	wrapped := "```json\n" + sampleDecisionsJSON + "\n```"
	out, err := parseDecisionsJSON(wrapped)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("got %d decisions, want 2", len(out))
	}
}

func TestParseDecisionsJSON_TrimsPreamble(t *testing.T) {
	preamble := "Sure, here are the decisions: " + sampleDecisionsJSON
	out, err := parseDecisionsJSON(preamble)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("got %d decisions, want 2", len(out))
	}
}

func TestNormalizeConcept(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SQLite", "sqlite"},
		{"Embedded DB", "embedded-db"},
		{"  graph   memory  ", "graph-memory"},
		{"prompt_caching", "prompt-caching"},
	}
	for _, c := range cases {
		if got := normalizeConcept(c.in); got != c.want {
			t.Errorf("normalizeConcept(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDefaultModelFor(t *testing.T) {
	cases := []struct{ provider, want string }{
		{"anthropic", "claude-haiku-4-5-20251001"},
		{"openai", "gpt-4o-mini"},
		{"azure", "gpt-4o-mini"},
		{"unknown", ""},
	}
	for _, c := range cases {
		if got := defaultModelFor(c.provider); got != c.want {
			t.Errorf("defaultModelFor(%q) = %q, want %q", c.provider, got, c.want)
		}
	}
}
