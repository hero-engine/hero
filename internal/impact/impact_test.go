package impact

import (
	"testing"
)

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
