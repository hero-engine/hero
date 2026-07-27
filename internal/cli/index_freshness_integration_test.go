package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/retrieval"
)

func TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge(t *testing.T) {
	repo := t.TempDir()
	binDir := t.TempDir()
	heroBin := filepath.Join(binDir, "hero")
	build := exec.Command("go", "build", "-o", heroBin, "../../cmd/hero")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build hero CLI: %v\n%s", err, out)
	}

	gitFreshness(t, repo, nil, "init", "-q", "-b", "main")
	gitFreshness(t, repo, nil, "config", "user.name", "Hook Test")
	gitFreshness(t, repo, nil, "config", "user.email", "hook-test@example.com")

	cfg := config.DefaultConfig()
	cfg.CodeScan.Parser = "heuristic"
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{
		Enabled: &enabled,
		Scope:   []string{"code"},
	}
	if err := cfg.Save(repo); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(repo, "main.go")
	writeHookSource(t, source, "InitialHookSymbol")
	runHeroBinary(t, repo, heroBin, "scan", "--code")
	if err := installNextHooksQuiet(repo); err != nil {
		t.Fatalf("install managed hooks: %v", err)
	}
	hookEnv := []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH")}
	gitFreshness(t, repo, hookEnv, "add", "main.go", ".hero/hero.json")
	gitFreshness(t, repo, hookEnv, "commit", "-m", "initial")
	assertHookIndexes(t, repo, "InitialHookSymbol", true)

	writeHookSource(t, source, "InitialHookSymbol", "CommitAddedSymbol")
	gitFreshness(t, repo, hookEnv, "add", "main.go")
	gitFreshness(t, repo, hookEnv, "commit", "-m", "commit add")
	assertHookIndexes(t, repo, "CommitAddedSymbol", true)

	writeHookSource(t, source, "InitialHookSymbol", "CommitChangedSymbol")
	gitFreshness(t, repo, hookEnv, "add", "main.go")
	gitFreshness(t, repo, hookEnv, "commit", "-m", "commit change")
	assertHookIndexes(t, repo, "CommitAddedSymbol", false)
	assertHookIndexes(t, repo, "CommitChangedSymbol", true)

	writeHookSource(t, source, "InitialHookSymbol")
	gitFreshness(t, repo, hookEnv, "add", "main.go")
	gitFreshness(t, repo, hookEnv, "commit", "-m", "commit delete")
	assertHookIndexes(t, repo, "CommitChangedSymbol", false)

	mergeLifecycle(t, repo, hookEnv, source, "merge-add", []string{"MergeAddedSymbol"})
	assertHookIndexes(t, repo, "MergeAddedSymbol", true)
	mergeLifecycle(t, repo, hookEnv, source, "merge-change", []string{"MergeChangedSymbol"})
	assertHookIndexes(t, repo, "MergeAddedSymbol", false)
	assertHookIndexes(t, repo, "MergeChangedSymbol", true)
	mergeLifecycle(t, repo, hookEnv, source, "merge-delete", nil)
	assertHookIndexes(t, repo, "MergeChangedSymbol", false)

	writeHookSource(t, source, "LockedHookSymbol")
	lock, busy, err := acquireCodeRefreshLock(filepath.Join(repo, ".hero"))
	if err != nil || busy {
		t.Fatalf("hold refresh lock: busy=%v err=%v", busy, err)
	}
	gitFreshness(t, repo, hookEnv, "add", "main.go")
	gitFreshness(t, repo, hookEnv, "commit", "-m", "locked refresh remains nonblocking")
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	assertHookIndexes(t, repo, "LockedHookSymbol", false)
	check := runHeroBinary(t, repo, heroBin, "check")
	if !strings.Contains(check, "Code index freshness:") ||
		!strings.Contains(check, "missing=1") {
		t.Fatalf("check did not expose the lock-skipped source:\n%s", check)
	}
}

func mergeLifecycle(t *testing.T, repo string, env []string, source, branch string, symbols []string) {
	t.Helper()
	gitFreshness(t, repo, env, "checkout", "-q", "-b", branch)
	if len(symbols) == 0 {
		if err := os.Remove(source); err != nil {
			t.Fatal(err)
		}
		gitFreshness(t, repo, env, "add", "-u", "main.go")
	} else {
		writeHookSource(t, source, symbols...)
		gitFreshness(t, repo, env, "add", "main.go")
	}
	gitFreshness(t, repo, env, "commit", "--no-verify", "-m", branch)
	gitFreshness(t, repo, env, "checkout", "-q", "main")
	gitFreshness(t, repo, env, "merge", "--ff-only", branch)
}

func writeHookSource(t *testing.T, path string, symbols ...string) {
	t.Helper()
	var body strings.Builder
	body.WriteString("package sample\n")
	for _, symbol := range symbols {
		body.WriteString("func " + symbol + "() {}\n")
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitFreshness(t *testing.T, repo string, extraEnv []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runHeroBinary(t *testing.T, repo, heroBin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(heroBin, args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hero %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertHookIndexes(t *testing.T, repo, symbol string, want bool) {
	t.Helper()
	heroDir := filepath.Join(repo, ".hero")
	gstore, err := graph.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	var graphCount int
	if err := gstore.DB().QueryRow(`
		SELECT COUNT(*) FROM nodes
		WHERE valid_to IS NULL AND type = 'Symbol' AND key LIKE ?
	`, "%"+symbol+"%").Scan(&graphCount); err != nil {
		gstore.Close()
		t.Fatal(err)
	}
	gstore.Close()
	assertHookPresence(t, "graph", graphCount > 0, want, symbol)

	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	var lexicalCount int
	if err := idx.RawDB().QueryRow(`
		SELECT COUNT(*)
		FROM fts_nodes
		JOIN node_index ON node_index.rowid = fts_nodes.rowid
		WHERE fts_nodes MATCH ? AND node_index.node_type = 'Symbol'
	`, symbol).Scan(&lexicalCount); err != nil {
		t.Fatal(err)
	}
	assertHookPresence(t, "FTS", lexicalCount > 0, want, symbol)

	model, err := embeddings.LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatal(err)
	}
	store, err := embeddings.OpenStorage(idx.RawDB())
	if err != nil {
		t.Fatal(err)
	}
	vectorHits, err := store.QuerySimilar(model.Embed(symbol), 100, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	vectorFound := false
	for _, hit := range vectorHits {
		if strings.Contains(hit.SourceID, symbol) {
			vectorFound = true
			break
		}
	}
	assertHookPresence(t, "vector", vectorFound, want, symbol)

	ret, err := retrieval.New(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ret.Close()
	results, err := ret.Retrieve(retrieval.Query{Text: symbol, SemanticOK: true, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	hybridFound := false
	var hybridMatch retrieval.Result
	for _, result := range results {
		if strings.Contains(result.Key, symbol) ||
			strings.Contains(result.Title, symbol) {
			hybridFound = true
			hybridMatch = result
			break
		}
	}
	if hybridFound != want {
		t.Logf("hybrid match for %s: %+v", symbol, hybridMatch)
	}
	assertHookPresence(t, "hybrid", hybridFound, want, symbol)
}

func assertHookPresence(t *testing.T, surface string, got, want bool, symbol string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s presence for %s = %v, want %v", surface, symbol, got, want)
	}
}
