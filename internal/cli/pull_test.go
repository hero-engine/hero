package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestPull_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "pull")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestPull_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("sync", "pull", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestPull_NoTrackerConfigured(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "pull", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for no tracker")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("error = %q, want 'no tracker configured'", err.Error())
	}
}

func TestPull_NoTrackerID(t *testing.T) {
	env := newTestEnv(t)

	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := env.heroDir + "/planning/features/csv-export/spec.md"
	_, err := runCmd("sync", "pull", specPath)
	if err == nil {
		t.Fatal("expected error for no tracker_id")
	}
	if !strings.Contains(err.Error(), "no tracker_id") {
		t.Errorf("error = %q, want 'no tracker_id'", err.Error())
	}
}

// --- mapTrackerStatus tests ---

func TestMapTrackerStatus(t *testing.T) {
	tests := []struct {
		status      string
		trackerType string
		want        string
	}{
		{"open", "github", string(spec.StatusPlanning)},
		{"closed", "github", string(spec.StatusCompleted)},
		{"Open", "github", string(spec.StatusPlanning)},
		{"CLOSED", "github", string(spec.StatusCompleted)},
		{"In Progress", "jira", string(spec.StatusDelivering)},
		{"in_progress", "linear", string(spec.StatusDelivering)},
		{"In Review", "jira", string(spec.StatusInReview)},
		{"Done", "jira", string(spec.StatusCompleted)},
		{"To Do", "jira", string(spec.StatusPlanning)},
		{"Backlog", "linear", string(spec.StatusPlanning)},
		{"weird-status", "github", ""},
	}

	for _, tt := range tests {
		got := mapTrackerStatus(tt.status, tt.trackerType)
		if got != tt.want {
			t.Errorf("mapTrackerStatus(%q, %q) = %q, want %q", tt.status, tt.trackerType, got, tt.want)
		}
	}
}
