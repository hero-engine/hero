package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeInflightSpec is a local fixture helper that writes a minimal
// spec.md so spec.Discover picks it up. Mirrors the work-data
// writeSpec helper but kept local to avoid cross-package test
// imports.
func writeInflightSpec(t *testing.T, heroDir, slug, typ, status string) string {
	t.Helper()
	dir := filepath.Join(heroDir, "planning", typ+"s", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\ntitle: " + slug + "\ntype: " + typ + "\nstatus: " + status + "\n---\n\nbody.\n"
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadInflight_FiltersByStatus(t *testing.T) {
	dir := t.TempDir()
	writeInflightSpec(t, dir, "a-deliver", "feature", "delivering")
	writeInflightSpec(t, dir, "a-plan", "feature", "planning")
	writeInflightSpec(t, dir, "a-review", "feature", "in-review")
	writeInflightSpec(t, dir, "a-done", "feature", "completed")

	got := LoadInflight(InflightInputs{HeroDir: dir})
	if got.Total != 3 {
		t.Fatalf("expected 3 in-flight specs (planning/delivering/in-review), got %d: %+v", got.Total, got.Rows)
	}
	statuses := map[string]bool{}
	for _, r := range got.Rows {
		statuses[r.Status] = true
	}
	for _, want := range []string{"delivering", "planning", "in-review"} {
		if !statuses[want] {
			t.Errorf("missing status %q in inflight set", want)
		}
	}
}

func TestLoadInflight_OrdersNewestTouchedFirst(t *testing.T) {
	dir := t.TempDir()
	p1 := writeInflightSpec(t, dir, "older", "feature", "delivering")
	p2 := writeInflightSpec(t, dir, "newer", "feature", "delivering")
	// Set mtimes so p2 is more recent.
	old := time.Now().Add(-7 * 24 * time.Hour)
	fresh := time.Now().Add(-30 * time.Minute)
	if err := os.Chtimes(p1, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	if err := os.Chtimes(p2, fresh, fresh); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	got := LoadInflight(InflightInputs{HeroDir: dir})
	if len(got.Rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(got.Rows))
	}
	if got.Rows[0].Slug != "newer" {
		t.Errorf("expected newest first, got %q", got.Rows[0].Slug)
	}
}

func TestLoadInflight_SkipsNonWorkTypes(t *testing.T) {
	dir := t.TempDir()
	writeInflightSpec(t, dir, "a-feat", "feature", "delivering")
	writeInflightSpec(t, dir, "a-note", "note", "planning")
	writeInflightSpec(t, dir, "a-conv", "convention", "planning")
	got := LoadInflight(InflightInputs{HeroDir: dir})
	for _, r := range got.Rows {
		if r.Slug == "a-note" || r.Slug == "a-conv" {
			t.Errorf("non-work spec %q surfaced in inflight strip", r.Slug)
		}
	}
}

func TestLoadInflight_AggregateAcrossProjects(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	p1 := writeInflightSpec(t, d1, "alpha-spec", "feature", "delivering")
	p2 := writeInflightSpec(t, d2, "beta-spec", "feature", "planning")
	old := time.Now().Add(-7 * 24 * time.Hour)
	fresh := time.Now().Add(-30 * time.Minute)
	os.Chtimes(p1, old, old)
	os.Chtimes(p2, fresh, fresh)

	got := LoadInflight(InflightInputs{
		Aggregate: []ActivityProject{
			{Slug: "alpha", HeroDir: d1},
			{Slug: "beta", HeroDir: d2},
		},
	})
	if !got.Aggregate {
		t.Errorf("expected Aggregate=true")
	}
	if len(got.Rows) != 2 {
		t.Fatalf("expected 2 rows across projects, got %d", len(got.Rows))
	}
	if got.Rows[0].Project != "beta" {
		t.Errorf("expected beta first (newer mtime), got %q", got.Rows[0].Project)
	}
}

func TestLoadInflight_EmptyHeroDir(t *testing.T) {
	got := LoadInflight(InflightInputs{HeroDir: ""})
	if got.Total != 0 || len(got.Rows) != 0 {
		t.Errorf("expected empty payload, got %+v", got)
	}
}
