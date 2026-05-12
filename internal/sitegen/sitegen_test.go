package sitegen

import (
	"os"
	"path/filepath"
	"strings"
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

func seed(t *testing.T, s *graph.Store) {
	t.Helper()
	// One open feature
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Feature", Key: "shipping-rewrite", Repo: "test",
		Props: map[string]any{
			"title": "Shipping rewrite", "status": "planning",
			"priority": "P0",
		},
		ContentHash: "h1",
	}); err != nil {
		t.Fatal(err)
	}
	// One initiative
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Initiative", Key: "platform", Repo: "test",
		Props:       map[string]any{"title": "Platform 2.0", "status": "active"},
		ContentHash: "h2",
	}); err != nil {
		t.Fatal(err)
	}
	// One decision
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Decision", Key: "use-sqlite", Repo: "test",
		Props:       map[string]any{"title": "Use SQLite for graph", "rationale": "Embedded, simple."},
		ContentHash: "h3",
	}); err != nil {
		t.Fatal(err)
	}
	// One note
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Note", Key: "buddy-model", Repo: "test",
		Props:       map[string]any{"title": "Buddy model", "body": "Some prose"},
		ContentHash: "h4",
	}); err != nil {
		t.Fatal(err)
	}
	// A commit
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Commit", Key: "abc1234567", Repo: "test",
		Props: map[string]any{
			"sha": "abc1234567", "subject": "feat: ship",
			"date": "2026-04-26T10:00:00Z", "author_name": "Alice",
		},
		ContentHash: "h5",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_WritesAllExpectedPages(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)
	out := t.TempDir()

	g := &Generator{Store: store, RepoKey: "test", OutDir: out}
	summary, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if summary.Index != 1 || summary.Features != 1 ||
		summary.Initiatives != 1 || summary.Decisions != 1 ||
		summary.Notes != 1 || summary.Activity != 1 {
		t.Errorf("summary = %+v", summary)
	}

	expectedFiles := []string{
		"index.html", "activity.html", "style.css",
		"features/shipping-rewrite.html",
		"initiatives/platform.html",
		"decisions/use-sqlite.html",
		"notes/buddy-model.html",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(out, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
}

func TestGenerate_IndexShowsCountsAndOpenWork(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)
	out := t.TempDir()
	g := &Generator{Store: store, RepoKey: "test", OutDir: out}
	if _, err := g.Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	html, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(html)
	for _, want := range []string{
		"Shipping rewrite", "Platform 2.0",
		"abc1234", "feat: ship",
		"In flight", "Initiatives", "Recent commits",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestGenerate_FeaturePageHasBodyAndStatus(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)
	out := t.TempDir()
	g := &Generator{Store: store, RepoKey: "test", OutDir: out}
	if _, err := g.Generate(); err != nil {
		t.Fatal(err)
	}

	html, err := os.ReadFile(filepath.Join(out, "features", "shipping-rewrite.html"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(html)
	for _, want := range []string{
		"Shipping rewrite", "shipping-rewrite",
		"P0", "planning",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("feature page missing %q", want)
		}
	}
}

func TestGenerate_DeterministicForSameGraph(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)
	out1, out2 := t.TempDir(), t.TempDir()

	g1 := &Generator{Store: store, RepoKey: "test", OutDir: out1}
	g2 := &Generator{Store: store, RepoKey: "test", OutDir: out2}
	if _, err := g1.Generate(); err != nil {
		t.Fatal(err)
	}
	if _, err := g2.Generate(); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"index.html", "features/shipping-rewrite.html"} {
		a, _ := os.ReadFile(filepath.Join(out1, name))
		b, _ := os.ReadFile(filepath.Join(out2, name))
		// Strip the "Generated" timestamp line — it's the only
		// non-deterministic bit by design (lets users see freshness).
		stripGen := func(s string) string {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if strings.Contains(l, "generated ") {
					lines[i] = "<GENERATED>"
				}
			}
			return strings.Join(lines, "\n")
		}
		if stripGen(string(a)) != stripGen(string(b)) {
			t.Errorf("%s differs across regenerations (modulo timestamp)", name)
		}
	}
}

func TestSlugForFile(t *testing.T) {
	cases := []struct{ in, want string }{
		{"simple", "simple"},
		{"with/slash", "with_slash"},
		{"with:colon", "with_colon"},
		{"with space", "with-space"},
	}
	for _, c := range cases {
		if got := slugForFile(c.in); got != c.want {
			t.Errorf("slugForFile(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
