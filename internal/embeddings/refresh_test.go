package embeddings

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupRefreshEnv(t *testing.T) (heroDir string, model *Model, indexDB *sql.DB) {
	t.Helper()

	heroDir = t.TempDir()
	model = makeTestModel(t)

	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening index db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enabling foreign keys: %v", err)
	}

	return heroDir, model, db
}

func TestRefresh_FullRun(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)

	// Create a spec.
	writeSpecFile(t, heroDir, "planning/features/auth", `---
title: Auth
slug: auth
type: feature
status: planning
---
## Goal
Implement authentication.

## Problem
No auth exists.
`)

	// Create a knowledge file.
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(knowledgeDir, "api.md"),
		[]byte("API design notes."),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	stats, err := Refresh(heroDir, model, indexDB, nil, []string{"spec", "knowledge"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// Should have 2 spec sections + 1 knowledge file = 3 added.
	if stats.Added != 3 {
		t.Errorf("Added = %d, want 3", stats.Added)
	}
	if stats.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", stats.Skipped)
	}
	if stats.Pruned != 0 {
		t.Errorf("Pruned = %d, want 0", stats.Pruned)
	}
	if stats.Elapsed <= 0 {
		t.Error("Elapsed should be positive")
	}

	// Verify chunks are in storage.
	storage, err := OpenStorage(indexDB)
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 3 {
		t.Errorf("storage total = %d, want 3", st.Total)
	}
}

func TestRefresh_Incremental(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)

	writeSpecFile(t, heroDir, "planning/features/cache", `---
title: Cache
slug: cache
type: feature
status: planning
---
## Goal
Add caching.
`)

	// First refresh.
	stats1, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatalf("first Refresh: %v", err)
	}
	if stats1.Added != 1 {
		t.Errorf("first: Added = %d, want 1", stats1.Added)
	}

	// Second refresh with no changes — should skip.
	stats2, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatalf("second Refresh: %v", err)
	}
	if stats2.Added != 0 {
		t.Errorf("second: Added = %d, want 0", stats2.Added)
	}
	if stats2.Skipped != 1 {
		t.Errorf("second: Skipped = %d, want 1", stats2.Skipped)
	}

	// Edit the spec content.
	writeSpecFile(t, heroDir, "planning/features/cache", `---
title: Cache
slug: cache
type: feature
status: planning
---
## Goal
Add caching with Redis backend.
`)

	// Third refresh — should detect change and re-embed.
	stats3, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatalf("third Refresh: %v", err)
	}
	if stats3.Added != 1 {
		t.Errorf("third: Added = %d, want 1 (updated chunk)", stats3.Added)
	}
	if stats3.Skipped != 0 {
		t.Errorf("third: Skipped = %d, want 0", stats3.Skipped)
	}
}

func TestRefresh_Prune(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)

	// Create two specs.
	writeSpecFile(t, heroDir, "planning/features/alpha", `---
title: Alpha
slug: alpha
type: feature
status: planning
---
## Goal
Alpha feature.
`)
	writeSpecFile(t, heroDir, "planning/features/beta", `---
title: Beta
slug: beta
type: feature
status: planning
---
## Goal
Beta feature.
`)

	// First refresh — both specs indexed.
	stats1, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatal(err)
	}
	if stats1.Added != 2 {
		t.Errorf("first: Added = %d, want 2", stats1.Added)
	}

	// Delete the beta spec.
	betaDir := filepath.Join(heroDir, "planning/features/beta")
	if err := os.RemoveAll(betaDir); err != nil {
		t.Fatalf("removing beta spec: %v", err)
	}

	// Second refresh — beta should be pruned.
	stats2, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatal(err)
	}
	if stats2.Pruned != 1 {
		t.Errorf("second: Pruned = %d, want 1", stats2.Pruned)
	}

	// Verify only alpha remains.
	storage, err := OpenStorage(indexDB)
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.ByCorpus["spec"] != 1 {
		t.Errorf("spec count = %d, want 1", st.ByCorpus["spec"])
	}
}

func TestRefresh_NilGraphDB(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)

	// Refresh with events and code in scope but nil graphDB — should skip gracefully.
	stats, err := Refresh(heroDir, model, indexDB, nil, []string{"event", "code"})
	if err != nil {
		t.Fatalf("Refresh with nil graphDB: %v", err)
	}

	if stats.Added != 0 {
		t.Errorf("Added = %d, want 0", stats.Added)
	}
	if stats.Pruned != 0 {
		t.Errorf("Pruned = %d, want 0", stats.Pruned)
	}
}

func TestRefresh_ScopeFilter(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)

	// Create a spec and a knowledge file.
	writeSpecFile(t, heroDir, "planning/features/scoped", `---
title: Scoped
slug: scoped
type: feature
status: planning
---
## Goal
Test scope filtering.
`)

	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(knowledgeDir, "note.md"), []byte("A note."), 0o644); err != nil {
		t.Fatal(err)
	}

	// Refresh only spec corpus.
	stats, err := Refresh(heroDir, model, indexDB, nil, []string{"spec"})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Added != 1 {
		t.Errorf("Added = %d, want 1 (only spec)", stats.Added)
	}

	// Knowledge should not be in storage.
	storage, err := OpenStorage(indexDB)
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.ByCorpus["knowledge"]; ok {
		t.Error("knowledge corpus should not be in storage when not in scope")
	}
}

func TestRefresh_StatsString(t *testing.T) {
	stats := RefreshStats{
		Added:   5,
		Updated: 2,
		Pruned:  1,
		Skipped: 10,
	}

	s := stats.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	// Verify it contains the key fields.
	for _, want := range []string{"added=5", "updated=2", "pruned=1", "skipped=10"} {
		if !contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}

func TestTextHash_Deterministic(t *testing.T) {
	text := "hello world"
	h1 := textHash(text)
	h2 := textHash(text)

	if h1 != h2 {
		t.Errorf("textHash not deterministic: %q != %q", h1, h2)
	}

	// Different text should produce different hash.
	h3 := textHash("different text")
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}

	// Verify it's a valid hex string of expected length (sha256 = 64 hex chars).
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64", len(h1))
	}
}

func TestRefresh_WithEvents(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)
	graphDB := setupGraphTestDB(t)

	// Insert event nodes.
	insertTestEvent(t, graphDB, "UserAsk", "ask-1", "How to test?", "Unit tests with Go.")
	insertTestEvent(t, graphDB, "SessionReflection", "refl-1", "Summary", "Completed auth feature.")

	stats, err := Refresh(heroDir, model, indexDB, graphDB, []string{"event"})
	if err != nil {
		t.Fatalf("Refresh with events: %v", err)
	}

	if stats.Added != 2 {
		t.Errorf("Added = %d, want 2", stats.Added)
	}

	// Verify chunks in storage.
	storage, err := OpenStorage(indexDB)
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.ByCorpus["event"] != 2 {
		t.Errorf("event count = %d, want 2", st.ByCorpus["event"])
	}
}

func TestRefresh_WithCodeSymbols(t *testing.T) {
	heroDir, model, indexDB := setupRefreshEnv(t)
	graphDB := setupGraphTestDB(t)

	insertTestSymbol(t, graphDB, "pkg.Func", "func", "func Func()", "Does stuff.", "func Func() {}", "internal/pkg/func.go")

	stats, err := Refresh(heroDir, model, indexDB, graphDB, []string{"code"})
	if err != nil {
		t.Fatalf("Refresh with code: %v", err)
	}

	if stats.Added != 1 {
		t.Errorf("Added = %d, want 1", stats.Added)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
