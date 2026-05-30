package embeddings

import (
	"database/sql"
	"math"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Enable foreign keys as the real index does.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enabling foreign keys: %v", err)
	}

	s, err := OpenStorage(db)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	return s
}

func TestOpenStorage(t *testing.T) {
	s := setupTestStorage(t)
	if s == nil {
		t.Fatal("storage should not be nil")
	}
}

func TestOpenStorage_IdempotentMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Open twice — second call should be a no-op.
	s1, err := OpenStorage(db)
	if err != nil {
		t.Fatalf("first OpenStorage: %v", err)
	}
	if s1 == nil {
		t.Fatal("s1 should not be nil")
	}

	s2, err := OpenStorage(db)
	if err != nil {
		t.Fatalf("second OpenStorage: %v", err)
	}
	if s2 == nil {
		t.Fatal("s2 should not be nil")
	}
}

func TestUpsert_Insert(t *testing.T) {
	s := setupTestStorage(t)

	chunk := Chunk{
		ID:       "spec:auth-flow:problem",
		Corpus:   "spec",
		SourceID: "auth-flow",
		Section:  "## Problem",
		TextHash: "abc123",
		Vector:   []float32{1, 0, 0, 0},
	}

	changed, err := s.Upsert(chunk)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for new insert")
	}

	// Verify it's stored by querying.
	results, err := s.QuerySimilar([]float32{1, 0, 0, 0}, 10, nil)
	if err != nil {
		t.Fatalf("QuerySimilar: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "spec:auth-flow:problem" {
		t.Errorf("got ID %q, want %q", results[0].ID, "spec:auth-flow:problem")
	}
	if results[0].Corpus != "spec" {
		t.Errorf("got Corpus %q, want %q", results[0].Corpus, "spec")
	}
	if results[0].SourceID != "auth-flow" {
		t.Errorf("got SourceID %q, want %q", results[0].SourceID, "auth-flow")
	}
	if results[0].Section != "## Problem" {
		t.Errorf("got Section %q, want %q", results[0].Section, "## Problem")
	}
}

func TestUpsert_Skip(t *testing.T) {
	s := setupTestStorage(t)

	chunk := Chunk{
		ID:       "spec:auth-flow:problem",
		Corpus:   "spec",
		SourceID: "auth-flow",
		Section:  "## Problem",
		TextHash: "abc123",
		Vector:   []float32{1, 0, 0, 0},
	}

	// First insert.
	changed, err := s.Upsert(chunk)
	if err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for first insert")
	}

	// Same hash -> no update.
	changed, err = s.Upsert(chunk)
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if changed {
		t.Error("expected changed=false for same text_hash")
	}
}

func TestUpsert_Update(t *testing.T) {
	s := setupTestStorage(t)

	chunk := Chunk{
		ID:       "spec:auth-flow:problem",
		Corpus:   "spec",
		SourceID: "auth-flow",
		TextHash: "abc123",
		Vector:   []float32{1, 0, 0, 0},
	}

	_, err := s.Upsert(chunk)
	if err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Different hash -> update.
	chunk.TextHash = "def456"
	chunk.Vector = []float32{0, 1, 0, 0}
	changed, err := s.Upsert(chunk)
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for different text_hash")
	}

	// Verify the vector was updated.
	results, err := s.QuerySimilar([]float32{0, 1, 0, 0}, 10, nil)
	if err != nil {
		t.Fatalf("QuerySimilar: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Score < 0.99 {
		t.Errorf("expected score ~1.0, got %f (vector not updated?)", results[0].Score)
	}
}

func TestDelete(t *testing.T) {
	s := setupTestStorage(t)

	chunk := Chunk{
		ID:       "spec:auth-flow:problem",
		Corpus:   "spec",
		SourceID: "auth-flow",
		TextHash: "abc123",
		Vector:   []float32{1, 0, 0, 0},
	}

	if _, err := s.Upsert(chunk); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete("spec:auth-flow:problem"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, err := s.QuerySimilar([]float32{1, 0, 0, 0}, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestDelete_NonExistent(t *testing.T) {
	s := setupTestStorage(t)

	// Deleting a non-existent chunk should not error.
	if err := s.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
}

func TestPruneCorpus(t *testing.T) {
	s := setupTestStorage(t)

	// Insert 3 chunks in "spec" corpus.
	for _, id := range []string{"spec:a:", "spec:b:", "spec:c:"} {
		c := Chunk{
			ID:       id,
			Corpus:   "spec",
			SourceID: id,
			TextHash: id,
			Vector:   []float32{1, 0, 0, 0},
		}
		if _, err := s.Upsert(c); err != nil {
			t.Fatal(err)
		}
	}

	// Insert 1 chunk in "code" corpus (should not be pruned).
	code := Chunk{
		ID:       "code:main:",
		Corpus:   "code",
		SourceID: "main",
		TextHash: "h1",
		Vector:   []float32{0, 1, 0, 0},
	}
	if _, err := s.Upsert(code); err != nil {
		t.Fatal(err)
	}

	// Keep only "spec:a:" and "spec:c:".
	pruned, err := s.PruneCorpus("spec", []string{"spec:a:", "spec:c:"})
	if err != nil {
		t.Fatalf("PruneCorpus: %v", err)
	}
	if pruned != 1 {
		t.Errorf("pruned = %d, want 1", pruned)
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.ByCorpus["spec"] != 2 {
		t.Errorf("spec count = %d, want 2", stats.ByCorpus["spec"])
	}
	if stats.ByCorpus["code"] != 1 {
		t.Errorf("code count = %d, want 1 (should be untouched)", stats.ByCorpus["code"])
	}
}

func TestPruneCorpus_EmptyKeepSet(t *testing.T) {
	s := setupTestStorage(t)

	c := Chunk{
		ID: "spec:a:", Corpus: "spec", SourceID: "a",
		TextHash: "h1", Vector: []float32{1, 0, 0, 0},
	}
	if _, err := s.Upsert(c); err != nil {
		t.Fatal(err)
	}

	pruned, err := s.PruneCorpus("spec", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Errorf("pruned = %d, want 1", pruned)
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 0 {
		t.Errorf("total = %d, want 0", stats.Total)
	}
}

func TestQuerySimilar_Ranking(t *testing.T) {
	s := setupTestStorage(t)

	// Insert chunks with known vectors.
	chunks := []Chunk{
		{ID: "a", Corpus: "spec", SourceID: "a", TextHash: "h1", Vector: []float32{1, 0, 0, 0}},
		{ID: "b", Corpus: "spec", SourceID: "b", TextHash: "h2", Vector: []float32{0, 1, 0, 0}},
		{ID: "c", Corpus: "spec", SourceID: "c", TextHash: "h3", Vector: normalizeVec([]float32{0.9, 0.1, 0, 0})},
	}
	for _, c := range chunks {
		if _, err := s.Upsert(c); err != nil {
			t.Fatal(err)
		}
	}

	// Query with [1,0,0,0] — "a" should be closest, then "c", then "b".
	results, err := s.QuerySimilar([]float32{1, 0, 0, 0}, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].ID != "a" {
		t.Errorf("top result = %q, want %q", results[0].ID, "a")
	}
	if results[1].ID != "c" {
		t.Errorf("second result = %q, want %q", results[1].ID, "c")
	}
	if results[2].ID != "b" {
		t.Errorf("third result = %q, want %q", results[2].ID, "b")
	}
}

func TestQuerySimilar_CorpusFilter(t *testing.T) {
	s := setupTestStorage(t)

	chunks := []Chunk{
		{ID: "spec:1", Corpus: "spec", SourceID: "1", TextHash: "h1", Vector: []float32{1, 0, 0, 0}},
		{ID: "code:1", Corpus: "code", SourceID: "1", TextHash: "h2", Vector: []float32{1, 0, 0, 0}},
		{ID: "know:1", Corpus: "knowledge", SourceID: "1", TextHash: "h3", Vector: []float32{1, 0, 0, 0}},
	}
	for _, c := range chunks {
		if _, err := s.Upsert(c); err != nil {
			t.Fatal(err)
		}
	}

	// Filter to "spec" only.
	results, err := s.QuerySimilar([]float32{1, 0, 0, 0}, 10, []string{"spec"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for corpus=spec, got %d", len(results))
	}
	if results[0].ID != "spec:1" {
		t.Errorf("result ID = %q, want %q", results[0].ID, "spec:1")
	}

	// Filter to "spec" and "code".
	results, err = s.QuerySimilar([]float32{1, 0, 0, 0}, 10, []string{"spec", "code"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for corpora=[spec,code], got %d", len(results))
	}
}

func TestQuerySimilar_TopK(t *testing.T) {
	s := setupTestStorage(t)

	for i := 0; i < 5; i++ {
		c := Chunk{
			ID: "spec:" + string(rune('a'+i)), Corpus: "spec",
			SourceID: string(rune('a' + i)), TextHash: string(rune('a' + i)),
			Vector: []float32{1, 0, 0, 0},
		}
		if _, err := s.Upsert(c); err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.QuerySimilar([]float32{1, 0, 0, 0}, 3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results with topK=3, got %d", len(results))
	}
}

func TestStats(t *testing.T) {
	s := setupTestStorage(t)

	chunks := []Chunk{
		{ID: "spec:a", Corpus: "spec", SourceID: "a", TextHash: "h1", Vector: []float32{1, 0, 0, 0}},
		{ID: "spec:b", Corpus: "spec", SourceID: "b", TextHash: "h2", Vector: []float32{0, 1, 0, 0}},
		{ID: "code:a", Corpus: "code", SourceID: "a", TextHash: "h3", Vector: []float32{0, 0, 1, 0}},
	}
	for _, c := range chunks {
		if _, err := s.Upsert(c); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 {
		t.Errorf("total = %d, want 3", stats.Total)
	}
	if stats.ByCorpus["spec"] != 2 {
		t.Errorf("spec = %d, want 2", stats.ByCorpus["spec"])
	}
	if stats.ByCorpus["code"] != 1 {
		t.Errorf("code = %d, want 1", stats.ByCorpus["code"])
	}
}

func TestStats_Empty(t *testing.T) {
	s := setupTestStorage(t)

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 0 {
		t.Errorf("total = %d, want 0", stats.Total)
	}
}

func TestVectorRoundTrip(t *testing.T) {
	// Verify encodeVector/decodeVector round-trip correctly.
	original := []float32{1.5, -2.3, 0, math.SmallestNonzeroFloat32, math.MaxFloat32}
	encoded := encodeVector(original)
	decoded := decodeVector(encoded)

	if len(decoded) != len(original) {
		t.Fatalf("decoded length = %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("decoded[%d] = %v, want %v", i, decoded[i], original[i])
		}
	}
}

func TestStorageWithExistingDB(t *testing.T) {
	// Simulate opening storage on a database that already has other tables
	// (like the real index.db with specs, fts_specs, etc.).
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create a table that mimics the index.db schema.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS specs (slug TEXT PRIMARY KEY)`)
	if err != nil {
		t.Fatalf("creating specs table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO specs VALUES ('test-slug')`)
	if err != nil {
		t.Fatalf("inserting test spec: %v", err)
	}

	// Opening storage should not interfere with existing tables.
	s, err := OpenStorage(db)
	if err != nil {
		t.Fatalf("OpenStorage with existing tables: %v", err)
	}

	// Verify the existing table is still intact.
	var slug string
	if err := db.QueryRow(`SELECT slug FROM specs`).Scan(&slug); err != nil {
		t.Fatalf("querying existing table: %v", err)
	}
	if slug != "test-slug" {
		t.Errorf("existing data corrupted: got %q", slug)
	}

	// Verify storage works.
	c := Chunk{
		ID: "test:chunk", Corpus: "test", SourceID: "x",
		TextHash: "h", Vector: []float32{1, 0, 0, 0},
	}
	changed, err := s.Upsert(c)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
}

// --- Helpers ---

// normalizeVec returns a new L2-normalized copy of vec.
func normalizeVec(vec []float32) []float32 {
	out := make([]float32, len(vec))
	copy(out, vec)
	var sum float32
	for _, v := range out {
		sum += v * v
	}
	inv := float32(1.0 / math.Sqrt(float64(sum)))
	for i := range out {
		out[i] *= inv
	}
	return out
}

