package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const taskTestSpec = `---
title: Checkout flow
type: feature
status: delivering
---

## Goal

Stand up the checkout.

## Acceptance Criteria

- AC-1: User can pay.

## Boundaries

Out of scope.
`

func TestHeroTask_AddCreatesTasksSection(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/checkout-flow/spec.md", taskTestSpec)

	out, err := runCmd("task", "add", "checkout-flow", "Fix login redirect loop", "--kind", "qa-blocker", "--assignee", "chet")
	if err != nil {
		t.Fatalf("task add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "T-1") {
		t.Errorf("output missing T-1: %q", out)
	}

	body, _ := os.ReadFile(filepath.Join(env.heroDir, "planning/features/checkout-flow/spec.md"))
	if !strings.Contains(string(body), "## Tasks") {
		t.Errorf("Tasks section not added:\n%s", body)
	}
	if !strings.Contains(string(body), "T-1 Fix login redirect loop") {
		t.Errorf("task line missing:\n%s", body)
	}
	if !strings.Contains(string(body), "kind: qa-blocker") {
		t.Errorf("metadata missing:\n%s", body)
	}
	if !strings.Contains(string(body), "## Boundaries") {
		t.Errorf("Boundaries section lost:\n%s", body)
	}
}

func TestHeroTask_AddSecondIncrementsID(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/checkout-flow/spec.md", taskTestSpec)

	if _, err := runCmd("task", "add", "checkout-flow", "First task"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	out, err := runCmd("task", "add", "checkout-flow", "Second task")
	if err != nil {
		t.Fatalf("second add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "T-2") {
		t.Errorf("expected T-2 in output: %q", out)
	}
}

func TestHeroTask_StartAndDoneTransitions(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/checkout-flow/spec.md", taskTestSpec)

	if _, err := runCmd("task", "add", "checkout-flow", "Wire it up"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := runCmd("task", "start", "checkout-flow", "T-1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	body, _ := os.ReadFile(filepath.Join(env.heroDir, "planning/features/checkout-flow/spec.md"))
	if !strings.Contains(string(body), "[/] T-1") {
		t.Errorf("T-1 not flipped to doing:\n%s", body)
	}
	if !strings.Contains(string(body), "started:") {
		t.Errorf("started timestamp missing:\n%s", body)
	}

	if _, err := runCmd("task", "done", "checkout-flow", "T-1"); err != nil {
		t.Fatalf("done: %v", err)
	}
	body, _ = os.ReadFile(filepath.Join(env.heroDir, "planning/features/checkout-flow/spec.md"))
	if !strings.Contains(string(body), "[x] T-1") {
		t.Errorf("T-1 not flipped to done:\n%s", body)
	}
	if !strings.Contains(string(body), "done:") {
		t.Errorf("done timestamp missing:\n%s", body)
	}
}

func TestHeroTask_AddUnknownSlugErrors(t *testing.T) {
	newTestEnv(t)
	if _, err := runCmd("task", "add", "no-such-spec", "Hi"); err == nil {
		t.Errorf("expected error for unknown slug")
	}
}

func TestHeroTask_ListEmptyEmitsHelpfulMessage(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/empty-feat/spec.md", taskTestSpec)

	out, err := runCmd("task", "list", "empty-feat")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "No tasks") {
		t.Errorf("expected 'No tasks' hint: %q", out)
	}
}

func TestHeroTask_ListJSONAfterAdd(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/checkout-flow/spec.md", taskTestSpec)

	if _, err := runCmd("task", "add", "checkout-flow", "Test task", "--kind", "qa-blocker"); err != nil {
		t.Fatalf("add: %v", err)
	}
	out, err := runCmd("task", "list", "checkout-flow", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	if !strings.Contains(out, `"task_id":"T-1"`) {
		t.Errorf("JSON output missing task_id: %q", out)
	}
	if !strings.Contains(out, `"kind":"qa-blocker"`) {
		t.Errorf("JSON output missing kind: %q", out)
	}
}

// TestHeroTask_AcceptanceUntouched verifies that adding tasks doesn't
// disturb the existing acceptance-criteria section. Belt-and-
// suspenders for the boundary contract: tasks ships beside AC, not
// on top of it.
func TestHeroTask_AcceptanceUntouched(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/checkout-flow/spec.md", taskTestSpec)

	if _, err := runCmd("task", "add", "checkout-flow", "New task"); err != nil {
		t.Fatalf("add: %v", err)
	}
	body, _ := os.ReadFile(filepath.Join(env.heroDir, "planning/features/checkout-flow/spec.md"))
	if !strings.Contains(string(body), "## Acceptance Criteria") {
		t.Errorf("Acceptance Criteria section lost:\n%s", body)
	}
	if !strings.Contains(string(body), "AC-1: User can pay.") {
		t.Errorf("AC-1 line lost:\n%s", body)
	}
}
