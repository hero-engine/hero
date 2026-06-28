package cli

import (
	"os"
	"path/filepath"
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

const goalSupervisedInitiative = `---
title: Drive
type: initiative
status: planning
autonomy: supervised
horizon: now
---
# Drive

## Goal

Run all the children to completion.
`

func TestGoalPauseWritesQuestionThenResumesOnAnswer(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/drive/spec.md", goalSupervisedInitiative)
	env.addSpec("planning/initiatives/drive/child-a/spec.md", goalChildA)

	// 1. --check pauses (supervised), writes the question + ledger.
	out, err := runCmd("goal", "drive", "--check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, `"verdict": "pause"`) {
		t.Fatalf("supervised should pause, got:\n%s", out)
	}
	nextPath := filepath.Join(env.heroDir, "NEXT.md")
	if b, _ := os.ReadFile(nextPath); !strings.Contains(string(b), "Drive paused") {
		t.Errorf("pause question not written to NEXT.md, got:\n%s", b)
	}
	if _, statErr := os.Stat(filepath.Join(env.heroDir, "drive", "drive.json")); statErr != nil {
		t.Errorf("run ledger not written: %v", statErr)
	}

	// 2. Idempotent: a second --check still pauses, one question block.
	if _, err := runCmd("goal", "drive", "--check"); err != nil {
		t.Fatalf("check 2: %v", err)
	}
	if b, _ := os.ReadFile(nextPath); strings.Count(string(b), "Drive paused — needs you") != 1 {
		t.Errorf("question should not duplicate on repeated check:\n%s", b)
	}

	// 3. Answer clears the question.
	if _, err := runCmd("goal", "drive", "--answer", "go ahead"); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if b, _ := os.ReadFile(nextPath); strings.Contains(string(b), "Drive paused") {
		t.Errorf("question should be cleared after answer, got:\n%s", b)
	}

	// 4. --check now resumes (continue) past the answered transition.
	out4, err := runCmd("goal", "drive", "--check")
	if err != nil {
		t.Fatalf("check 4: %v", err)
	}
	if !strings.Contains(out4, `"verdict": "continue"`) {
		t.Fatalf("should resume to continue after answer, got:\n%s", out4)
	}
}

func TestGoalAnswerRecordsOutcomeAndTrust(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/drive/spec.md", goalInitiative) // guided
	env.addSpec("planning/initiatives/drive/child-a/spec.md", goalChildA)

	// Seed a ledger with a *promotable* pause (DesignFork) at child-a.
	driveDir := filepath.Join(env.heroDir, "drive")
	if err := os.MkdirAll(driveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ledgerJSON := `{"initiative":"drive","pause":{"spec":"child-a","category":"DesignFork","reason":"pick an approach"}}`
	if err := os.WriteFile(filepath.Join(driveDir, "drive.json"), []byte(ledgerJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Answer it: records an approved outcome for DesignFork + a feed event.
	if _, err := runCmd("goal", "drive", "--answer", "go with option A"); err != nil {
		t.Fatalf("answer: %v", err)
	}
	// Event logged.
	if b, _ := os.ReadFile(filepath.Join(env.heroDir, "events.log")); !strings.Contains(string(b), "drive.pause_outcome") {
		t.Errorf("pause outcome not logged to events.log, got:\n%s", b)
	}
	// --trust reflects the recorded DesignFork streak.
	out, err := runCmd("goal", "drive", "--trust")
	if err != nil {
		t.Fatalf("trust: %v", err)
	}
	if !strings.Contains(out, "DesignFork") {
		t.Errorf("--trust should show the recorded DesignFork category, got:\n%s", out)
	}

	// --untrust resets it.
	if _, err := runCmd("goal", "drive", "--untrust", "DesignFork"); err != nil {
		t.Fatalf("untrust: %v", err)
	}
	out2, err := runCmd("goal", "drive", "--trust")
	if err != nil {
		t.Fatalf("trust 2: %v", err)
	}
	if strings.Contains(out2, "DesignFork") {
		t.Errorf("--untrust should have removed DesignFork, got:\n%s", out2)
	}
}
