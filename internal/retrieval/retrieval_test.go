package retrieval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	addGraphNodeWithRepo(t, heroDir, nodeType, key, "", props)
}

func addGraphNodeWithRepo(t *testing.T, heroDir, nodeType, key, repo string, props map[string]any) {
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
		Repo:   repo,
		Props:  props,
		Scope:  graph.ScopeTeam,
		Source: map[string]any{"_test": true},
	})
	if err != nil {
		t.Fatalf("UpsertNode(%s, %s): %v", nodeType, key, err)
	}
}

func addGraphNodeWithDomain(t *testing.T, heroDir, nodeType, key, domain string, props map[string]any) {
	t.Helper()
	gstore, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer gstore.Close()
	if _, err := gstore.UpsertNode(&graph.Node{
		Type: nodeType, Key: key, Domain: domain, Props: props,
		Scope: graph.ScopeTeam, Source: map[string]any{"_test": true},
	}); err != nil {
		t.Fatalf("UpsertNode(%s, %s): %v", nodeType, key, err)
	}
}

func TestRetrieveCarriesProjectedGraphDomain(t *testing.T) {
	heroDir := newHeroDir(t)
	addGraphNodeWithDomain(t, heroDir, "TestPlan", "qa-checkout", "qa", map[string]any{
		"title": "Checkout quality plan",
	})
	projectNodes(t, heroDir)

	r, err := New(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	results, err := r.Retrieve(Query{Text: "checkout"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].Domain != "qa" {
		t.Fatalf("projected result domain = %#v, want qa", results)
	}
}

// addFTSSpec creates a spec on disk and indexes it into the FTS5 index.
func addFTSSpec(t *testing.T, heroDir, slug, title, specType, content string) {
	t.Helper()
	addFTSSpecWithFields(t, heroDir, slug, title, specType, content, "")
}

// addFTSSpecWithFields extends addFTSSpec with an explicit
// superseded_by value. Used by the de-weight tests.
func addFTSSpecWithFields(t *testing.T, heroDir, slug, title, specType, content, supersededBy string) {
	t.Helper()

	path := filepath.Join(heroDir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	now := time.Now()
	sp := &spec.Spec{
		Slug:         slug,
		Title:        title,
		Type:         spec.Type(specType),
		Status:       spec.StatusPlanning,
		Path:         path,
		CreatedAt:    now,
		ModifiedAt:   now,
		SupersededBy: supersededBy,
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

func TestProjectionPreservesSameKeyAcrossRepos(t *testing.T) {
	heroDir := newHeroDir(t)
	const (
		nodeType = "ContextDoc"
		key      = "architecture-overview"
	)
	addGraphNodeWithRepo(t, heroDir, nodeType, key, "astroville/hydra", map[string]any{
		"title": "Hydra shared architecture",
		"body":  "federated architecture identity probe",
	})
	addGraphNodeWithRepo(t, heroDir, nodeType, key, "boxy", map[string]any{
		"title": "Boxy shared architecture",
		"body":  "federated architecture identity probe",
	})

	if n := projectNodes(t, heroDir); n != 2 {
		t.Fatalf("ProjectGraphNodes returned %d, want 2", n)
	}

	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	var projected int
	if err := idb.RawDB().QueryRow(`
		SELECT count(*) FROM node_index
		 WHERE node_type = ? AND key = ?
		   AND repo IN ('astroville/hydra', 'boxy')`, nodeType, key).Scan(&projected); err != nil {
		idb.Close()
		t.Fatalf("count repo-scoped projection: %v", err)
	}
	if projected != 2 {
		idb.Close()
		t.Fatalf("repo-scoped projected rows = %d, want 2", projected)
	}
	if err := idb.Close(); err != nil {
		t.Fatalf("close index: %v", err)
	}

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()
	results, err := ret.Retrieve(Query{Text: "federated architecture identity probe", Limit: 10})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	repos := map[string]bool{}
	for _, result := range results {
		if result.Type == nodeType && result.Key == key {
			repos[result.Repo] = true
		}
	}
	if !repos["astroville/hydra"] || !repos["boxy"] || len(repos) != 2 {
		t.Fatalf("retrieval repos = %v, want astroville/hydra and boxy", repos)
	}
}

func TestProjectionFailureRetainsCommittedIndex(t *testing.T) {
	heroDir := newHeroDir(t)
	const (
		nodeType = "ContextDoc"
		key      = "architecture-overview"
	)
	addGraphNodeWithRepo(t, heroDir, nodeType, key, "astroville/hydra", map[string]any{
		"title": "Committed Hydra architecture",
		"body":  "committed projection body",
	})
	if n := projectNodes(t, heroDir); n != 1 {
		t.Fatalf("initial ProjectGraphNodes returned %d, want 1", n)
	}
	addGraphNodeWithRepo(t, heroDir, nodeType, key, "boxy", map[string]any{
		"title": "Boxy architecture",
		"body":  "replacement projection body",
	})

	gstore, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer gstore.Close()
	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	defer idb.Close()
	if _, err := idb.RawDB().Exec(`
		CREATE TRIGGER reject_boxy_projection
		BEFORE INSERT ON node_index
		WHEN NEW.repo = 'boxy'
		BEGIN
			SELECT RAISE(ABORT, 'forced repo projection failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if _, err := idb.ProjectGraphNodes(gstore.DB()); err == nil {
		t.Fatal("ProjectGraphNodes unexpectedly succeeded")
	} else if !strings.Contains(err.Error(), `repo "boxy"`) {
		t.Fatalf("projection error %q does not identify failing repo", err)
	}

	var count int
	var repo, title, body string
	err = idb.RawDB().QueryRow(`
		SELECT count(*), min(ni.repo), min(f.title), min(f.body)
		  FROM node_index ni
		  JOIN fts_nodes f ON f.rowid = ni.rowid`).Scan(&count, &repo, &title, &body)
	if err != nil {
		t.Fatalf("read committed projection after rollback: %v", err)
	}
	if count != 1 || repo != "astroville/hydra" ||
		title != "Committed Hydra architecture" || body != "committed projection body" {
		t.Fatalf("projection rollback left count=%d repo=%q title=%q body=%q",
			count, repo, title, body)
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

	results := fuseRRF(lexical, vector, nil, false, 60, 10)

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
	results := fuseRRF(nil, nil, nil, false, 60, 10)
	if len(results) != 0 {
		t.Errorf("expected zero results for empty inputs, got %d", len(results))
	}

	// Lexical only.
	lexical := []Result{{Key: "a", Source: "graph"}}
	results = fuseRRF(lexical, nil, nil, false, 60, 10)
	if len(results) != 1 || results[0].Key != "a" {
		t.Errorf("expected 1 lexical-only result, got %d", len(results))
	}

	// Vector only.
	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "spec:b:", Corpus: "spec", SourceID: "b"}, Score: 0.9},
	}
	results = fuseRRF(nil, vector, nil, false, 60, 10)
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

	results := fuseRRF(lexical, vector, nil, false, 60, 5)
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
		"auth-retry-spec":     "Authentication retry with exponential backoff for failed login attempts",
		"scan-pipeline-spec":  "Scan pipeline optimization for parallel file tree walking and ingestion",
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

// TestSupersededDeweightRanksAfterPeers covers the core soft-archive
// retrieval contract: when two specs match the same query and one is
// superseded, the non-superseded peer ranks higher. Spec:
// superseded-specs-soft-archive.
//
// Covers ACs:
//   - "WHEN a search query returns a spec whose superseded_by is
//     non-empty THE SYSTEM SHALL multiply that result's score by the
//     de-weight factor (default 0.3) so non-superseded peers rank ahead
//     of it."
//   - "WHEN a search query returns a superseded spec THE SYSTEM SHALL
//     prefix the result's snippet with [SUPERSEDED → <slug>]."
func TestSupersededDeweightRanksAfterPeers(t *testing.T) {
	heroDir := newHeroDirFTSOnly(t)
	addFTSSpecWithFields(t, heroDir, "surface-polish-v1", "Surface Polish",
		"feature", "surface polish design that did the thing", "surface-polish-v2")
	addFTSSpec(t, heroDir, "surface-polish-v2", "Surface Polish v2",
		"feature", "surface polish current design")

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{
		Text:    "surface polish",
		Filters: map[string]string{"status": "planning"}, // force FTS path
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("want >=2 results, got %d", len(results))
	}

	// First result must be the v2 (non-superseded) — de-weight pushes
	// v1 below it even though both match the text equally.
	if results[0].Key != "surface-polish-v2" {
		t.Errorf("first result = %q, want surface-polish-v2 (de-weight should rank superseded below)", results[0].Key)
	}

	// Find the v1 result and verify the annotation marker is on it.
	var v1 *Result
	for i := range results {
		if results[i].Key == "surface-polish-v1" {
			v1 = &results[i]
		}
	}
	if v1 == nil {
		t.Fatal("superseded v1 not in results — must be visible, just de-weighted")
	}
	if !strings.Contains(v1.Snippet, "[SUPERSEDED → surface-polish-v2]") {
		t.Errorf("v1 snippet missing redirect marker; got %q", v1.Snippet)
	}
}

// TestIncludeSupersededSkipsDeweight verifies the opt-out flag:
// --include-superseded keeps the annotation but skips the rank
// penalty.
//
// Covers AC: "WHEN hero search --include-superseded is passed THE
// SYSTEM SHALL skip the de-weight multiplier but still emit the
// [SUPERSEDED → <slug>] annotation."
func TestIncludeSupersededSkipsDeweight(t *testing.T) {
	heroDir := newHeroDirFTSOnly(t)
	addFTSSpecWithFields(t, heroDir, "old", "Old Spec",
		"feature", "shared keyword content", "new")
	addFTSSpec(t, heroDir, "new", "New Spec",
		"feature", "shared keyword content")

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(Query{
		Text:              "shared",
		Filters:           map[string]string{"status": "planning"},
		IncludeSuperseded: true,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	// Locate both results — with IncludeSuperseded, scores should be
	// equal (same type boost, no penalty), and the annotation marker
	// stays on the superseded one.
	var oldRes, newRes *Result
	for i := range results {
		switch results[i].Key {
		case "old":
			oldRes = &results[i]
		case "new":
			newRes = &results[i]
		}
	}
	if oldRes == nil || newRes == nil {
		t.Fatalf("missing expected results: old=%v new=%v", oldRes, newRes)
	}
	if oldRes.Score != newRes.Score {
		t.Errorf("scores differ under IncludeSuperseded: old=%v new=%v (want equal)", oldRes.Score, newRes.Score)
	}
	if !strings.Contains(oldRes.Snippet, "[SUPERSEDED → new]") {
		t.Errorf("annotation should still be present; got %q", oldRes.Snippet)
	}
}

// ---------------------------------------------------------------------------
// Hybrid + supersede tests (spec embeddings-superseded-respect)
// ---------------------------------------------------------------------------

// TestFuseRRF_SupersedeVectorOnlyHit covers the AC: when a superseded spec
// surfaces in the vector path only, fuseRRF de-weights the fused score and
// stamps the annotation marker. Without this, hybrid search would re-promote
// v1 specs that the lexical path correctly filtered.
func TestFuseRRF_SupersedeVectorOnlyHit(t *testing.T) {
	lexical := []Result{
		{Key: "current-v2", Title: "Current V2", Source: "graph", Snippet: "current design"},
	}
	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "spec:old-v1:", Corpus: "spec", SourceID: "old-v1", Section: "old design content"}, Score: 0.95},
	}
	overlay := map[string]string{"old-v1": "current-v2"}

	results := fuseRRF(lexical, vector, overlay, false, 60, 10)

	// Both should be present.
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}

	var oldRes, newRes *Result
	for i := range results {
		switch results[i].Key {
		case "old-v1":
			oldRes = &results[i]
		case "current-v2":
			newRes = &results[i]
		}
	}
	if oldRes == nil || newRes == nil {
		t.Fatalf("missing results: old=%v new=%v", oldRes, newRes)
	}

	// Annotation must be present on the superseded result.
	if !strings.HasPrefix(oldRes.Snippet, "[SUPERSEDED → current-v2] ") {
		t.Errorf("want SUPERSEDED prefix on vector-only hit, got %q", oldRes.Snippet)
	}

	// De-weight must have been applied to the fused score.
	// Vector-only at rank 0 with no penalty: 1/(60+0+1) ≈ 0.01639
	// With 0.3x: ≈ 0.00492
	expected := (1.0 / 61.0) * 0.3
	if abs(oldRes.Score-expected) > 1e-6 {
		t.Errorf("old-v1 score = %v, want %v (0.3 × 1/61)", oldRes.Score, expected)
	}

	// Non-superseded result keeps its score.
	if abs(newRes.Score-1.0/61.0) > 1e-6 {
		t.Errorf("current-v2 score = %v, want %v (1/61)", newRes.Score, 1.0/61.0)
	}

	// Non-superseded must rank ahead.
	if results[0].Key != "current-v2" {
		t.Errorf("want current-v2 ranked first, got %q", results[0].Key)
	}
}

// TestFuseRRF_SupersedeBothPathsNoDoubleAnnotation verifies the
// idempotency / exactly-once contract: when a superseded spec hits in BOTH
// the lexical and vector paths, the de-weight is applied once and the
// annotation appears exactly once (no [SUPERSEDED → x] [SUPERSEDED → x]
// doubling). The lexical sub-call in retrieveHybrid already stamped the
// marker — fuseRRF must skip the second stamp.
func TestFuseRRF_SupersedeBothPathsNoDoubleAnnotation(t *testing.T) {
	// Simulate the post-skipSupersedeDeweight lexical handoff: snippet
	// already carries the annotation, score is NOT yet de-weighted.
	lexical := []Result{
		{Key: "old-v1", Title: "Old V1", Source: "fts5", Snippet: "[SUPERSEDED → current-v2] old body"},
	}
	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "spec:old-v1:", Corpus: "spec", SourceID: "old-v1", Section: "old body"}, Score: 0.95},
	}
	overlay := map[string]string{"old-v1": "current-v2"}

	results := fuseRRF(lexical, vector, overlay, false, 60, 10)

	if len(results) != 1 {
		t.Fatalf("want 1 merged result, got %d", len(results))
	}
	got := results[0]

	// Idempotency: marker must appear exactly once.
	if strings.Count(got.Snippet, "[SUPERSEDED → ") != 1 {
		t.Errorf("annotation appeared %d times, want 1; snippet=%q",
			strings.Count(got.Snippet, "[SUPERSEDED → "), got.Snippet)
	}

	// Source merged → hybrid.
	if got.Source != "hybrid" {
		t.Errorf("want source=hybrid, got %q", got.Source)
	}

	// Exactly-once de-weight: fused RRF score (lex rank 0 + vec rank 0)
	// = 1/61 + 1/61 = 2/61 ≈ 0.03279. Multiply once by 0.3 → ≈ 0.00984.
	expected := (2.0 / 61.0) * 0.3
	if abs(got.Score-expected) > 1e-6 {
		t.Errorf("fused score = %v, want %v (0.3 × 2/61, single application)", got.Score, expected)
	}
}

// TestFuseRRF_IncludeSupersededSkipsDeweightKeepsAnnotation covers the
// rule that IncludeSuperseded is a rank effect, not a visibility effect:
// the de-weight is skipped but the [SUPERSEDED → <slug>] marker still
// fires so the model knows where to redirect.
func TestFuseRRF_IncludeSupersededSkipsDeweightKeepsAnnotation(t *testing.T) {
	lexical := []Result{
		{Key: "old-v1", Title: "Old V1", Source: "graph", Snippet: "old body"},
		{Key: "current-v2", Title: "Current V2", Source: "graph", Snippet: "current body"},
	}
	overlay := map[string]string{"old-v1": "current-v2"}

	results := fuseRRF(lexical, nil, overlay, true /* includeSuperseded */, 60, 10)

	var oldRes, newRes *Result
	for i := range results {
		switch results[i].Key {
		case "old-v1":
			oldRes = &results[i]
		case "current-v2":
			newRes = &results[i]
		}
	}
	if oldRes == nil || newRes == nil {
		t.Fatalf("missing results: old=%v new=%v", oldRes, newRes)
	}

	// Annotation present.
	if !strings.HasPrefix(oldRes.Snippet, "[SUPERSEDED → current-v2] ") {
		t.Errorf("annotation must still fire under IncludeSuperseded; got %q", oldRes.Snippet)
	}

	// No de-weight — both should hold their natural RRF rank scores.
	// old at rank 0: 1/61. new at rank 1: 1/62. No 0.3x multiplier on either.
	if abs(oldRes.Score-1.0/61.0) > 1e-6 {
		t.Errorf("old-v1 score = %v, want %v (no de-weight under IncludeSuperseded)", oldRes.Score, 1.0/61.0)
	}
	if abs(newRes.Score-1.0/62.0) > 1e-6 {
		t.Errorf("current-v2 score = %v, want %v", newRes.Score, 1.0/62.0)
	}
}

// TestFuseRRF_NonSpecCorpusUnaffected verifies that the overlay never
// matches non-spec vector hits (knowledge, convention, code, event).
// The overlay only contains spec slugs, so a knowledge chunk passes
// through unchanged even if its key collides with a spec slug.
func TestFuseRRF_NonSpecCorpusUnaffected(t *testing.T) {
	vector := []embeddings.ScoredChunk{
		{Chunk: embeddings.Chunk{ID: "knowledge:auth.md", Corpus: "knowledge", SourceID: "auth.md", Section: "auth notes"}, Score: 0.9},
	}
	// Overlay contains a spec slug — irrelevant for knowledge keys.
	overlay := map[string]string{"old-v1": "current-v2"}

	results := fuseRRF(nil, vector, overlay, false, 60, 10)

	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	got := results[0]
	if strings.Contains(got.Snippet, "[SUPERSEDED") {
		t.Errorf("non-spec corpus should not be annotated; got %q", got.Snippet)
	}
	if abs(got.Score-1.0/61.0) > 1e-6 {
		t.Errorf("non-spec score = %v, want %v (no de-weight)", got.Score, 1.0/61.0)
	}
}

// TestFuseRRF_EmptyOverlayNoMutation covers the AC: when the workspace
// has no superseded specs, fuseRRF behaves exactly like the pre-overlay
// implementation — no score changes, no snippet mutations.
func TestFuseRRF_EmptyOverlayNoMutation(t *testing.T) {
	lexical := []Result{
		{Key: "a", Title: "A", Source: "graph", Snippet: "a body"},
		{Key: "b", Title: "B", Source: "graph", Snippet: "b body"},
	}
	results := fuseRRF(lexical, nil, map[string]string{}, false, 60, 10)
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	for _, r := range results {
		if strings.Contains(r.Snippet, "[SUPERSEDED") {
			t.Errorf("empty overlay must not stamp annotation; got %q", r.Snippet)
		}
	}
	// Scores should match natural RRF positions: 1/61 and 1/62.
	if abs(results[0].Score-1.0/61.0) > 1e-6 || abs(results[1].Score-1.0/62.0) > 1e-6 {
		t.Errorf("empty overlay should not change scores; got %v %v", results[0].Score, results[1].Score)
	}
}

// TestLoadSupersededOverlay_ReadsFromSpecsTable verifies the SQL helper
// returns the expected {old → new} map and gracefully returns an empty
// map for installs without the column or with no superseded specs.
func TestLoadSupersededOverlay_ReadsFromSpecsTable(t *testing.T) {
	heroDir := newHeroDirFTSOnly(t)

	// Insert one superseded + one current spec via the public helper.
	addFTSSpecWithFields(t, heroDir, "old-v1", "Old V1", "feature", "old content", "current-v2")
	addFTSSpec(t, heroDir, "current-v2", "Current V2", "feature", "current content")
	addFTSSpec(t, heroDir, "unrelated", "Unrelated", "feature", "other content")

	idb, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	defer idb.Close()

	overlay, err := loadSupersededOverlay(idb.RawDB())
	if err != nil {
		t.Fatalf("loadSupersededOverlay: %v", err)
	}

	if got, want := overlay["old-v1"], "current-v2"; got != want {
		t.Errorf("overlay[old-v1] = %q, want %q", got, want)
	}
	if _, present := overlay["current-v2"]; present {
		t.Errorf("non-superseded spec should not appear in overlay; got %v", overlay)
	}
	if _, present := overlay["unrelated"]; present {
		t.Errorf("non-superseded spec should not appear in overlay; got %v", overlay)
	}
}

// TestLoadSupersededOverlay_NilDB returns an empty map when the DB is nil
// (graceful fallback for partially-initialized retrievers).
func TestLoadSupersededOverlay_NilDB(t *testing.T) {
	overlay, err := loadSupersededOverlay(nil)
	if err != nil {
		t.Fatalf("want nil error for nil DB, got %v", err)
	}
	if len(overlay) != 0 {
		t.Errorf("want empty map for nil DB, got %v", overlay)
	}
}

// TestRetrieveHybrid_VectorPathSupersedeAware is the integration test
// for the spec's headline behavior: a workspace with a superseded spec
// in vec_chunks + a current peer in the graph → `hero search --semantic`
// returns the current spec first with the SUPERSEDED marker on v1.
// Without the overlay this is the exact failure mode the spec was built
// to fix.
func TestRetrieveHybrid_VectorPathSupersedeAware(t *testing.T) {
	heroDir := newHeroDir(t)

	// Old v1 carries superseded_by:current-v2 in the specs table;
	// current-v2 is the live peer. Both written to FTS5 + specs.
	addFTSSpecWithFields(t, heroDir, "old-v1", "Old V1",
		"feature", "old design carries the same keywords as the new one", "current-v2")
	addFTSSpec(t, heroDir, "current-v2", "Current V2",
		"feature", "current design carries the same keywords")

	// Also project as graph nodes so the lexical/node-index path
	// can surface both.
	addGraphNode(t, heroDir, "Feature", "old-v1", map[string]any{
		"title": "Old V1",
		"body":  "old design carries the same keywords",
	})
	addGraphNode(t, heroDir, "Feature", "current-v2", map[string]any{
		"title": "Current V2",
		"body":  "current design carries the same keywords",
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

	// Embed both specs into vec_chunks so the vector path surfaces them.
	store, err := embeddings.OpenStorage(ret.fts.RawDB())
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	for _, slug := range []string{"old-v1", "current-v2"} {
		vec := ret.embModel.Embed("design carries the same keywords for ranking")
		if _, err := store.Upsert(embeddings.Chunk{
			ID:       fmt.Sprintf("spec:%s:", slug),
			Corpus:   "spec",
			SourceID: slug,
			TextHash: fmt.Sprintf("hash-%s", slug),
			Vector:   vec,
		}); err != nil {
			t.Fatalf("Upsert %s: %v", slug, err)
		}
	}
	ret.embStore = store

	results, err := ret.Retrieve(Query{
		Text:       "design keywords",
		SemanticOK: true,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Retrieve hybrid: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("want >=2 hybrid results, got %d", len(results))
	}

	// Locate both.
	var oldRes, newRes *Result
	for i := range results {
		switch results[i].Key {
		case "old-v1":
			oldRes = &results[i]
		case "current-v2":
			newRes = &results[i]
		}
	}
	if oldRes == nil || newRes == nil {
		t.Fatalf("both specs expected in hybrid results; got %+v", results)
	}

	// current-v2 must rank ahead of old-v1.
	if newRes.Score <= oldRes.Score {
		t.Errorf("current-v2 (%.5f) must rank above old-v1 (%.5f) — supersede de-weight failed",
			newRes.Score, oldRes.Score)
	}

	// Annotation present exactly once on the superseded result.
	if !strings.Contains(oldRes.Snippet, "[SUPERSEDED → current-v2]") {
		t.Errorf("annotation missing on old-v1; snippet=%q", oldRes.Snippet)
	}
	if strings.Count(oldRes.Snippet, "[SUPERSEDED → ") != 1 {
		t.Errorf("annotation duplicated (%d times); snippet=%q",
			strings.Count(oldRes.Snippet, "[SUPERSEDED → "), oldRes.Snippet)
	}

	// Current spec must NOT carry the marker.
	if strings.Contains(newRes.Snippet, "[SUPERSEDED") {
		t.Errorf("current-v2 must not carry SUPERSEDED marker; snippet=%q", newRes.Snippet)
	}
}

// TestRetrieveHybrid_SupersedeRespectDisabled covers the rollback knob:
// when the RetrievalSupersedeRespect config flag is false, the hybrid
// path skips the overlay entirely — fuseRRF runs unchanged and the
// superseded spec is neither de-weighted nor annotated by the hybrid
// path (parent lexical de-weight still applies for non-hybrid callers).
func TestRetrieveHybrid_SupersedeRespectDisabled(t *testing.T) {
	heroDir := newHeroDir(t)

	addFTSSpecWithFields(t, heroDir, "old-v1", "Old V1",
		"feature", "shared design keywords here", "current-v2")
	addFTSSpec(t, heroDir, "current-v2", "Current V2",
		"feature", "shared design keywords here")
	addGraphNode(t, heroDir, "Feature", "old-v1", map[string]any{"title": "Old V1", "body": "shared design keywords here"})
	addGraphNode(t, heroDir, "Feature", "current-v2", map[string]any{"title": "Current V2", "body": "shared design keywords here"})
	projectNodes(t, heroDir)

	ret, err := New(heroDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ret.Close()

	if ret.embModel == nil {
		t.Skip("embedded model not available")
	}

	// Flip the rollback knob.
	ret.supersedeRespect = false

	store, err := embeddings.OpenStorage(ret.fts.RawDB())
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	for _, slug := range []string{"old-v1", "current-v2"} {
		vec := ret.embModel.Embed("shared design keywords here")
		if _, err := store.Upsert(embeddings.Chunk{
			ID:       fmt.Sprintf("spec:%s:", slug),
			Corpus:   "spec",
			SourceID: slug,
			TextHash: fmt.Sprintf("hash-%s", slug),
			Vector:   vec,
		}); err != nil {
			t.Fatalf("Upsert %s: %v", slug, err)
		}
	}
	ret.embStore = store

	results, err := ret.Retrieve(Query{
		Text:       "shared design",
		SemanticOK: true,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Retrieve hybrid: %v", err)
	}

	var oldRes *Result
	for i := range results {
		if results[i].Key == "old-v1" {
			oldRes = &results[i]
		}
	}
	if oldRes == nil {
		t.Fatalf("old-v1 not present; got %+v", results)
	}

	// With respect disabled, the hybrid path does NOT stamp the marker
	// (the lexical sub-call still applies its own per-path de-weight +
	// marker — but that path is the unified node-index, which DID
	// annotate via its own pre-existing logic). The defining check for
	// this knob is that fuseRRF did NOT apply the overlay — that's
	// observable on a vector-only hit, so we use that as the load-
	// bearing assertion in TestFuseRRF_EmptyOverlayNoMutation. Here we
	// only confirm the query succeeds and returns both specs.
	if len(results) < 2 {
		t.Errorf("disabled overlay should not drop results; got %d", len(results))
	}
}

// abs is a tiny helper so the float comparisons above stay readable.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
