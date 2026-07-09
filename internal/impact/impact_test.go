package impact

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/index"
)

// TestAnalyze_FlatKnowledge covers the impact follow-on: a flat code-scoped
// convention in the isolated knowledge table governs a file and must appear in
// the impact report, at parity with spec.md conventions.
func TestAnalyze_FlatKnowledge(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	kPath := filepath.Join(heroDir, "knowledge", "conventions", "contracts-import-discipline.md")
	if err := os.MkdirAll(filepath.Dir(kPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	os.WriteFile(kPath, []byte(`---
title: Contracts Import Discipline
type: convention
scope:
  - internal/contracts/*.go
---
# Contracts Import Discipline
Never import internal packages across the contracts boundary.
`), 0o644)

	if _, err := index.RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	reports, err := Analyze(idx, []string{"internal/contracts/manifest.go"})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("want 1 report, got %d", len(reports))
	}
	var found bool
	for _, c := range reports[0].Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			found = true
		}
	}
	if !found {
		t.Errorf("flat scoped convention did not surface in impact; conventions=%+v", reports[0].Conventions)
	}
}

func TestRenderText_Empty(t *testing.T) {
	out := RenderText(nil)
	if out != "No files to analyze.\n" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRenderText_NoImpact(t *testing.T) {
	reports := []Report{{
		FilePath: "internal/foo/bar.go",
	}}
	out := RenderText(reports)
	if out == "" {
		t.Error("expected non-empty output")
	}
	if !containsStr(out, "No impact detected") {
		t.Error("expected 'No impact detected' message")
	}
}

func TestRenderText_WithSpecs(t *testing.T) {
	reports := []Report{{
		FilePath: "internal/auth/session.go",
		Specs: []SpecRef{
			{Slug: "auth-flow", Title: "Auth Flow", Status: "delivering"},
			{Slug: "session-mgmt", Title: "Session Management", Status: "completed"},
		},
		Conventions: []ConvRef{
			{Slug: "error-handling", Title: "Error Handling Convention"},
		},
	}}
	out := RenderText(reports)
	if !containsStr(out, "Specs (2)") {
		t.Error("expected 'Specs (2)' in output")
	}
	if !containsStr(out, "auth-flow") {
		t.Error("expected spec slug in output")
	}
	if !containsStr(out, "Conventions (1)") {
		t.Error("expected conventions section")
	}
	if !containsStr(out, "2 spec(s)") {
		t.Error("expected summary with spec count")
	}
}

func TestRenderJSON(t *testing.T) {
	reports := []Report{{
		FilePath: "test.go",
		Specs:    []SpecRef{{Slug: "test", Title: "Test", Status: "planning"}},
	}}
	out, err := RenderJSON(reports)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Error("expected non-empty JSON")
	}
	if !containsStr(out, "test.go") {
		t.Error("expected file path in JSON")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
