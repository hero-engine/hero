package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckEmpty(t *testing.T) {
	env := newTestEnv(t)
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
