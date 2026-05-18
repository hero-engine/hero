package recap

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSince_Default(t *testing.T) {
	since, err := ParseSince("")
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Since(since)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected ~24h ago, got %v ago", diff)
	}
}

func TestParseSince_Hours(t *testing.T) {
	since, err := ParseSince("48h")
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Since(since)
	if diff < 47*time.Hour || diff > 49*time.Hour {
		t.Errorf("expected ~48h ago, got %v ago", diff)
	}
}

func TestParseSince_Days(t *testing.T) {
	since, err := ParseSince("2d")
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Since(since)
	if diff < 47*time.Hour || diff > 49*time.Hour {
		t.Errorf("expected ~48h ago, got %v ago", diff)
	}
}

func TestParseSince_Weeks(t *testing.T) {
	since, err := ParseSince("1w")
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Since(since)
	expected := 7 * 24 * time.Hour
	if diff < expected-time.Hour || diff > expected+time.Hour {
		t.Errorf("expected ~7d ago, got %v ago", diff)
	}
}

func TestParseSince_ISODate(t *testing.T) {
	since, err := ParseSince("2026-04-20")
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := time.Parse("2006-01-02", "2026-04-20")
	if !since.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, since)
	}
}

func TestParseSince_Invalid(t *testing.T) {
	_, err := ParseSince("garbage")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestRenderText_Empty(t *testing.T) {
	r := &Recap{
		Since: time.Now().Add(-24 * time.Hour),
		Until: time.Now(),
	}
	out := RenderText(r)
	if out == "" {
		t.Error("expected non-empty output")
	}
	if !contains(out, "No activity") {
		t.Error("expected 'No activity' message")
	}
}

func TestRenderText_WithSpecs(t *testing.T) {
	r := &Recap{
		Since: time.Now().Add(-24 * time.Hour),
		Until: time.Now(),
		Specs: []SpecActivity{{
			Slug:      "csv-export",
			Title:     "CSV Export",
			NewStatus: "delivering",
			Commits: []CommitSummary{{
				Hash:    "abc12345",
				Subject: "add streaming",
				Author:  "test",
				Date:    "2026-04-22",
			}},
			FilesTouched: []string{"internal/export/csv.go"},
		}},
		Unmatched: []CommitSummary{{
			Hash:    "def67890",
			Subject: "fix typo in README",
			Author:  "test",
			Date:    "2026-04-22",
		}},
	}
	out := RenderText(r)
	if !contains(out, "csv-export") {
		t.Error("expected spec slug in output")
	}
	if !contains(out, "1 commits") {
		t.Error("expected commit count")
	}
	if !contains(out, "unmatched") {
		t.Error("expected unmatched section")
	}
}

func TestRenderJSON(t *testing.T) {
	r := &Recap{
		Since: time.Now().Add(-24 * time.Hour),
		Until: time.Now(),
	}
	out, err := RenderJSON(r)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestIsKnowledgePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".hero/planning/decisions/use-fts5/spec.md", true},
		{".hero/planning/conventions/error-handling/spec.md", true},
		{".hero/knowledge/context.md", true},
		{"internal/api/handler.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		got := isKnowledgePath(tt.path)
		if got != tt.want {
			t.Errorf("isKnowledgePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// TestBuild_EmptyRepo verifies that hero recap against a freshly-initialized
// git repo with no commits returns an empty recap (not an error). Regression
// guard for the "your current branch 'main' does not have any commits yet"
// path that previously bubbled up as `building recap: reading git log: exit
// status 128`.
func TestBuild_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v: %s", err, out)
	}
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir heroDir: %v", err)
	}

	r, err := Build(heroDir, dir, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Build on empty repo returned error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Recap")
	}
	if len(r.Specs) != 0 || len(r.Knowledge) != 0 || len(r.Unmatched) != 0 {
		t.Errorf("expected empty recap, got specs=%d knowledge=%d unmatched=%d",
			len(r.Specs), len(r.Knowledge), len(r.Unmatched))
	}
	// And the human render should fall through to "No activity in this window."
	if out := RenderText(r); !contains(out, "No activity") {
		t.Errorf("expected 'No activity' in render, got: %q", out)
	}
}

// TestBuild_NotAGitRepo confirms we still surface real git failures rather
// than swallowing them along with the empty-repo case.
func TestBuild_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir heroDir: %v", err)
	}

	_, err := Build(heroDir, dir, time.Now().Add(-24*time.Hour))
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
	if !contains(err.Error(), "reading git log") {
		t.Errorf("expected 'reading git log' in error, got: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && containsStr(s, sub)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
