package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

func TestCheckMissingFiles(t *testing.T) {
	dir := t.TempDir()
	// Create one file, leave the other missing
	os.WriteFile(filepath.Join(dir, "exists.go"), []byte("package x"), 0o644)

	s := &spec.Spec{
		Slug:         "test-slug",
		FilesTouched: []string{"exists.go", "missing.go"},
	}

	r := &Report{Slug: s.Slug}
	checkMissingFiles(r, s, dir)

	if len(r.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(r.Signals))
	}
	if r.Signals[0].Kind != "missing_file" {
		t.Errorf("expected missing_file, got %s", r.Signals[0].Kind)
	}
	if r.Signals[0].Severity != SeverityWarning {
		t.Errorf("expected warning severity, got %s", r.Signals[0].Severity)
	}
}

func TestCheckMissingFiles_AllExist(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package x"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package x"), 0o644)

	s := &spec.Spec{FilesTouched: []string{"a.go", "b.go"}}
	r := &Report{}
	checkMissingFiles(r, s, dir)

	if len(r.Signals) != 0 {
		t.Errorf("expected 0 signals, got %d", len(r.Signals))
	}
}

// TestCheckConventions_FlatKnowledge covers the drift follow-on: a flat
// code-scoped convention in the isolated knowledge table governs a touched
// file and must appear in the drift report, at parity with spec.md conventions.
func TestCheckConventions_FlatKnowledge(t *testing.T) {
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

	s := &spec.Spec{Slug: "some-work", FilesTouched: []string{"internal/contracts/manifest.go"}}
	r := &Report{Slug: s.Slug}
	checkConventions(r, s, idx)

	var found bool
	for _, c := range r.Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			found = true
		}
	}
	if !found {
		t.Errorf("flat scoped convention did not surface in drift; conventions=%+v", r.Conventions)
	}

	// Non-matching file → no flat convention.
	s2 := &spec.Spec{Slug: "other", FilesTouched: []string{"internal/other/thing.go"}}
	r2 := &Report{Slug: s2.Slug}
	checkConventions(r2, s2, idx)
	for _, c := range r2.Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			t.Errorf("flat convention surfaced for non-matching file")
		}
	}
}

func TestCheckBoundaries(t *testing.T) {
	s := &spec.Spec{
		Sections: map[string]string{
			"boundaries": "- Does **not** touch `internal/auth/middleware.go`\n- Does **not** modify the payment gateway",
		},
	}

	changed := map[string]bool{
		"internal/auth/middleware.go": true,
		"internal/api/handler.go":    true,
	}

	r := &Report{}
	checkBoundaries(r, s, "/tmp", changed)

	if len(r.Signals) != 1 {
		t.Fatalf("expected 1 boundary violation, got %d", len(r.Signals))
	}
	if r.Signals[0].Kind != "boundary_violation" {
		t.Errorf("expected boundary_violation, got %s", r.Signals[0].Kind)
	}
	if r.Signals[0].Severity != SeverityViolation {
		t.Errorf("expected violation severity, got %s", r.Signals[0].Severity)
	}
}

func TestCheckBoundaries_NoBoundaries(t *testing.T) {
	s := &spec.Spec{Sections: map[string]string{}}
	r := &Report{}
	checkBoundaries(r, s, "/tmp", map[string]bool{"foo.go": true})

	if len(r.Signals) != 0 {
		t.Errorf("expected 0 signals, got %d", len(r.Signals))
	}
}

func TestCheckCriteria_EARS(t *testing.T) {
	dir := t.TempDir()
	// Create a changed file that contains relevant keywords
	os.WriteFile(filepath.Join(dir, "handler.go"), []byte("func streamResponse() { buffer := new(bytes.Buffer) }"), 0o644)

	s := &spec.Spec{
		Sections: map[string]string{
			"acceptance criteria": "- WHEN export size exceeds 10MB THE SYSTEM SHALL stream rather than buffer",
		},
	}

	changed := map[string]bool{"handler.go": true}
	r := &Report{}
	checkCriteria(r, s, dir, changed)

	if len(r.Criteria) != 1 {
		t.Fatalf("expected 1 criterion status, got %d", len(r.Criteria))
	}
	if !r.Criteria[0].Addressed {
		t.Error("criterion should be addressed — 'stream' and 'buffer' appear in handler.go")
	}
}

func TestCheckCriteria_Unaddressed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package main\nfunc hello() {}"), 0o644)

	s := &spec.Spec{
		Sections: map[string]string{
			"acceptance criteria": "- WHEN export size exceeds 10MB THE SYSTEM SHALL stream rather than buffer",
		},
	}

	changed := map[string]bool{"handler.go": true}
	r := &Report{}
	checkCriteria(r, s, dir, changed)

	if len(r.Criteria) != 1 {
		t.Fatalf("expected 1 criterion status, got %d", len(r.Criteria))
	}
	if r.Criteria[0].Addressed {
		t.Error("criterion should not be addressed — keywords not in code")
	}
	if len(r.Signals) == 0 {
		t.Error("expected an unaddressed_criterion signal")
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		text string
		want int // minimum expected keywords
	}{
		{"stream rather than buffer", 2},
		{"--foo flag", 1},
		{"exit code 0", 2},
		{"the system shall", 0}, // all stop words
	}

	for _, tt := range tests {
		got := extractKeywords(tt.text)
		if len(got) < tt.want {
			t.Errorf("extractKeywords(%q) = %v, want at least %d keywords", tt.text, got, tt.want)
		}
	}
}

func TestExtractNegativePaths(t *testing.T) {
	tests := []struct {
		boundaries string
		want       int
	}{
		{"Does **not** touch `internal/auth/middleware.go`", 1},
		{"Does not modify the auth middleware", 1},
		{"Must not change `pkg/core.go`\nShall not edit `lib/util.go`", 2},
		{"This is fine to change", 0},
	}

	for _, tt := range tests {
		got := extractNegativePaths(tt.boundaries)
		if len(got) != tt.want {
			t.Errorf("extractNegativePaths(%q) = %v (len %d), want %d", tt.boundaries, got, len(got), tt.want)
		}
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		signals []Signal
		want    int
	}{
		{nil, 0},
		{[]Signal{{Severity: SeverityWarning}}, 1},
		{[]Signal{{Severity: SeverityViolation}}, 2},
		{[]Signal{{Severity: SeverityWarning}, {Severity: SeverityViolation}}, 2},
	}

	for _, tt := range tests {
		got := exitCode(tt.signals)
		if got != tt.want {
			t.Errorf("exitCode(%v) = %d, want %d", tt.signals, got, tt.want)
		}
	}
}

func TestRenderText_NoSpecs(t *testing.T) {
	out := RenderText(nil)
	if out != "No specs to analyze.\n" {
		t.Errorf("unexpected output for empty: %q", out)
	}
}

func TestRenderText_Clean(t *testing.T) {
	reports := []*Report{{
		Slug:   "my-feature",
		Status: "delivering",
	}}
	out := RenderText(reports)
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestRenderJSON(t *testing.T) {
	reports := []*Report{{
		Slug:     "test",
		Status:   "delivering",
		ExitCode: 0,
		Signals:  []Signal{},
	}}
	out, err := RenderJSON(reports)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		changed string
		pattern string
		want    bool
	}{
		{"internal/auth/middleware.go", "internal/auth/middleware.go", true},
		{"internal/auth/middleware.go", "auth/middleware", true},
		{"internal/api/handler.go", "auth/middleware", false},
	}

	for _, tt := range tests {
		got := pathMatches(tt.changed, tt.pattern)
		if got != tt.want {
			t.Errorf("pathMatches(%q, %q) = %v, want %v", tt.changed, tt.pattern, got, tt.want)
		}
	}
}
