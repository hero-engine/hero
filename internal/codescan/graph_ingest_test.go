package codescan

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func newTestStore(t *testing.T) *graph.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := graph.Open(filepath.Join(dir, "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// sampleResult builds a small scan result mirroring the shape of a real
// hero scan: one repo, two packages, with cli importing config.
func sampleResult() *Result {
	return &Result{
		ProjectRoot: "/tmp/example-repo",
		Complete:    true,
		Packages: []Package{
			{
				Name: "cli", Path: "internal/cli", Language: "go",
				Files:     []string{"internal/cli/root.go", "internal/cli/run.go"},
				LineCount: 200, FileCount: 2,
				Symbols: []Symbol{
					{Name: "Execute", Kind: SymFunc, Exported: true, Signature: "func() error", File: "internal/cli/root.go", Line: 24},
					{Name: "SetVersion", Kind: SymFunc, Exported: true, Signature: "func(v string)", File: "internal/cli/root.go", Line: 19},
				},
			},
			{
				Name: "config", Path: "internal/config", Language: "go",
				Files:     []string{"internal/config/config.go"},
				LineCount: 120, FileCount: 1,
				Symbols: []Symbol{
					{Name: "Load", Kind: SymFunc, Exported: true, Signature: "func(string) (*Config, error)", File: "internal/config/config.go", Line: 42},
				},
			},
		},
		DepGraph: []DepEdge{
			{From: "internal/cli", To: "internal/config"},
		},
	}
}

func TestWriteGraphPopulatesAllNodeAndEdgeTypes(t *testing.T) {
	store := newTestStore(t)
	summary, err := WriteGraph(sampleResult(), store, "engineering")
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}

	if summary.Repos != 1 {
		t.Errorf("Repos = %d, want 1", summary.Repos)
	}
	if summary.Packages != 2 {
		t.Errorf("Packages = %d, want 2", summary.Packages)
	}
	if summary.Files != 3 {
		t.Errorf("Files = %d, want 3", summary.Files)
	}
	if summary.Symbols != 3 {
		t.Errorf("Symbols = %d, want 3", summary.Symbols)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.NodesByType["Repo"] != 1 {
		t.Errorf("Repo nodes = %d, want 1", stats.NodesByType["Repo"])
	}
	if stats.NodesByType["Package"] != 2 {
		t.Errorf("Package nodes = %d, want 2", stats.NodesByType["Package"])
	}
	if stats.NodesByType["File"] != 3 {
		t.Errorf("File nodes = %d, want 3", stats.NodesByType["File"])
	}
	if stats.NodesByType["Symbol"] != 3 {
		t.Errorf("Symbol nodes = %d, want 3", stats.NodesByType["Symbol"])
	}
	if stats.EdgesByType["imports"] != 1 {
		t.Errorf("imports edges = %d, want 1", stats.EdgesByType["imports"])
	}
	if stats.EdgesByType["defines"] != 3 {
		t.Errorf("defines edges = %d, want 3", stats.EdgesByType["defines"])
	}
	// belongs_to: package→repo (2) + file→package (3) + symbol→package (3) = 8
	if stats.EdgesByType["belongs_to"] != 8 {
		t.Errorf("belongs_to edges = %d, want 8", stats.EdgesByType["belongs_to"])
	}
}

func TestWriteGraphIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("first WriteGraph: %v", err)
	}
	beforeStats, _ := store.Stats()

	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("second WriteGraph: %v", err)
	}
	afterStats, _ := store.Stats()

	if beforeStats.HistoryRows.Nodes != afterStats.HistoryRows.Nodes {
		t.Errorf("history nodes grew on idempotent re-ingest: %d → %d",
			beforeStats.HistoryRows.Nodes, afterStats.HistoryRows.Nodes)
	}
	if beforeStats.HistoryRows.Edges != afterStats.HistoryRows.Edges {
		t.Errorf("history edges grew on idempotent re-ingest: %d → %d",
			beforeStats.HistoryRows.Edges, afterStats.HistoryRows.Edges)
	}
}

func TestWriteGraphChangedPackageInvalidatesPriorRow(t *testing.T) {
	store := newTestStore(t)
	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("first WriteGraph: %v", err)
	}

	// Modify cli package: add a symbol
	r := sampleResult()
	r.Packages[0].Symbols = append(r.Packages[0].Symbols,
		Symbol{Name: "NewThing", Kind: SymFunc, Exported: true, File: "internal/cli/root.go", Line: 200})

	if _, err := WriteGraph(r, store, "engineering"); err != nil {
		t.Fatalf("second WriteGraph: %v", err)
	}

	stats, _ := store.Stats()
	// Current Package nodes still 2 — the cli row was invalidated and replaced
	if stats.NodesByType["Package"] != 2 {
		t.Errorf("current Package nodes = %d, want 2", stats.NodesByType["Package"])
	}
	// History grew by 1 (cli replaced)
	if stats.HistoryRows.Nodes < 9 {
		t.Errorf("history nodes = %d, want at least 9 after update", stats.HistoryRows.Nodes)
	}
}

func TestImportsEdgeResolvesPackageIDs(t *testing.T) {
	store := newTestStore(t)
	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	cliID, err := store.GetNodeID("Package", "example-repo:internal/cli", "")
	if err != nil {
		t.Fatalf("GetNodeID cli: %v", err)
	}
	cfgID, err := store.GetNodeID("Package", "example-repo:internal/config", "")
	if err != nil {
		t.Fatalf("GetNodeID config: %v", err)
	}
	imps, err := store.EdgesFrom(cliID, "imports")
	if err != nil {
		t.Fatalf("EdgesFrom: %v", err)
	}
	if len(imps) != 1 {
		t.Fatalf("imports edges from cli = %d, want 1", len(imps))
	}
	if imps[0].ToID != cfgID {
		t.Errorf("imports edge points to %d, want config %d", imps[0].ToID, cfgID)
	}
}

func TestWriteGraphRetiresDeletedCodeNodesAndIncidentEdges(t *testing.T) {
	store := newTestStore(t)
	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("initial WriteGraph: %v", err)
	}

	result := sampleResult()
	result.Packages = result.Packages[:1]
	result.Packages[0].Files = result.Packages[0].Files[:1]
	result.Packages[0].FileCount = 1
	result.Packages[0].Symbols = result.Packages[0].Symbols[:1]
	result.DepGraph = nil
	summary, err := WriteGraph(result, store, "engineering")
	if err != nil {
		t.Fatalf("reconcile WriteGraph: %v", err)
	}
	if summary.RetiredPackages != 1 || summary.RetiredFiles != 2 || summary.RetiredSymbols != 2 {
		t.Fatalf("retired summary = %+v, want package=1 files=2 symbols=2", summary)
	}
	if summary.RetiredEdges == 0 {
		t.Fatalf("retired summary = %+v, want incident edges retired", summary)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.NodesByType["Package"] != 1 || stats.NodesByType["File"] != 1 || stats.NodesByType["Symbol"] != 1 {
		t.Fatalf("current nodes = %+v, want one package/file/symbol", stats.NodesByType)
	}
	if stats.EdgesByType["imports"] != 0 || stats.EdgesByType["defines"] != 1 {
		t.Fatalf("current edges = %+v, deleted-node edges survived", stats.EdgesByType)
	}

	empty := &Result{ProjectRoot: result.ProjectRoot, Complete: true}
	emptySummary, err := WriteGraph(empty, store, "engineering")
	if err != nil {
		t.Fatalf("empty-tree WriteGraph: %v", err)
	}
	if emptySummary.RetiredPackages != 1 || emptySummary.RetiredFiles != 1 || emptySummary.RetiredSymbols != 1 {
		t.Fatalf("empty-tree retirement = %+v", emptySummary)
	}
	if _, err := store.GetNode("Repo", "example-repo", "example-repo"); err != nil {
		t.Fatalf("Repo must survive authoritative empty tree: %v", err)
	}

	retry, err := WriteGraph(empty, store, "engineering")
	if err != nil {
		t.Fatalf("idempotent empty-tree retry: %v", err)
	}
	if retry.RetiredPackages+retry.RetiredFiles+retry.RetiredSymbols+retry.RetiredEdges != 0 {
		t.Fatalf("idempotent retry retired more rows: %+v", retry)
	}
}

func TestWriteGraphRetirementIsolatesRepoAndSourceKind(t *testing.T) {
	store := newTestStore(t)
	localRepo := repoKeyFor("/tmp/example-repo")
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Package", Domain: "engineering", Key: localRepo + ":manual",
		Repo: localRepo, Props: map[string]any{"name": "manual"},
		Source: map[string]any{"kind": "manual"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Package", Domain: "engineering", Key: "sibling:owned",
		Repo: "sibling", Props: map[string]any{"name": "sibling"},
		Source: map[string]any{"kind": "codescan"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteGraph(&Result{ProjectRoot: "/tmp/example-repo", Complete: true}, store, "engineering"); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if _, err := store.GetNode("Package", localRepo+":manual", localRepo); err != nil {
		t.Fatalf("non-codescan local node was retired: %v", err)
	}
	if _, err := store.GetNode("Package", "sibling:owned", "sibling"); err != nil {
		t.Fatalf("sibling codescan node was retired: %v", err)
	}
}

func TestWriteGraphImportOnlyChangeRebuildsImportsEdge(t *testing.T) {
	store := newTestStore(t)
	first := sampleResult()
	first.Packages[0].Imports = []string{"internal/config"}
	if _, err := WriteGraph(first, store, "engineering"); err != nil {
		t.Fatalf("first WriteGraph: %v", err)
	}
	oldCLI, err := store.GetNodeID("Package", "example-repo:internal/cli", "example-repo")
	if err != nil {
		t.Fatal(err)
	}

	second := sampleResult()
	second.Packages = append(second.Packages, Package{
		Name: "other", Path: "internal/other", Language: "go",
		Files: []string{"internal/other/other.go"}, FileCount: 1,
	})
	second.Packages[0].Imports = []string{"internal/other"}
	second.DepGraph = []DepEdge{{From: "internal/cli", To: "internal/other"}}
	if _, err := WriteGraph(second, store, "engineering"); err != nil {
		t.Fatalf("second WriteGraph: %v", err)
	}
	newCLI, err := store.GetNodeID("Package", "example-repo:internal/cli", "example-repo")
	if err != nil {
		t.Fatal(err)
	}
	if newCLI == oldCLI {
		t.Fatal("import-only package change did not invalidate the package node")
	}
	edges, err := store.EdgesFrom(newCLI, "imports")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("current imports edges = %d, want 1", len(edges))
	}
	otherID, err := store.GetNodeID("Package", "example-repo:internal/other", "example-repo")
	if err != nil {
		t.Fatal(err)
	}
	if edges[0].ToID != otherID {
		t.Fatalf("imports target = %d, want %d", edges[0].ToID, otherID)
	}
}

func TestWriteGraphRollsBackUpsertsWhenReconciliationFails(t *testing.T) {
	store := newTestStore(t)
	repo := repoKeyFor("/tmp/example-repo")
	_, err := store.DB().Exec(`
		INSERT INTO nodes
			(type, key, props, scope, repo, unit, domain, content_hash, source, valid_from, valid_to, ingested_at)
		VALUES ('Package', ?, '{}', 'team', ?, '', 'engineering', NULL, '{', 'now', NULL, 'now')`,
		repo+":malformed", repo)
	if err != nil {
		t.Fatal(err)
	}
	result := &Result{
		ProjectRoot: "/tmp/example-repo",
		Complete:    true,
		Packages: []Package{{
			Name: "new", Path: "new", Language: "go",
			Files: []string{"new/new.go"}, FileCount: 1,
		}},
	}
	if _, err := WriteGraphContext(context.Background(), result, store, "engineering"); err == nil {
		t.Fatal("expected malformed source JSON to fail reconciliation")
	}
	if _, err := store.GetNode("Package", repo+":new", repo); err != graph.ErrNotFound {
		t.Fatalf("upsert survived failed aggregate transaction: %v", err)
	}
}

func TestWriteGraphRejectsIncompleteResultWithoutRetirement(t *testing.T) {
	store := newTestStore(t)
	if _, err := WriteGraph(sampleResult(), store, "engineering"); err != nil {
		t.Fatalf("initial WriteGraph: %v", err)
	}
	before, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := WriteGraph(&Result{
		ProjectRoot: "/tmp/example-repo",
		Complete:    false,
	}, store, "engineering"); err == nil {
		t.Fatal("expected incomplete Result to be rejected")
	}

	after, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if after.NodesByType["Package"] != before.NodesByType["Package"] ||
		after.NodesByType["File"] != before.NodesByType["File"] ||
		after.NodesByType["Symbol"] != before.NodesByType["Symbol"] {
		t.Fatalf("incomplete Result retired nodes: before=%+v after=%+v",
			before.NodesByType, after.NodesByType)
	}
	if after.HistoryRows.Nodes != before.HistoryRows.Nodes ||
		after.HistoryRows.Edges != before.HistoryRows.Edges {
		t.Fatalf("incomplete Result mutated graph history: before=%+v after=%+v",
			before.HistoryRows, after.HistoryRows)
	}
}
