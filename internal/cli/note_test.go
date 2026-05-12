package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoteCreatesWithSlug(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("note", "auth-brainstorm")
	if err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	if !strings.Contains(output, "Created note") {
		t.Errorf("output missing 'Created note': %q", output)
	}
	if !strings.Contains(output, "auth-brainstorm") {
		t.Errorf("output missing slug: %q", output)
	}
}

func TestNoteCreatesSpecFile(t *testing.T) {
	env := newTestEnv(t)

	_, err := runCmd("note", "my-thoughts")
	if err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "notes", "my-thoughts", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: note") {
		t.Error("spec.md missing type: note")
	}
	if !strings.Contains(content, "# My Thoughts") {
		t.Error("spec.md missing title heading")
	}
}

func TestNoteInlineText(t *testing.T) {
	env := newTestEnv(t)

	_, err := runCmd("note", "thinking about auth flow")
	if err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	// Should create a slug from the text
	specPath := filepath.Join(env.heroDir, "knowledge", "notes", "thinking-about-auth-flow", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "thinking about auth flow") {
		t.Error("spec.md missing inline text as title")
	}
	if !strings.Contains(content, "thinking about auth flow") {
		t.Error("spec.md missing inline text as body")
	}
}

func TestNoteMultipleArgsInlineText(t *testing.T) {
	env := newTestEnv(t)

	_, err := runCmd("note", "maybe", "we", "should", "use", "redis")
	if err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "notes", "maybe-we-should-use-redis", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "maybe we should use redis") {
		t.Errorf("spec.md missing inline text: %s", content)
	}
}

func TestNoteAutoSlug(t *testing.T) {
	env := newTestEnv(t)

	_, err := runCmd("note")
	if err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	// Should auto-generate a date-based slug
	notesDir := filepath.Join(env.heroDir, "knowledge", "notes")
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no note directory created")
	}

	// Slug should start with a date pattern like "2026-04-12"
	name := entries[0].Name()
	if len(name) < 10 || name[4] != '-' || name[7] != '-' {
		t.Errorf("auto-generated slug doesn't look like a date: %q", name)
	}
}

func TestNoteRefusesDuplicate(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("note", "my-idea")
	if err != nil {
		t.Fatalf("first note returned error: %v", err)
	}

	_, err = runCmd("note", "my-idea")
	if err == nil {
		t.Fatal("second note should fail due to collision")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

func TestNoteFromFile(t *testing.T) {
	env := newTestEnv(t)

	// Create a source file
	sourceContent := "## Discussion\n\nWe talked about using JWT tokens for auth.\nThe consensus was to use short-lived tokens with refresh.\n"
	sourcePath := filepath.Join(env.dir, "convo.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := runCmd("note", "auth-discussion", "--from", sourcePath)
	if err != nil {
		t.Fatalf("note --from returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "notes", "auth-discussion", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "JWT tokens") {
		t.Error("spec.md missing content from source file")
	}
	if !strings.Contains(content, "short-lived tokens") {
		t.Error("spec.md missing content from source file")
	}
}

func TestNoteFromFileMissing(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("note", "bad-import", "--from", "/nonexistent/file.md")
	if err == nil {
		t.Fatal("note --from with missing file should fail")
	}
}

func TestNoteRequiresWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("note", "orphan-thought")
	if err == nil {
		t.Fatal("note should fail without hero workspace")
	}

	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error should mention workspace: %v", err)
	}
}

func TestTextToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"thinking about auth flow", "thinking-about-auth-flow"},
		{"Let's use Redis!", "lets-use-redis"},
		{"simple", "simple"},
		{"UPPER CASE STUFF", "upper-case-stuff"},
		{"lots   of   spaces", "lots-of-spaces"},
		{"special!@#$chars", "specialchars"},
		{"trailing-hyphen-", "trailing-hyphen"},
	}

	for _, tt := range tests {
		got := textToSlug(tt.input)
		if got != tt.want {
			t.Errorf("textToSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Tests for `hero note` showing up in `hero knowledge` were removed
// when `hero knowledge` was archived (graph-aware `hero search`
// supersedes it). Note creation is exercised by the file-existence
// assertions in TestNote* above.

func TestNewNoteType(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "design-thoughts", "--type", "note")
	if err != nil {
		t.Fatalf("new --type note returned error: %v", err)
	}

	if !strings.Contains(output, "note") {
		t.Errorf("output missing note type: %q", output)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "notes", "design-thoughts", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: note") {
		t.Error("spec.md missing type: note")
	}
	if !strings.Contains(content, "# Design Thoughts") {
		t.Error("spec.md missing title")
	}
}
