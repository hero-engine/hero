package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const featureWithKickoff = `---
title: Has Kickoff
type: feature
status: planning
horizon: now
tags: [demo, kickoff]
---
# Has Kickoff

## Kickoff

Demo spec with a real kickoff body.

**Status:** planning — written for the test.
**Pick up at:** confirm the test asserts.

→ ` + "`hero queue`" + `

## Goal

Body content.
`

const featurePinned = `---
title: Pinned Spec
type: feature
status: planning
pinned: true
---
# Pinned Spec

## Kickoff

Pinned to the top of the queue.

## Goal
x
`

const featureBlocked = `---
title: Blocked Spec
type: feature
status: planning
relations:
  - target: missing-prereq
    kind: depends-on
---
# Blocked Spec

## Kickoff

Should not show up in queue (depends on incomplete spec).
`

const featurePrereqOpen = `---
title: Prereq Open
type: feature
status: planning
slug: missing-prereq
---
# Prereq Open

## Kickoff

The dependency that keeps Blocked Spec out of the queue.
`

const featureCompleted = `---
title: Completed Spec
type: feature
status: completed
---
# Completed Spec

## Kickoff

Should not appear in default list output.
`

func TestListExcludesCompletedByDefault(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	env.addSpec("specs/done-spec/spec.md", featureCompleted)

	out, err := runCmd("list", "--format", "text")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "has-kickoff") {
		t.Errorf("expected has-kickoff in output, got:\n%s", out)
	}
	if strings.Contains(out, "done-spec") {
		t.Errorf("completed spec should be excluded by default, got:\n%s", out)
	}
}

func TestListWithExplicitCompletedStatus(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	env.addSpec("specs/done-spec/spec.md", featureCompleted)

	out, err := runCmd("list", "--status", "completed", "--format", "text")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "done-spec") {
		t.Errorf("explicit --status completed should surface the spec, got:\n%s", out)
	}
	if strings.Contains(out, "has-kickoff") {
		t.Errorf("only completed should match; got:\n%s", out)
	}
}

func TestListReadyAndBlockedAreMutuallyExclusive(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	_, err := runCmd("list", "--ready", "--blocked")
	if err == nil {
		t.Fatal("expected error when both --ready and --blocked are set")
	}
}

func TestListBlockedFiltersToUnmetDeps(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	env.addSpec("specs/blocked/spec.md", featureBlocked)
	env.addSpec("specs/missing-prereq/spec.md", featurePrereqOpen)

	out, err := runCmd("list", "--blocked", "--format", "text")
	if err != nil {
		t.Fatalf("list --blocked: %v", err)
	}
	if !strings.Contains(out, "blocked") {
		t.Errorf("expected blocked spec in output, got:\n%s", out)
	}
	if strings.Contains(out, "has-kickoff") {
		t.Errorf("has-kickoff has no deps, should not be blocked: \n%s", out)
	}
}

func TestQueuePrioritizesPinned(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/regular/spec.md", featureWithKickoff)
	env.addSpec("specs/pinned/spec.md", featurePinned)

	out, err := runCmd("queue", "--format", "text")
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	pinIdx := strings.Index(out, "pinned")
	regIdx := strings.Index(out, "regular")
	if pinIdx < 0 {
		t.Fatalf("pinned spec missing from queue:\n%s", out)
	}
	if regIdx < 0 {
		t.Fatalf("regular spec missing from queue:\n%s", out)
	}
	if pinIdx > regIdx {
		t.Errorf("pinned should come before non-pinned (pin=%d reg=%d):\n%s", pinIdx, regIdx, out)
	}
}

func TestQueueExcludesBlocked(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	env.addSpec("specs/blocked/spec.md", featureBlocked)
	env.addSpec("specs/missing-prereq/spec.md", featurePrereqOpen)

	out, err := runCmd("queue", "--format", "text")
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if strings.Contains(out, "specs/blocked") || strings.Contains(out, "Blocked Spec") {
		t.Errorf("queue should not contain blocked spec, got:\n%s", out)
	}
}

func TestQueueRendersKickoffBody(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)

	out, err := runCmd("queue", "--format", "kickoff")
	if err != nil {
		t.Fatalf("queue kickoff: %v", err)
	}
	if !strings.Contains(out, "Demo spec with a real kickoff body.") {
		t.Errorf("queue kickoff format should render body content, got:\n%s", out)
	}
	if !strings.Contains(out, "has-kickoff") {
		t.Errorf("queue kickoff should include slug header, got:\n%s", out)
	}
}

func TestListJSONFormatIncludesKickoff(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)

	out, err := runCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("list json: %v", err)
	}
	if !strings.Contains(out, `"slug": "has-kickoff"`) {
		t.Errorf("expected slug in JSON, got:\n%s", out)
	}
	if !strings.Contains(out, `"kickoff":`) {
		t.Errorf("expected kickoff field in JSON, got:\n%s", out)
	}
}

func TestListUnknownSortFails(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	_, err := runCmd("list", "--sort", "by-vibe")
	if err == nil {
		t.Fatal("expected error for unknown sort key")
	}
}

func TestQueueWriteCreatesSnapshot(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)
	env.addSpec("specs/pinned/spec.md", featurePinned)

	if _, err := runCmd("queue", "write", "-q"); err != nil {
		t.Fatalf("queue write: %v", err)
	}

	target := filepath.Join(env.heroDir, "QUEUE.md")
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected QUEUE.md at %s: %v", target, err)
	}
	content := string(body)

	for _, want := range []string{
		"Auto-generated by `hero queue write`",
		"# Hero Ready Queue",
		"pinned",
		"Demo spec with a real kickoff body.",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("QUEUE.md missing %q in:\n%s", want, content)
		}
	}

	// Pinned spec should appear before regular spec in the snapshot.
	pinIdx := strings.Index(content, "pinned")
	regIdx := strings.Index(content, "has-kickoff")
	if pinIdx < 0 || regIdx < 0 {
		t.Fatalf("missing slugs in snapshot:\n%s", content)
	}
	if pinIdx > regIdx {
		t.Errorf("pinned should rank ahead of non-pinned in snapshot")
	}
}

func TestQueueWriteEmptyQueue(t *testing.T) {
	env := newTestEnv(t)
	// No specs at all → empty queue, but write should still succeed.
	if _, err := runCmd("queue", "write", "-q"); err != nil {
		t.Fatalf("queue write on empty workspace: %v", err)
	}

	target := filepath.Join(env.heroDir, "QUEUE.md")
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("QUEUE.md should still be written: %v", err)
	}
	if !strings.Contains(string(body), "Queue is empty") {
		t.Errorf("empty queue should produce empty-state message:\n%s", body)
	}
}

func TestQueueWriteOverwrites(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/has-kickoff/spec.md", featureWithKickoff)

	// First write
	if _, err := runCmd("queue", "write", "-q"); err != nil {
		t.Fatalf("first write: %v", err)
	}

	// Pollute the file with hand edits to confirm overwrite
	target := filepath.Join(env.heroDir, "QUEUE.md")
	if err := os.WriteFile(target, []byte("HAND EDIT — should be wiped\n"), 0o644); err != nil {
		t.Fatalf("pollute file: %v", err)
	}

	// Second write should fully overwrite.
	if _, err := runCmd("queue", "write", "-q"); err != nil {
		t.Fatalf("second write: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read after second write: %v", err)
	}
	if strings.Contains(string(body), "HAND EDIT") {
		t.Errorf("second write should overwrite hand edits; got:\n%s", body)
	}
	if !strings.Contains(string(body), "# Hero Ready Queue") {
		t.Errorf("second write should produce real snapshot; got:\n%s", body)
	}
}
