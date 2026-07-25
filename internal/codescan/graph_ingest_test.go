package codescan

import (
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
