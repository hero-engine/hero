package integrity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

const completedSpecMarkdown = `---
title: Sample Spec
type: feature
status: completed
priority: P0
---

## Goal

Ship the thing.
`

func TestAutoDowngradeRegressions_DowngradesCompletedSpecsWithFailingACs(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte(completedSpecMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t)
	seedFailingCriterion(t, store, "feat-x:AC-1", "first AC", "feat-x", "regressed")
	seedFailingCriterion(t, store, "feat-x:AC-2", "second AC", "feat-x", "passing")

	specs := []*spec.Spec{
		{Slug: "feat-x", Path: specPath, Status: spec.StatusCompleted},
	}

	out, err := AutoDowngradeRegressions(specs, store, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("downgrades = %d, want 1", len(out))
	}
	if out[0].NewStatus != spec.StatusRegressed {
		t.Errorf("NewStatus = %q, want regressed", out[0].NewStatus)
	}
	if len(out[0].RegressedACs) != 1 || out[0].RegressedACs[0] != "feat-x:AC-1" {
		t.Errorf("RegressedACs = %v", out[0].RegressedACs)
	}

	body, _ := os.ReadFile(specPath)
	if !strings.Contains(string(body), "status: regressed") {
		t.Errorf("file not rewritten:\n%s", body)
	}
	if !strings.Contains(string(body), "auto_downgraded:") {
		t.Errorf("annotation missing:\n%s", body)
	}
}

func TestAutoDowngradeRegressions_SkipsSpecsAlreadyRegressed(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte(strings.Replace(completedSpecMarkdown, "completed", "regressed", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t)
	seedFailingCriterion(t, store, "feat-x:AC-1", "first", "feat-x", "regressed")
	specs := []*spec.Spec{
		{Slug: "feat-x", Path: specPath, Status: spec.StatusRegressed},
	}

	out, err := AutoDowngradeRegressions(specs, store, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("downgrades = %d, want 0 (already regressed)", len(out))
	}
}

func TestAutoDowngradeRegressions_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte(completedSpecMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(specPath)

	store := openTestStore(t)
	seedFailingCriterion(t, store, "feat-x:AC-1", "first", "feat-x", "failing")
	specs := []*spec.Spec{
		{Slug: "feat-x", Path: specPath, Status: spec.StatusCompleted},
	}

	out, err := AutoDowngradeRegressions(specs, store, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Errorf("plan should still report 1, got %d", len(out))
	}
	after, _ := os.ReadFile(specPath)
	if string(after) != string(original) {
		t.Errorf("dry-run rewrote the file:\n%s", after)
	}
}

func TestAutoDowngradeRegressions_PlanningSpecsAreLeftAlone(t *testing.T) {
	store := openTestStore(t)
	seedFailingCriterion(t, store, "feat-x:AC-1", "first", "feat-x", "failing")
	specs := []*spec.Spec{
		{Slug: "feat-x", Path: "/not/used", Status: spec.StatusPlanning},
	}
	out, err := AutoDowngradeRegressions(specs, store, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("planning spec touched: %+v", out)
	}
}

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func seedFailingCriterion(t *testing.T, store *graph.Store, key, statement, parent, status string) {
	t.Helper()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Criterion",
		Key:  key,
		Props: map[string]any{
			"ac_id":     key,
			"statement": statement,
			"status":    status,
			"parent":    parent,
		},
		Repo:   "repo-x",
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}
