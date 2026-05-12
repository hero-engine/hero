package nextdoc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	s, err := graph.Open(filepath.Join(t.TempDir(), "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func writeNext(t *testing.T, heroDir, body string) {
	t.Helper()
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleNext = `---
updated: 2026-04-26T06:55:00Z
session: opencode-go/kimi-k2.6-2026-04-26
branch: main
---

## Just finished

- Some commit

## Tried and failed

- bcrypt rounds=12 too slow on the worker pool
- shrinking the snapshot interval — doubled write amplification

## Context to carry forward

- something
`

func TestParseNextExtractsSessionAndAttempts(t *testing.T) {
	p, err := parseNext(sampleNext)
	if err != nil {
		t.Fatalf("parseNext: %v", err)
	}
	if p.session != "opencode-go/kimi-k2.6-2026-04-26" {
		t.Errorf("session = %q", p.session)
	}
	if p.branch != "main" {
		t.Errorf("branch = %q", p.branch)
	}
	if len(p.attempts) != 2 {
		t.Errorf("attempts = %d, want 2", len(p.attempts))
	}
}

func TestParseNextNothingMessageSkipped(t *testing.T) {
	in := `---
session: x
---

## Tried and failed

Nothing this session.
`
	p, _ := parseNext(in)
	if len(p.attempts) != 0 {
		t.Errorf("expected 0 attempts, got %d", len(p.attempts))
	}
}

func TestWriteGraphCreatesAttemptNodesAndEdges(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), "hero")
	writeNext(t, heroDir, sampleNext)

	store := openTestStore(t)
	summary, err := WriteGraph(heroDir, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Sessions != 1 {
		t.Errorf("Sessions = %d, want 1", summary.Sessions)
	}
	if summary.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2", summary.Attempts)
	}
	if summary.Edges != 2 {
		t.Errorf("Edges = %d, want 2", summary.Edges)
	}

	stats, _ := store.Stats()
	if stats.NodesByType["Attempt"] != 2 {
		t.Errorf("Attempt nodes = %d, want 2", stats.NodesByType["Attempt"])
	}
	if stats.EdgesByType["attempted_in"] != 2 {
		t.Errorf("attempted_in edges = %d, want 2", stats.EdgesByType["attempted_in"])
	}
}

func TestWriteGraphIdempotent(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), "hero")
	writeNext(t, heroDir, sampleNext)
	store := openTestStore(t)
	if _, err := WriteGraph(heroDir, "test-repo", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteGraph(heroDir, "test-repo", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes {
		t.Errorf("history grew: %d → %d", before.HistoryRows.Nodes, after.HistoryRows.Nodes)
	}
}

func TestWriteGraphMissingNextIsNoop(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), "hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := openTestStore(t)
	summary, err := WriteGraph(heroDir, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", summary.Attempts)
	}
}
