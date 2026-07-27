package cli

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/index"
	_ "modernc.org/sqlite"
)

func TestIncrementalCodeRefreshEmbeddingsConfiguredScopeUnchangedAndDelete(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc Existing() {}\nfunc Removed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "knowledge", "note.md"), []byte("embedding coverage"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{
		Enabled: &enabled,
		Scope:   []string{"code", "knowledge"},
	}
	cfg.CodeScan.Parser = "heuristic"

	first, err := refreshCodeIndex(context.Background(), cfg, root, heroDir, codeRefreshOptions{Parser: "heuristic"})
	if err != nil {
		t.Fatalf("bootstrap refresh: %v", err)
	}
	if first.Embeddings.Stats == nil ||
		first.Embeddings.Stats.Corpora["code"].Added != 2 ||
		first.Embeddings.Stats.Corpora["knowledge"].Added != 1 {
		t.Fatalf("configured bootstrap embeddings = %+v", first.Embeddings)
	}

	unchanged, err := refreshCodeIndex(context.Background(), cfg, root, heroDir, codeRefreshOptions{
		Incremental: true, Quiet: true, Parser: "heuristic",
	})
	if err != nil {
		t.Fatalf("unchanged refresh: %v", err)
	}
	if unchanged.Changed || !unchanged.PostStructureReady || unchanged.Embeddings.Stats == nil {
		t.Fatalf("unchanged refresh stats = %+v", unchanged)
	}
	codeStats := unchanged.Embeddings.Stats.Corpora["code"]
	if codeStats.Added != 0 || codeStats.Updated != 0 || codeStats.Pruned != 0 || codeStats.Skipped != 2 {
		t.Fatalf("unchanged code embeddings = %+v", codeStats)
	}

	if err := os.WriteFile(source, []byte("package main\nfunc Existing() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deleted, err := refreshCodeIndex(context.Background(), cfg, root, heroDir, codeRefreshOptions{
		Incremental: true, Quiet: true, Parser: "heuristic",
	})
	if err != nil {
		t.Fatalf("delete refresh: %v", err)
	}
	if deleted.Embeddings.Stats.Corpora["code"].Pruned != 1 {
		t.Fatalf("deleted code embeddings = %+v", deleted.Embeddings.Stats.Corpora["code"])
	}
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	var count int
	if err := idx.RawDB().QueryRow(`SELECT COUNT(*) FROM vec_chunks WHERE corpus = 'code'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("code chunk count after delete = %d, want 1", count)
	}
}

func TestEmbeddingsRefreshCLIQuietDeadlineIsSilentNonBlockingAndPreservesCorpus(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/quiet/spec.md", `---
title: Quiet
slug: quiet
type: feature
status: planning
---
## Goal
Stay silent.
`)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{Enabled: &enabled, Scope: []string{"spec"}}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}

	idx, err := index.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	model, err := embeddings.LoadModelFromConfig(cfg.EmbeddingsModel())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := embeddings.Refresh(env.heroDir, model, idx.RawDB(), nil, []string{"spec"}); err != nil {
		t.Fatal(err)
	}
	store, err := embeddings.OpenStorage(idx.RawDB())
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(env.heroDir, index.IndexFileName)
	locker, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer locker.Close()
	conn, err := locker.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")

	stdout, stderr, cmdErr := executeCapturingOutput(
		"embeddings", "refresh", "--if-stale", "--deadline", "50ms", "-q",
	)
	if cmdErr != nil {
		t.Fatalf("quiet refresh must be non-blocking: %v", cmdErr)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("quiet refresh output stdout=%q stderr=%q", stdout, stderr)
	}
	if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}

	checkDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer checkDB.Close()
	afterStore, err := embeddings.OpenStorage(checkDB)
	if err != nil {
		t.Fatal(err)
	}
	after, err := afterStore.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if after.ByCorpus["spec"] != before.ByCorpus["spec"] ||
		after.Corpora["spec"].SuccessfulExtractionAt != before.Corpora["spec"].SuccessfulExtractionAt {
		t.Fatalf("deadline changed prior corpus: before=%+v after=%+v", before, after)
	}
}

func executeCapturingOutput(args ...string) (stdout, stderr string, err error) {
	mu.Lock()
	defer mu.Unlock()

	oldStdout, oldStderr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW

	embeddingsRefreshDeadline = defaultIncrementalScanDeadline
	embeddingsRefreshQuiet = false
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	rootCmd.SetArgs(nil)
	embeddingsRefreshDeadline = defaultIncrementalScanDeadline
	embeddingsRefreshQuiet = false

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	var outBuf, errBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, outR)
	_, _ = io.Copy(&errBuf, errR)
	return outBuf.String(), errBuf.String(), err
}

func TestEmbeddingPhaseUnavailableModelIsTruthful(t *testing.T) {
	cfg := config.DefaultConfig()
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{
		Enabled: &enabled,
		Model:   "model-that-does-not-exist",
		Scope:   []string{"spec"},
	}
	outcome, err := runEmbeddingPhase(context.Background(), cfg, t.TempDir(), nil, nil)
	if err == nil || !outcome.Unavailable || outcome.Skipped || outcome.Reason == "" {
		t.Fatalf("unavailable model outcome=%+v err=%v", outcome, err)
	}
}

func TestEmbeddingRefreshDeadlineFlagRejectsNonPositive(t *testing.T) {
	env := newTestEnv(t)
	_ = env
	start := time.Now()
	_, _, err := executeCapturingOutput("embeddings", "refresh", "--deadline", "0s", "-q")
	if err != nil {
		t.Fatalf("quiet invalid deadline must normalize failure: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("invalid deadline did not fail promptly")
	}
}
