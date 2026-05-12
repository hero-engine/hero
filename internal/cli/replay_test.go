package cli

import (
	"strings"
	"testing"
	"time"
)

func TestReplayCmd_SpecNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sprint", "retro", "nonexistent-spec")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestReplayCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("sprint", "retro", "some-spec")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestReplayCmd_RequiresArg(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("sprint", "retro")
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestReplayCmd_BasicOutput(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/auth-login/spec.md", `---
title: Auth Login
type: feature
status: completed
claimed_by: alice
tracker_id: GH-42
---

## Goal
Build login flow.

## Design
OAuth2 integration.

## Changes
- internal/auth/login.go
- internal/auth/session.go

## Acceptance Criteria
- Users can log in
`)

	out, err := runCmd("sprint", "retro", "auth-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check header info
	if !strings.Contains(out, "Replay: auth-login") {
		t.Error("expected spec slug in output")
	}
	if !strings.Contains(out, "Auth Login") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "completed") {
		t.Error("expected status in output")
	}
	if !strings.Contains(out, "alice") {
		t.Error("expected owner in output")
	}
	if !strings.Contains(out, "GH-42") {
		t.Error("expected tracker ID in output")
	}

	// Check sections completeness
	if !strings.Contains(out, "[x] Goal") {
		t.Error("expected Goal checked")
	}
	if !strings.Contains(out, "[x] Design") {
		t.Error("expected Design checked")
	}
	if !strings.Contains(out, "[x] Changes") {
		t.Error("expected Changes checked")
	}
	if !strings.Contains(out, "[x] Acceptance Criteria") {
		t.Error("expected Acceptance Criteria checked")
	}
}

func TestReplayCmd_NoFilesData(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/minimal/spec.md", `---
title: Minimal Spec
type: feature
status: planning
---

## Goal
Just a goal, no changes section.
`)

	out, err := runCmd("sprint", "retro", "minimal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No file tracking data") {
		t.Error("expected 'no file tracking data' message")
	}
}

func TestReplayCmd_FileAnalysis(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/with-files/spec.md", `---
title: With Files
type: feature
status: delivering
---

## Goal
Test file analysis.

## Changes
- src/foo.go
- src/bar.go
- src/baz.go
`)

	out, err := runCmd("sprint", "retro", "with-files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should show file analysis even without git (gitFiles will be nil)
	if !strings.Contains(out, "File Analysis") {
		t.Error("expected File Analysis section")
	}
	if !strings.Contains(out, "Planned:    3 files") {
		t.Error("expected 3 planned files")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		hours    float64
		expected string
	}{
		{"less than hour", 0.5, "less than an hour"},
		{"one hour", 1, "1 hour"},
		{"few hours", 5, "5 hours"},
		{"one day", 24, "1 day"},
		{"few days", 72, "3 days"},
		{"one week", 168, "1 week"},
		{"two weeks", 336, "2 weeks"},
		{"week and days", 192, "1 week 1 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := time.Duration(tt.hours * float64(time.Hour))
			result := formatDuration(d)
			if result != tt.expected {
				t.Errorf("formatDuration(%v hours) = %q, want %q", tt.hours, result, tt.expected)
			}
		})
	}
}
