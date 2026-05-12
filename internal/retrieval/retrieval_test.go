package retrieval

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	_, err = gstore.UpsertNode(&graph.Node{
		Type:  nodeType,
		Key:   key,
		Props: props,
		Scope: graph.ScopeTeam,
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
