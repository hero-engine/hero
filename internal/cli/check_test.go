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
