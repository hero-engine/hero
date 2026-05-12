package knowledge

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

func writeRaw(t *testing.T, heroDir, slug, frontmatter, body string) {
	t.Helper()
	rawDir := filepath.Join(heroDir, "knowledge", "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := frontmatter + body
	if err := os.WriteFile(filepath.Join(rawDir, slug+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRawGraphMissingDirIsNoop(t *testing.T) {
	store := openTestStore(t)
	heroDir := filepath.Join(t.TempDir(), "hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary, err := WriteRawGraph(heroDir, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteRawGraph: %v", err)
	}
	if summary.Documents != 0 {
		t.Errorf("Documents = %d, want 0", summary.Documents)
	}
}

func TestWriteRawGraphCreatesDocumentNodes(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), "hero")
	writeRaw(t, heroDir, "vendor-doc",
		"---\nsource: https://example.com/doc\ningested: 2026-04-26T00:00:00Z\ntitle: Vendor Doc\ntype: context\n---\n\n",
		"This is the body.\n",
	)
	writeRaw(t, heroDir, "another", "---\ntitle: Untitled\n---\n\n", "x")

	store := openTestStore(t)
	summary, err := WriteRawGraph(heroDir, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteRawGraph: %v", err)
	}
	if summary.Documents != 2 {
		t.Errorf("Documents = %d, want 2", summary.Documents)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Document"] != 2 {
		t.Errorf("Document nodes = %d, want 2", stats.NodesByType["Document"])
	}
}

func TestWriteRawGraphIdempotentOnUnchangedBytes(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), "hero")
	writeRaw(t, heroDir, "x", "---\ntitle: x\n---\n\n", "body")

	store := openTestStore(t)
	if _, err := WriteRawGraph(heroDir, "test-repo", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteRawGraph(heroDir, "test-repo", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes {
		t.Errorf("history grew on idempotent re-ingest: %d → %d",
			before.HistoryRows.Nodes, after.HistoryRows.Nodes)
	}
}

func TestParseRawFrontmatter(t *testing.T) {
	in := []byte("---\nsource: https://x\ningested: t\ntitle: T\ntype: decision\n---\n\nbody")
	fm := parseRawFrontmatter(in)
	if fm.source != "https://x" || fm.title != "T" || fm.docType != "decision" {
		t.Errorf("parsed wrong: %+v", fm)
	}
}

func TestParseRawFrontmatterMissingIsZero(t *testing.T) {
	fm := parseRawFrontmatter([]byte("no frontmatter here"))
	if fm.title != "" || fm.source != "" {
		t.Errorf("expected zero, got %+v", fm)
	}
}
