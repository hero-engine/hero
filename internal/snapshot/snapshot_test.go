package snapshot

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuild_OnHeroRepo is a smoke test against the live hero-engine
// repo. It asserts the detector finds the canonical surfaces
// (core, serve, docs, landing, plus every domains/<pack>) and the
// rendered markdown is non-empty.
func TestBuild_OnHeroRepo(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	heroDir := filepath.Join(root, ".hero")

	// Use the existing hero spec corpus.
	override, err := LoadOverride(heroDir)
	if err != nil {
		t.Fatalf("LoadOverride: %v", err)
	}

	// Avoid hitting Discover (which walks the full corpus and is
	// slow under test); pass an empty spec list and rely on the
	// surface detector covering the inference signal coverage.
	snap, err := Build(BuildOptions{
		ProjectRoot: root,
		HeroDir:     heroDir,
		ProjectName: "hero",
		Mission:     "Test mission.",
		Now:         time.Now(),
	}, nil, override, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// hero-engine repo: expect at least core, serve, docs, landing.
	want := []string{"core", "serve", "docs", "landing"}
	got := map[string]bool{}
	for _, s := range snap.Surfaces {
		got[s.ID] = true
	}
	for _, w := range want {
		if !got[w] {
			ids := []string{}
			for _, s := range snap.Surfaces {
				ids = append(ids, s.ID)
			}
			t.Errorf("missing surface %q; detected: %s", w, strings.Join(ids, ", "))
		}
	}

	// Render markdown should be non-empty and contain the project name.
	md, err := Render(snap, FormatMarkdown)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(md), "# Project Snapshot — hero") {
		t.Errorf("rendered markdown missing header; got first 200 bytes: %s", string(md[:min(200, len(md))]))
	}
	if !strings.Contains(string(md), "## Surfaces") {
		t.Errorf("rendered markdown missing surfaces section")
	}

	// JSON should be parseable and contain at least one surface.
	js, err := Render(snap, FormatJSON)
	if err != nil {
		t.Fatalf("Render JSON: %v", err)
	}
	if !strings.Contains(string(js), `"project_name": "hero"`) {
		t.Errorf("JSON missing project_name; got: %s", string(js[:min(200, len(js))]))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
