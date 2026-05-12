package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const refreshSpec = `---
title: Refresh Demo
type: feature
status: planning
---
# Refresh Demo

## Goal
Body content for the FTS index.
`

func writeRefreshSpec(t *testing.T, heroDir, slug, content string) string {
	t.Helper()
	dir := filepath.Join(heroDir, "specs", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func newRefreshHeroDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "specs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return heroDir
}

func TestRefreshIfStale_NoOp_WhenIndexCurrent(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeRefreshSpec(t, heroDir, "alpha", refreshSpec)

	// First refresh indexes everything.
	first, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if first.Indexed != 1 || first.Updated != 0 || first.Removed != 0 {
		t.Errorf("first: want Indexed=1 Updated=0 Removed=0, got %+v", first)
	}

	// Second refresh, no disk changes — should be a no-op.
	second, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if !second.IsClean() {
		t.Errorf("second refresh should be clean, got %+v", second)
	}
}

func TestRefreshIfStale_IndexesNewSpec(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeRefreshSpec(t, heroDir, "alpha", refreshSpec)
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("seed refresh: %v", err)
	}

	// Add a second spec post-seed.
	writeRefreshSpec(t, heroDir, "beta", refreshSpec)
	stats, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if stats.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1; got %+v", stats.Indexed, stats)
	}
	if stats.Updated != 0 || stats.Removed != 0 {
		t.Errorf("expected only Indexed bump, got %+v", stats)
	}
}

func TestRefreshIfStale_UpdatesModifiedSpec(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	path := writeRefreshSpec(t, heroDir, "alpha", refreshSpec)
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Bump file mtime to something distinctly later than the seed.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	stats, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if stats.Updated != 1 {
		t.Errorf("Updated = %d, want 1; got %+v", stats.Updated, stats)
	}
	if stats.Indexed != 0 || stats.Removed != 0 {
		t.Errorf("expected only Updated bump, got %+v", stats)
	}
}

func TestRefreshIfStale_RemovesOrphanFromIndex(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	path := writeRefreshSpec(t, heroDir, "alpha", refreshSpec)
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Remove the spec from disk; index still has it.
	if err := os.RemoveAll(filepath.Dir(path)); err != nil {
		t.Fatalf("remove: %v", err)
	}

	stats, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if stats.Removed != 1 {
		t.Errorf("Removed = %d, want 1; got %+v", stats.Removed, stats)
	}

	// Confirm the slug is gone.
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()
	row := idx.db.QueryRow(`SELECT COUNT(*) FROM specs WHERE slug = ?`, "alpha")
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if n != 0 {
		t.Errorf("orphan still in index: count=%d", n)
	}
}

func TestRefreshIfStale_HandlesMixedDiff(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	pathA := writeRefreshSpec(t, heroDir, "alpha", refreshSpec)
	writeRefreshSpec(t, heroDir, "beta", refreshSpec)
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Modify alpha, delete beta, add gamma.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(pathA, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(heroDir, "specs", "beta")); err != nil {
		t.Fatalf("remove beta: %v", err)
	}
	writeRefreshSpec(t, heroDir, "gamma", refreshSpec)

	stats, err := RefreshIfStale(heroDir)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if stats.Indexed != 1 || stats.Updated != 1 || stats.Removed != 1 {
		t.Errorf("mixed diff: want 1/1/1, got Indexed=%d Updated=%d Removed=%d (full=%+v)",
			stats.Indexed, stats.Updated, stats.Removed, stats)
	}
}
