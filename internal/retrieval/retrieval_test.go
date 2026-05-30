package retrieval

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newHeroDir creates a temp dir with both graph.db and index.db fully
// initialised using the real package constructors.
func newHeroDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Open (and immediately close) both stores to run their migrations.
	gstore, err := graph.Open(dir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	gstore.Close()

	idb, err := index.Open(dir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	idb.Close()
	return dir
}

// newHeroDirFTSOnly creates a temp dir with only the FTS5 index.
func newHeroDirFTSOnly(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	idb, err := index.Open(dir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	idb.Close()
	return dir
}

// addGraphNode inserts a node into the graph using graph.Open + UpsertNode.
func addGraphNode(t *testing.T, heroDir, nodeType, key string, props map[string]any) {
	t.Helper()
	gstore, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer gstore.Close()

	domain := "engineering"
	if graph.IsGlobalNodeType(nodeType) {
		domain = ""
	}
	_, err = gstore.UpsertNode(&graph.Node{
		Type:   nodeType,
		Domain: domain,
		Key:    key,
		Props:  props,
		Scope:  graph.ScopeTeam,
		Source: map[string]any{"_test": true},
	})
	if err != nil {
		t.Fatalf("UpsertNode(%s, %s): %v", nodeType, key, err)
	}
}

// addFTSSpec creates a spec on disk and indexes it into the FTS5 index.
func addFTSSpec(t *testing.T, heroDir, slug, title, specType, content string) {
	t.Helper()

	path := filepath.Join(heroDir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	now := time.Now()
	sp := &spec.Spec{
		Slug:       slug,
		Title:      title,
		Type:       spec.Type(specType),
		Status:     spec.StatusPlanning,
		Path:       path,
		CreatedAt:  now,
		ModifiedAt: now,
	}

	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	defer idb.Close()

	if err := idb.IndexSpec(sp, content); err != nil {
		t.Fatalf("IndexSpec: %v", err)
	}
}

// projectNodes opens the index and graph DBs in heroDir and runs
// ProjectGraphNodes to populate fts_nodes + node_index.
func projectNodes(t *testing.T, heroDir string) int {
	t.Helper()
	gstore, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("graph.Open for projection: %v", err)
	}
	defer gstore.Close()

	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open for projection: %v", err)
	}
	defer idb.Close()

	n, err := idb.ProjectGraphNodes(gstore.DB())
	if err != nil {
		t.Fatalf("ProjectGraphNodes: %v", err)
	}
	return n
}

// ---------------------------------------------------------------------------
// Routing unit tests
// ---------------------------------------------------------------------------

// TestRoutingFiltersGoFTS verifies that a Query with Filters routes to FTS5.
func TestRoutingFiltersGoFTS(t *testing.T) {
	heroDir := newHeroDirFTSOnly(t)
	addFTSSpec(t, heroDir, "auth-feature", "Auth Feature", "feature", "authentication feature spec")

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{
		Text:    "auth",
		Filters: map[string]string{"status": "planning"},
	})
	if err != nil {
		t.Fatalf("Retrieve with filter: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected FTS5 results, got none")
	}
	for _, r := range results {
		if r.Source != "fts5" {
			t.Errorf("expected source=fts5, got %q", r.Source)
		}
	}
}

// TestRoutingTypesGoFTS verifies that a Query with Types routes to FTS5.
func TestRoutingTypesGoFTS(t *testing.T) {
	heroDir := newHeroDirFTSOnly(t)
	addFTSSpec(t, heroDir, "my-decision", "My Decision", "decision", "an architectural decision")

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{
		Text:  "architectural",
		Types: []string{"decision"},
	})
	if err != nil {
		t.Fatalf("Retrieve with types: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected FTS5 results, got none")
	}
	for _, r := range results {
		if r.Source != "fts5" {
			t.Errorf("expected source=fts5, got %q", r.Source)
		}
	}
}

// TestRoutingGraphFirst verifies that a plain-text query uses graph-first routing.
func TestRoutingGraphFirst(t *testing.T) {
	heroDir := newHeroDir(t)
	addGraphNode(t, heroDir, "Feature", "routing-test-feature", map[string]any{
		"title": "routing test unique marker xyzquux",
		"body":  "routing test marker content",
	})

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "xyzquux"})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected graph results, got none")
	}
	if results[0].Source != "graph" {
		t.Errorf("expected source=graph for first result, got %q", results[0].Source)
	}
}

// TestRoutingFallbackToFTS verifies that when the graph has zero hits for a
// plain-text query, the retrieval layer falls through to FTS5.
func TestRoutingFallbackToFTS(t *testing.T) {
	heroDir := newHeroDir(t)
	// Only insert in FTS5, not in the graph.
	addFTSSpec(t, heroDir, "fts-only-spec", "FTS Only Spec", "feature",
		"fallbackonlyterm content that lives only in the FTS5 index")

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "fallbackonlyterm"})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected FTS5 fallback results, got none")
	}
	for _, r := range results {
		if r.Source != "fts5" {
			t.Errorf("expected source=fts5 in fallback, got %q", r.Source)
		}
	}
}

// ---------------------------------------------------------------------------
// Type-boost smoke test (d9997ea regression)
// ---------------------------------------------------------------------------

// TestTypeBoostRegressionD9997ea is the regression test for the architectural
// smell documented in commit d9997ea: "The 'Task' search on go-task/task used
// to return 99 commit messages and 1 useful result."
//
// A corpus with many Commit nodes and one Feature node matching the same query
// term must return the Feature first. If Commits drown the Feature, type boosts
// are not working correctly as MULTIPLIERS.
func TestTypeBoostRegressionD9997ea(t *testing.T) {
	heroDir := newHeroDir(t)

	// One high-signal Feature that matches "task".
	addGraphNode(t, heroDir, "Feature", "task-manager-feature", map[string]any{
		"title": "Task Manager Feature",
		"body":  "Task management implementation spec",
	})

	// Fifty low-signal Commit nodes that also match "task".
	for i := 0; i < 50; i++ {
		addGraphNode(t, heroDir, "Commit", fmt.Sprintf("commit-%03d", i), map[string]any{
			"subject": fmt.Sprintf("fix: Task item %d in worker", i),
		})
	}

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "task", Limit: 30})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// The Feature must be ranked first — type boost 10× vs Commit's 1×.
	top := results[0]
	if top.Type != "Feature" {
		t.Errorf("d9997ea regression: expected Feature at rank 1, got %q (key=%q, score=%v)",
			top.Type, top.Key, top.Score)
		for i, r := range results {
			if i >= 5 {
				break
			}
			t.Logf("  [%d] type=%s key=%s score=%v", i, r.Type, r.Key, r.Score)
		}
	}

	// Every Commit score must be strictly less than the Feature score.
	for _, r := range results {
		if r.Type == "Commit" && r.Score >= top.Score {
			t.Errorf("d9997ea regression: Commit score %v >= Feature score %v — type boost not applied",
				r.Score, top.Score)
			break
		}
	}
}

// ---------------------------------------------------------------------------
// typeBoost unit tests
// ---------------------------------------------------------------------------

func TestTypeBoostValues(t *testing.T) {
	cases := []struct {
		nodeType string
		want     float64
	}{
		{"Feature", 10},
		{"Bug", 10},
		{"Initiative", 10},
		{"Decision", 10},
		{"Convention", 9},
		{"Rule", 9},
		{"ContextDoc", 8},
		{"Note", 8},
		{"Symbol", 6},
		{"Package", 6},
		{"File", 5},
		{"Issue", 3},
		{"Commit", 1},
		{"Person", 1},
		{"Unknown", 4},
	}
	for _, c := range cases {
		got := typeBoost(c.nodeType)
		if got != c.want {
			t.Errorf("typeBoost(%q) = %v, want %v", c.nodeType, got, c.want)
		}
	}
}

// TestTypeBoostIsMultiplier verifies that scores are computed as
// base * typeBoost and not as additive weights. Two nodes with identical text
// match quality (same key match) must have scores in the ratio of their type
// boosts.
func TestTypeBoostIsMultiplier(t *testing.T) {
	heroDir := newHeroDir(t)

	addGraphNode(t, heroDir, "Feature", "multipliertest-xq7", map[string]any{
		"title": "multipliertest-xq7",
	})
	addGraphNode(t, heroDir, "Commit", "multipliertest-xq7-commit", map[string]any{
		"subject": "multipliertest-xq7 in worker",
	})

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "multipliertest-xq7", Limit: 10})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	var featureScore, commitScore float64
	for _, r := range results {
		switch r.Type {
		case "Feature":
			featureScore = r.Score
		case "Commit":
			commitScore = r.Score
		}
	}

	if featureScore == 0 {
		t.Fatal("Feature not found in results")
	}
	if commitScore == 0 {
		t.Fatal("Commit not found in results")
	}

	// Both nodes key-match the query, so both get base=2.0.
	// Feature score = 2 * typeBoost("Feature") = 2 * 10 = 20.
	// Commit score  = 2 * typeBoost("Commit")  = 2 *  1 =  2.
	// The ratio equals the typeBoost ratio (10), proving boosts are
	// MULTIPLIERS rather than additive constants.
	ratio := featureScore / commitScore
	const wantRatio = 10.0
	if ratio != wantRatio {
		t.Errorf("type boost ratio = %v, want %v (feature=%v commit=%v)",
			ratio, wantRatio, featureScore, commitScore)
	}
}

// ---------------------------------------------------------------------------
// Phase B — projection + BM25 tests
// ---------------------------------------------------------------------------

// TestProjectionPopulatesFTSNodes verifies that ProjectGraphNodes writes all
// current graph nodes into fts_nodes + node_index.
func TestProjectionPopulatesFTSNodes(t *testing.T) {
	heroDir := newHeroDir(t)

	addGraphNode(t, heroDir, "Feature", "auth-feature", map[string]any{
		"title": "Authentication Feature",
		"body":  "OAuth2 login flow implementation",
	})
	addGraphNode(t, heroDir, "Commit", "abc123", map[string]any{
		"subject": "fix: authentication retry logic",
	})
	addGraphNode(t, heroDir, "ContextDoc", "auth-overview", map[string]any{
		"title": "Auth Architecture Overview",
		"body":  "Describes the authentication architecture",
	})

	n := projectNodes(t, heroDir)
	if n != 3 {
		t.Fatalf("ProjectGraphNodes returned %d, want 3", n)
	}

	// Verify via raw SQL that fts_nodes and node_index are populated.
	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	defer idb.Close()

	var count int
	err = idb.RawDB().QueryRow("SELECT count(*) FROM node_index").Scan(&count)
	if err != nil {
		t.Fatalf("count node_index: %v", err)
	}
	if count != 3 {
		t.Errorf("node_index has %d rows, want 3", count)
	}

	var ftsCount int
	err = idb.RawDB().QueryRow("SELECT count(*) FROM fts_nodes").Scan(&ftsCount)
	if err != nil {
		t.Fatalf("count fts_nodes: %v", err)
	}
	if ftsCount != 3 {
		t.Errorf("fts_nodes has %d rows, want 3", ftsCount)
	}
}

// TestBM25RankingViaNodeIndex verifies that after projection, a Retrieve()
// call routes through the unified node index (BM25) and returns results
// ranked by BM25 * typeBoost.
func TestBM25RankingViaNodeIndex(t *testing.T) {
	heroDir := newHeroDir(t)

	addGraphNode(t, heroDir, "Feature", "task-manager", map[string]any{
		"title": "Task Manager Feature",
		"body":  "Task management spec with comprehensive task handling",
	})
	addGraphNode(t, heroDir, "Commit", "commit-task-001", map[string]any{
		"subject": "fix: task item in worker",
	})

	projectNodes(t, heroDir)

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "task", Limit: 10})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results from node index, got none")
	}

	// Feature must rank above Commit due to typeBoost(Feature)=10 vs typeBoost(Commit)=1.
	if results[0].Type != "Feature" {
		t.Errorf("expected Feature at rank 1, got %q (key=%q score=%v)",
			results[0].Type, results[0].Key, results[0].Score)
	}

	// Source should be "graph" (from the unified node index path).
	for _, r := range results {
		if r.Source != "graph" {
			t.Errorf("expected source=graph, got %q for %s/%s", r.Source, r.Type, r.Key)
		}
	}
}

// TestBM25RegressionD9997ea is the Phase B version of the d9997ea regression
// test. With BM25 ranking, 50 Commits matching "task" must not drown the
// single Feature — typeBoost multipliers ensure the Feature ranks first.
func TestBM25RegressionD9997ea(t *testing.T) {
	heroDir := newHeroDir(t)

	addGraphNode(t, heroDir, "Feature", "task-manager-feature", map[string]any{
		"title": "Task Manager Feature",
		"body":  "Task management implementation spec",
	})

	for i := 0; i < 50; i++ {
		addGraphNode(t, heroDir, "Commit", fmt.Sprintf("commit-%03d", i), map[string]any{
			"subject": fmt.Sprintf("fix: Task item %d in worker", i),
		})
	}

	projectNodes(t, heroDir)

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "task", Limit: 30})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	top := results[0]
	if top.Type != "Feature" {
		t.Errorf("d9997ea BM25 regression: expected Feature at rank 1, got %q (key=%q score=%v)",
			top.Type, top.Key, top.Score)
		for i, r := range results {
			if i >= 5 {
				break
			}
			t.Logf("  [%d] type=%s key=%s score=%v", i, r.Type, r.Key, r.Score)
		}
	}

	for _, r := range results {
		if r.Type == "Commit" && r.Score >= top.Score {
			t.Errorf("d9997ea BM25 regression: Commit score %v >= Feature score %v",
				r.Score, top.Score)
			break
		}
	}
}

// TestBM25FallbackToGraphLIKE verifies AC-5: when fts_nodes has zero matches
// (e.g. no projection has run), retrieval falls through to graph LIKE matching.
func TestBM25FallbackToGraphLIKE(t *testing.T) {
	heroDir := newHeroDir(t)

	// Insert into graph but do NOT project — fts_nodes is empty.
	addGraphNode(t, heroDir, "Feature", "unique-xyzfoo-feature", map[string]any{
		"title": "unique xyzfoo feature",
	})

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{Text: "xyzfoo"})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected graph LIKE fallback results, got none")
	}
	if results[0].Source != "graph" {
		t.Errorf("expected source=graph for fallback, got %q", results[0].Source)
	}
}

// ---------------------------------------------------------------------------
// Phase C — fuseRRF unit tests
// ---------------------------------------------------------------------------

// TestFuseRRF_BothSets verifies that results appearing in both lexical and
// vector rankings get a higher RRF score than results in only one set, and
// that dual-ranked results have Source "hybrid".
func TestFuseRRF_BothSets(t *testing.T) {
	lexical := []Result{
		{Key: "auth-feature", Title: "Auth Feature", Type: "Feature", Source: "graph", Score: 20},
		{Key: "login-bug", Title: "Login Bug", Type: "Bug", Source: "graph", Score: 10},
	}

	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "spec:auth-feature:", Corpus: "spec", SourceID: "auth-feature"}, Score: 0.95},
		{Chunk: embeddings.Chunk{ID: "knowledge:security.md", Corpus: "knowledge", SourceID: "security.md"}, Score: 0.80},
	}

	results := fuseRRF(lexical, vector, 60, 10)

	if len(results) == 0 {
		t.Fatal("fuseRRF returned zero results")
	}

	// auth-feature should be ranked first (appears in both sets).
	if results[0].Key != "auth-feature" {
		t.Errorf("expected auth-feature at rank 1, got %q", results[0].Key)
	}
	if results[0].Source != "hybrid" {
		t.Errorf("expected source=hybrid for dual-ranked result, got %q", results[0].Source)
	}

	// auth-feature (in both) should have a higher score than login-bug (lexical only).
	var authScore, loginScore float64
	for _, r := range results {
		switch r.Key {
		case "auth-feature":
			authScore = r.Score
		case "login-bug":
			loginScore = r.Score
		}
	}
	if authScore <= loginScore {
		t.Errorf("dual-ranked auth-feature score %v should be > single-ranked login-bug %v",
			authScore, loginScore)
	}

	// login-bug should keep source="graph" (only in lexical).
	for _, r := range results {
		if r.Key == "login-bug" && r.Source != "graph" {
			t.Errorf("expected login-bug source=graph, got %q", r.Source)
		}
	}

	// security.md should appear from vector-only with source="vector".
	// For non-spec corpora the matching key is the chunk ID.
	found := false
	for _, r := range results {
		if r.Key == "knowledge:security.md" {
			found = true
			if r.Source != "vector" {
				t.Errorf("expected knowledge:security.md source=vector, got %q", r.Source)
			}
		}
	}
	if !found {
		t.Error("vector-only result knowledge:security.md not found in fused results")
	}
}

// TestFuseRRF_Empty verifies that fuseRRF handles empty inputs gracefully.
func TestFuseRRF_Empty(t *testing.T) {
	results := fuseRRF(nil, nil, 60, 10)
	if len(results) != 0 {
		t.Errorf("expected zero results for empty inputs, got %d", len(results))
	}

	// Lexical only.
	lexical := []Result{{Key: "a", Source: "graph"}}
	results = fuseRRF(lexical, nil, 60, 10)
	if len(results) != 1 || results[0].Key != "a" {
		t.Errorf("expected 1 lexical-only result, got %d", len(results))
	}

	// Vector only.
	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "spec:b:", Corpus: "spec", SourceID: "b"}, Score: 0.9},
	}
	results = fuseRRF(nil, vector, 60, 10)
	if len(results) != 1 || results[0].Key != "b" {
		t.Errorf("expected 1 vector-only result, got %d", len(results))
	}
}

// TestFuseRRF_LimitRespected verifies that fuseRRF respects the limit.
func TestFuseRRF_LimitRespected(t *testing.T) {
	var lexical []Result
	for i := 0; i < 20; i++ {
		lexical = append(lexical, Result{Key: fmt.Sprintf("lex-%d", i), Source: "graph"})
	}
	var vector []embeddings.ScoredChunk
	for i := 0; i < 20; i++ {
		vector = append(vector, embeddings.ScoredChunk{
			Chunk: embeddings.Chunk{ID: fmt.Sprintf("spec:vec-%d:", i), Corpus: "spec", SourceID: fmt.Sprintf("vec-%d", i)},
			Score: float64(20-i) / 20.0,
		})
	}

	results := fuseRRF(lexical, vector, 60, 5)
	if len(results) != 5 {
		t.Errorf("expected 5 results (limit), got %d", len(results))
	}
}

// TestRetrieveHybrid_WithEmbeddedModel tests the full hybrid retrieval path
// using the embedded model. It sets up specs and knowledge in both FTS5 and
// the vector index, then verifies hybrid search returns results from both.
func TestRetrieveHybrid_WithEmbeddedModel(t *testing.T) {
	heroDir := newHeroDir(t)

	// Insert graph nodes.
	addGraphNode(t, heroDir, "Feature", "auth-retry-spec", map[string]any{
		"title": "Authentication Retry Logic",
		"body":  "Implement exponential backoff for failed authentication attempts",
	})
	addGraphNode(t, heroDir, "Feature", "scan-pipeline-spec", map[string]any{
		"title": "Scan Pipeline Optimization",
		"body":  "Improve scan performance by parallelizing file tree walk",
	})
	addGraphNode(t, heroDir, "Feature", "landing-page-design", map[string]any{
		"title": "Landing Page Design",
		"body":  "Create marketing landing page with hero bolt logo",
	})

	// Project into node_index for BM25.
	projectNodes(t, heroDir)

	// Open the retriever (will load the embedded model).
	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	if ret.embModel == nil {
		t.Skip("embedded model not available")
	}

	// Populate vec_chunks with test content.
	store, err := embeddings.OpenStorage(ret.fts.RawDB())
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}

	texts := map[string]string{
		"auth-retry-spec":    "Authentication retry with exponential backoff for failed login attempts",
		"scan-pipeline-spec": "Scan pipeline optimization for parallel file tree walking and ingestion",
		"landing-page-design": "Marketing landing page with hero bolt logo and call to action",
	}
	for key, text := range texts {
		vec := ret.embModel.Embed(text)
		_, err := store.Upsert(embeddings.Chunk{
			ID:       fmt.Sprintf("spec:%s:", key),
			Corpus:   "spec",
			SourceID: key,
			TextHash: fmt.Sprintf("hash-%s", key),
			Vector:   vec,
		})
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}
	ret.embStore = store

	// Query for something semantically related to auth retry.
	results, err := ret.Retrieve(Query{
		Text:       "login failure backoff",
		SemanticOK: true,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Retrieve hybrid: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected hybrid results, got none")
	}

	// The auth-retry spec should rank highly — it matches both
	// the lexical content and the semantic meaning.
	foundAuth := false
	for i, r := range results {
		if r.Key == "auth-retry-spec" {
			foundAuth = true
			if i > 3 {
				t.Errorf("auth-retry-spec ranked %d, expected top-3", i)
			}
			// Should be marked hybrid (appears in both rankings).
			t.Logf("auth-retry-spec: rank=%d source=%s score=%.4f", i, r.Source, r.Score)
			break
		}
	}
	if !foundAuth {
		t.Error("auth-retry-spec not found in hybrid results")
		for i, r := range results {
			if i >= 5 {
				break
			}
			t.Logf("  [%d] key=%s source=%s score=%.4f", i, r.Key, r.Source, r.Score)
		}
	}
}

// TestRetrieveHybrid_VectorOnlyResult verifies that vector-only results
// (not in lexical set) still appear in hybrid output with source="vector".
func TestRetrieveHybrid_VectorOnlyResult(t *testing.T) {
	heroDir := newHeroDir(t)

	// Only add one node to the graph (lexical), but embed two specs.
	addGraphNode(t, heroDir, "Feature", "visible-spec", map[string]any{
		"title": "Visible Spec Unique Marker qrstvwx",
	})
	projectNodes(t, heroDir)

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	if ret.embModel == nil {
		t.Skip("embedded model not available")
	}

	store, err := embeddings.OpenStorage(ret.fts.RawDB())
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}

	// Embed a spec that does NOT exist in the graph.
	vec := ret.embModel.Embed("database migration schema upgrade for authentication tables")
	_, err = store.Upsert(embeddings.Chunk{
		ID:       "spec:hidden-spec:",
		Corpus:   "spec",
		SourceID: "hidden-spec",
		TextHash: "hash-hidden",
		Vector:   vec,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	ret.embStore = store

	results, err := ret.Retrieve(Query{
		Text:       "database migration schema",
		SemanticOK: true,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	// hidden-spec should appear from vector search only.
	found := false
	for _, r := range results {
		if r.Key == "hidden-spec" {
			found = true
			if r.Source != "vector" {
				t.Errorf("expected source=vector for vector-only result, got %q", r.Source)
			}
			break
		}
	}
	if !found {
		t.Error("hidden-spec (vector-only) not found in hybrid results")
	}
}

// TestRetrieve_SemanticOK_NoModel verifies that when SemanticOK=true but no
// embedding model is loaded, retrieval gracefully falls through to BM25-only.
func TestRetrieve_SemanticOK_NoModel(t *testing.T) {
	heroDir := newHeroDir(t)

	addGraphNode(t, heroDir, "Feature", "semantic-test-feature", map[string]any{
		"title": "Semantic test unique marker xyzquux",
		"body":  "Semantic test marker content",
	})

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	// Ensure no embedding model is loaded (default for test environments).
	if ret.embModel != nil {
		t.Skip("embedding model unexpectedly available in test environment")
	}

	results, err := ret.Retrieve(Query{Text: "xyzquux", SemanticOK: true})
	if err != nil {
		t.Fatalf("Retrieve with SemanticOK: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results from BM25 fallback, got none")
	}
	// Should fall through to graph/FTS5 path when no model is available.
	if results[0].Source != "graph" {
		t.Errorf("expected source=graph for BM25 fallback, got %q", results[0].Source)
	}
}
