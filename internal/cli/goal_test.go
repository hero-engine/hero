package cli

import (
	"strings"
	"testing"
)

const goalInitiative = `---
title: Drive
type: initiative
status: planning
autonomy: guided
horizon: now
---
# Drive

## Goal

Run all the children to completion.
`

const goalChildA = `---
title: Child A
type: feature
status: planning
horizon: now
relations:
  - target: drive
    kind: parent
---
# Child A

## Kickoff

deliver child a
`

func TestGoalEmit(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/drive/spec.md", goalInitiative)
	env.addSpec("planning/initiatives/drive/child-a/spec.md", goalChildA)

	out, err := runCmd("goal", "drive")
	if err != nil {
		t.Fatalf("goal emit: %v", err)
	}
	if !strings.Contains(out, "Run all the children to completion.") {
		t.Errorf("emit should print the Goal objective, got:\n%s", out)
	}
	if !strings.Contains(out, "Run until every child") || !strings.Contains(out, "child-a") {
		t.Errorf("emit should print the derived run condition listing children, got:\n%s", out)
	}
}

func TestGoalCheckContinue(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/drive/spec.md", goalInitiative)
	env.addSpec("planning/initiatives/drive/child-a/spec.md", goalChildA)

	out, err := runCmd("goal", "drive", "--check")
	if err != nil {
		t.Fatalf("goal check: %v", err)
	}
	if !strings.Contains(out, `"verdict": "continue"`) {
		t.Errorf("guided run with a ready child should continue, got:\n%s", out)
	}
	if !strings.Contains(out, `"next_spec": "child-a"`) {
		t.Errorf("check should name the next spec, got:\n%s", out)
	}
	if !strings.Contains(out, "deliver child a") {
		t.Errorf("continue verdict should carry the child kickoff, got:\n%s", out)
	}
}

func TestGoalRejectsNonInitiative(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/drive/spec.md", goalInitiative)
	env.addSpec("planning/initiatives/drive/child-a/spec.md", goalChildA)

	_, err := runCmd("goal", "child-a")
	if err == nil {
		t.Fatal("hero goal on a non-initiative spec should error")
	}
	if !strings.Contains(err.Error(), "not an initiative") {
		t.Errorf("error should explain it needs an initiative, got: %v", err)
	}
}
