package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/index"
)

func TestCheckEmpty(t *testing.T) {
	env := newTestEnv(t)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	cfg.Embeddings = &config.EmbeddingsConfig{Enabled: &disabled}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}

	if !strings.Contains(output, "Hero workspace health check") {
		t.Errorf("check should show header: %q", output)
	}

	if !strings.Contains(output, "No issues found") {
		t.Errorf("check should report no issues for empty workspace: %q", output)
	}
}

func TestCheckReportsUnavailableConfiguredEmbeddingSources(t *testing.T) {
	env := newTestEnv(t)
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if !strings.Contains(output, "Embeddings freshness:") ||
		!strings.Contains(output, "code unavailable=") ||
		!strings.Contains(output, "event unavailable=") {
		t.Fatalf("check must truthfully report configured graph corpora unavailable:\n%s", output)
	}
}

func TestCheck_WikilinkEdgeWarning(t *testing.T) {
	env := newTestEnv(t)
	content := `---
title: Uses Wikilinks
type: feature
status: planning
slug: uses-wikilinks
---
# Uses Wikilinks

This depends on [[config-loader]] and relates to [[watcher]].

## Kickoff

Pick up here.
`
	env.addSpec("planning/features/uses-wikilinks/spec.md", content)
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check errored: %v", err)
	}
	if !strings.Contains(output, "[[wikilinks]]") {
		t.Errorf("check should warn about wikilinks: %q", output)
	}
	if !strings.Contains(output, "config-loader") {
		t.Errorf("check should name the wikilink target: %q", output)
	}
}

func TestCheckWithSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-one/spec.md", `---
title: Feature One
type: feature
status: planning
---
# Feature One
`)

	env.addSpec("conventions/conv-one/spec.md", `---
title: Convention One
type: convention
status: active
---
# Convention One
`)

	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}

	if !strings.Contains(output, "Corpus: 2 specs total") {
		t.Errorf("check should show corpus size: %q", output)
	}
}

func TestCheckNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("check")
	if err == nil {
		t.Fatal("check should error without workspace")
	}
}

func TestCheckFlagsStalePreCommitHook(t *testing.T) {
	env := newTestEnv(t)
	// Initialize the test workspace as a git repo and install a stale
	// managed block so the check can detect the drift.
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	stale := "#!/usr/bin/env bash\n" + hookMarkerStart + "\n# stale\n" + hookMarkerEnd + "\n"
	if err := os.WriteFile(hookPath, []byte(stale), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, err := runCmd("check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, "Pre-commit hook is stale") {
		t.Errorf("expected stale-hook warning, got:\n%s", out)
	}
	if strings.Contains(out, "Pre-commit hook not installed") {
		t.Errorf("should not say 'not installed' when block is present:\n%s", out)
	}
}

// TestCheckFlagsHookWithoutStaging is Test Plan #7 — a repo with a Hero
// pre-commit hook (generic `hero hook` dispatch) but NO handoff-file
// staging block must be flagged distinctly: the warning names the
// missing staging invariant and the single fix command.
// Spec: next-unconditional-commit-staging.
func TestCheckFlagsHookWithoutStaging(t *testing.T) {
	env := newTestEnv(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// A generic Hero pre-commit hook: has the `# Hero git hook` marker
	// but NOT the hero-next staging managed block.
	generic := "#!/bin/sh\n# Hero git hook — pre-commit\nhero hook pre-commit \"$@\"\n"
	if err := os.WriteFile(hookPath, []byte(generic), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, err := runCmd("check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, "handoff files are not staged") {
		t.Errorf("expected staging-invariant warning, got:\n%s", out)
	}
	if !strings.Contains(out, "hero next install-hooks") {
		t.Errorf("expected single fix command in warning, got:\n%s", out)
	}
	if strings.Contains(out, "Pre-commit hook not installed") {
		t.Errorf("should not say 'not installed' when a Hero hook is present:\n%s", out)
	}
}

func TestCheckFlagsMissingKickoff(t *testing.T) {
	env := newTestEnv(t)

	// One open feature without `## Kickoff`.
	env.addSpec("planning/features/needs-kickoff/spec.md", `---
title: Needs Kickoff
type: feature
status: planning
---
# Needs Kickoff

## Goal
nothing else.
`)

	// One with a kickoff — should not appear in the missing list.
	env.addSpec("planning/features/has-kickoff/spec.md", `---
title: Has Kickoff
type: feature
status: planning
---
# Has Kickoff

## Kickoff
Paste-ready opener.

## Goal
body.
`)

	// Convention/knowledge spec — kickoff is irrelevant, must skip.
	env.addSpec("conventions/some-conv/spec.md", `---
title: Some Convention
type: convention
status: active
---
# Some Convention
`)

	// Completed work spec — kickoff irrelevant, must skip.
	env.addSpec("specs/done-thing/spec.md", `---
title: Done Thing
type: feature
status: completed
---
# Done Thing
`)

	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check error: %v", err)
	}
	header := "Specs missing `## Kickoff` section"
	headerIdx := strings.Index(output, header)
	if headerIdx < 0 {
		t.Fatalf("expected kickoff-missing section, got:\n%s", output)
	}
	// Block ends at the next blank-line-separated paragraph.
	tail := output[headerIdx:]
	endIdx := strings.Index(tail, "\n\n")
	if endIdx < 0 {
		endIdx = len(tail)
	}
	block := tail[:endIdx]
	if !strings.Contains(block, "needs-kickoff") {
		t.Errorf("needs-kickoff should be flagged in block: %q", block)
	}
	if strings.Contains(block, "has-kickoff") {
		t.Errorf("has-kickoff should not be flagged: %q", block)
	}
	if strings.Contains(block, "some-conv") {
		t.Errorf("convention should be skipped: %q", block)
	}
	if strings.Contains(block, "done-thing") {
		t.Errorf("completed spec should be skipped: %q", block)
	}
	// Count assertion: only 1 missing-kickoff spec.
	if !strings.Contains(block, "(1)") {
		t.Errorf("expected (1) in header, got: %q", block)
	}
}

func TestCheckStaleDaysFlag(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-one/spec.md", `---
title: Feature One
type: feature
status: planning
---
# Feature One
`)

	env.indexAll()

	// With --stale-days=0, everything should be stale
	output, err := runCmd("check", "--stale-days", "0")
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}

	// The spec was just created, so with stale-days=0 it might or might not be stale
	// depending on timing. We just verify the command doesn't error.
	if !strings.Contains(output, "Hero workspace health check") {
		t.Errorf("check should show header: %q", output)
	}
}

// Severity-aware summary + kickoff collapse: many missing-Kickoff
// scaffolds are an advisory category, the list is collapsed, and the
// summary distinguishes advisory findings from failures.
func TestCheck_SeveritySummaryAndKickoffCollapse(t *testing.T) {
	env := newTestEnv(t)
	for _, slug := range []string{"nk1", "nk2", "nk3", "nk4", "nk5", "nk6", "nk7"} {
		env.addSpec("planning/features/"+slug+"/spec.md",
			"---\ntitle: "+slug+"\ntype: feature\nstatus: planning\nslug: "+slug+"\n---\n# "+slug+"\n")
	}
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check errored: %v", err)
	}
	if !strings.Contains(output, "and 2 more") {
		t.Errorf("expected kickoff list collapsed to '… and 2 more', got:\n%s", output)
	}
	if !strings.Contains(output, "advisory check(s)") {
		t.Errorf("expected severity-aware summary mentioning advisory check(s), got:\n%s", output)
	}
}

func TestMissingGoalInitiatives(t *testing.T) {
	env := newTestEnv(t)
	// Initiative WITH a Goal run-opener — should not be flagged.
	env.addSpec("planning/initiatives/has-goal/spec.md", `---
title: Has Goal
type: initiative
status: planning
---
# Has Goal

## Goal

Run the children autonomously.
`)
	// Initiative WITHOUT a Goal body — should be flagged (advisory).
	env.addSpec("planning/initiatives/no-goal/spec.md", `---
title: No Goal
type: initiative
status: planning
---
# No Goal

## Problem

Has no Goal run-opener.
`)
	// A leaf feature with no Goal — must NOT be flagged by this check.
	env.addSpec("planning/features/leaf/spec.md", `---
title: Leaf
type: feature
status: planning
---
# Leaf

## Kickoff

opener.
`)

	missing, err := missingGoalInitiatives(env.heroDir)
	if err != nil {
		t.Fatalf("missingGoalInitiatives: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("missingGoalInitiatives = %d specs, want 1", len(missing))
	}
	if missing[0].Slug != "no-goal" {
		t.Errorf("flagged %q, want no-goal (only the goalless initiative)", missing[0].Slug)
	}
}

// TestCheck_NearMissRelationKeyWarning covers AC-6 at the CLI surface: a
// frontmatter key that reads like a relation but forms no edge is named in the
// report rather than dropped in silence.
func TestCheck_NearMissRelationKeyWarning(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/gov/spec.md", `---
title: Governance
type: initiative
status: planning
slug: gov
subspecs: [alpha, bravo]
---
# Governance

## Kickoff

Pick up here.
`)
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check errored: %v", err)
	}
	if !strings.Contains(output, "look like relations") {
		t.Errorf("check should warn about near-miss relation keys: %q", output)
	}
	if !strings.Contains(output, "subspecs") {
		t.Errorf("check should name the offending key: %q", output)
	}
	if !strings.Contains(output, `"child"`) {
		t.Errorf("check should name the likely intended relation: %q", output)
	}
}

// TestCheck_NoNearMissWarningForAcceptedKeys guards the other half: keys the
// parser accepts must not be reported, or the warning becomes noise.
func TestCheck_NoNearMissWarningForAcceptedKeys(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/gov2/spec.md", `---
title: Governance Two
type: initiative
status: planning
slug: gov2
children: [alpha, bravo]
---
# Governance Two

## Kickoff

Pick up here.
`)
	env.indexAll()

	output, err := runCmd("check")
	if err != nil {
		t.Fatalf("check errored: %v", err)
	}
	if strings.Contains(output, "look like relations") {
		t.Errorf("children: is accepted and must not be reported as a near miss: %q", output)
	}
}

func TestInspectCodeIndexFreshnessReportsActualChangedMissingDeletedSources(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(root, "a.go")
	b := filepath.Join(root, "b.go")
	if err := os.WriteFile(a, []byte("package sample\nfunc Alpha() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package sample\nfunc Bravo() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.CodeScan.Parser = "heuristic"
	disabled := false
	cfg.Embeddings = &config.EmbeddingsConfig{Enabled: &disabled}
	if _, err := refreshCodeIndex(context.Background(), cfg, root, heroDir,
		codeRefreshOptions{Parser: "heuristic"}); err != nil {
		t.Fatalf("bootstrap code refresh: %v", err)
	}
	if got := inspectCodeIndexFreshness(cfg, root, heroDir); got.Status != "pass" ||
		!strings.Contains(got.Message, "reparsed=0") {
		t.Fatalf("fresh code coverage = %+v", got)
	}

	if err := os.WriteFile(a, []byte("package sample\nfunc AlphaChanged() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.go"), []byte("package sample\nfunc Charlie() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := inspectCodeIndexFreshness(cfg, root, heroDir)
	if got.Status != "warn" ||
		!strings.Contains(got.Message, "changed=1") ||
		!strings.Contains(got.Message, "missing=1") ||
		!strings.Contains(got.Message, "deleted=1") {
		t.Fatalf("stale code coverage = %+v", got)
	}
}

func TestInspectEmbeddingFreshnessReportsCoverageWithoutMutatingVectors(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(knowledgeDir, "one.md")
	if err := os.WriteFile(note, []byte("Knowledge: initial"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{Enabled: &enabled, Scope: []string{"knowledge"}}
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	model, err := embeddings.LoadModelFromConfig(cfg.EmbeddingsModel())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := embeddings.Refresh(heroDir, model, idx.RawDB(), nil, cfg.EmbeddingsScope()); err != nil {
		t.Fatalf("bootstrap embeddings: %v", err)
	}
	if got := inspectEmbeddingFreshness(cfg, heroDir, idx.RawDB()); got.Status != "pass" {
		t.Fatalf("fresh embedding coverage = %+v", got)
	}

	if err := os.WriteFile(note, []byte("Knowledge: changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(knowledgeDir, "two.md"), []byte("Knowledge: new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.RawDB().Exec(`
		INSERT INTO vec_chunks
			(chunk_id, corpus, source_id, section, text_hash, vector, embedded_at)
		VALUES ('knowledge:orphan.md', 'knowledge', 'orphan.md', '', 'old', X'', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := idx.RawDB().QueryRow(`SELECT COUNT(*) FROM vec_chunks`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	got := inspectEmbeddingFreshness(cfg, heroDir, idx.RawDB())
	if got.Status != "warn" ||
		!strings.Contains(got.Message, "missing=1") ||
		!strings.Contains(got.Message, "mismatched=1") ||
		!strings.Contains(got.Message, "orphaned=1") {
		t.Fatalf("stale embedding coverage = %+v", got)
	}
	var after int
	if err := idx.RawDB().QueryRow(`SELECT COUNT(*) FROM vec_chunks`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("read-only freshness check changed vector rows: before=%d after=%d", before, after)
	}
}

func TestInspectEmbeddingFreshnessReportsUnavailableGraphCorpus(t *testing.T) {
	heroDir := t.TempDir()
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	cfg := config.DefaultConfig()
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{Enabled: &enabled, Scope: []string{"code"}}
	got := inspectEmbeddingFreshness(cfg, heroDir, idx.RawDB())
	if got.Status != "warn" || !strings.Contains(got.Message, "unavailable") {
		t.Fatalf("unavailable code corpus = %+v", got)
	}
}

func TestInspectEmbeddingFreshnessSkipsCodeWhenCodeScanDisabled(t *testing.T) {
	heroDir := t.TempDir()
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	cfg := config.DefaultConfig()
	cfg.CodeScan.Depth = "disabled"
	enabled := true
	cfg.Embeddings = &config.EmbeddingsConfig{
		Enabled: &enabled,
		Scope:   []string{"code", "knowledge"},
	}
	got := inspectEmbeddingFreshness(cfg, heroDir, idx.RawDB())
	if got.Status != "pass" ||
		!strings.Contains(got.Message, "code skipped=") ||
		!strings.Contains(got.Message, "knowledge missing=0") ||
		strings.Contains(got.Message, "code unavailable=") {
		t.Fatalf("disabled code-scan embedding coverage = %+v", got)
	}
	cfg.CodeScan = nil
	got = inspectEmbeddingFreshness(cfg, heroDir, idx.RawDB())
	if got.Status != "pass" ||
		!strings.Contains(got.Message, "code skipped=") ||
		strings.Contains(got.Message, "code unavailable=") {
		t.Fatalf("nil code-scan embedding coverage = %+v", got)
	}
}
