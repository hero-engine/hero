package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverKnowledge_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	entries := discoverKnowledge(dir)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestDiscoverKnowledge_NoKnowledgeDir(t *testing.T) {
	dir := t.TempDir()
	entries := discoverKnowledge(dir)
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
}

func TestDiscoverKnowledge_StandaloneWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	kDir := filepath.Join(dir, "knowledge", "conventions")
	os.MkdirAll(kDir, 0755)

	content := "---\ntitle: Cross-Repo Workflow\ntype: convention\nstatus: active\n---\n\nSome content here.\n"
	os.WriteFile(filepath.Join(kDir, "cross-repo-workflow.md"), []byte(content), 0644)

	entries := discoverKnowledge(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Category != "conventions" {
		t.Errorf("category = %q, want conventions", e.Category)
	}
	if e.Slug != "cross-repo-workflow" {
		t.Errorf("slug = %q, want cross-repo-workflow", e.Slug)
	}
	if e.Title != "Cross-Repo Workflow" {
		t.Errorf("title = %q, want Cross-Repo Workflow", e.Title)
	}
	if e.Checksum == "" {
		t.Error("checksum should not be empty")
	}
}

func TestDiscoverKnowledge_StandaloneWithHeading(t *testing.T) {
	dir := t.TempDir()
	kDir := filepath.Join(dir, "knowledge", "context")
	os.MkdirAll(kDir, 0755)

	content := "# CLI Sync Protocol\n\nDescription of the protocol.\n"
	os.WriteFile(filepath.Join(kDir, "cli-sync-protocol.md"), []byte(content), 0644)

	entries := discoverKnowledge(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Category != "context" {
		t.Errorf("category = %q, want context", e.Category)
	}
	if e.Slug != "cli-sync-protocol" {
		t.Errorf("slug = %q, want cli-sync-protocol", e.Slug)
	}
	if e.Title != "CLI Sync Protocol" {
		t.Errorf("title = %q, want CLI Sync Protocol", e.Title)
	}
}

func TestDiscoverKnowledge_SpecMdInSubdir(t *testing.T) {
	dir := t.TempDir()
	kDir := filepath.Join(dir, "knowledge", "rules", "project-rules")
	os.MkdirAll(kDir, 0755)

	content := "---\ntitle: Project Rules\n---\n\nRules content.\n"
	os.WriteFile(filepath.Join(kDir, "spec.md"), []byte(content), 0644)

	entries := discoverKnowledge(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Category != "rules" {
		t.Errorf("category = %q, want rules", e.Category)
	}
	if e.Slug != "project-rules" {
		t.Errorf("slug = %q, want project-rules", e.Slug)
	}
	if e.Title != "Project Rules" {
		t.Errorf("title = %q, want Project Rules", e.Title)
	}
}

func TestDiscoverKnowledge_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	convDir := filepath.Join(dir, "knowledge", "conventions")
	ctxDir := filepath.Join(dir, "knowledge", "context")
	os.MkdirAll(convDir, 0755)
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(convDir, "naming.md"), []byte("# Naming\n"), 0644)
	os.WriteFile(filepath.Join(convDir, "testing.md"), []byte("# Testing\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "arch.md"), []byte("# Architecture\n"), 0644)

	entries := discoverKnowledge(dir)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestDiscoverKnowledge_NoTitle(t *testing.T) {
	dir := t.TempDir()
	kDir := filepath.Join(dir, "knowledge")
	os.MkdirAll(kDir, 0755)

	os.WriteFile(filepath.Join(kDir, "bare-notes.md"), []byte("Some plain text without heading.\n"), 0644)

	entries := discoverKnowledge(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "bare-notes" {
		t.Errorf("title = %q, want bare-notes (slug fallback)", entries[0].Title)
	}
	if entries[0].Category != "general" {
		t.Errorf("category = %q, want general", entries[0].Category)
	}
}

func TestKnowledgeCategoryAndSlug(t *testing.T) {
	tests := []struct {
		rel      string
		wantCat  string
		wantSlug string
	}{
		{"foo.md", "general", "foo"},
		{"conventions/naming.md", "conventions", "naming"},
		{"decisions/auth/spec.md", "decisions", "auth"},
		{"rules/project-rules/spec.md", "rules", "project-rules"},
		{"context/deep/nested.md", "context", "deep/nested"},
	}
	for _, tt := range tests {
		cat, slug := knowledgeCategoryAndSlug(tt.rel)
		if cat != tt.wantCat || slug != tt.wantSlug {
			t.Errorf("knowledgeCategoryAndSlug(%q) = (%q, %q), want (%q, %q)",
				tt.rel, cat, slug, tt.wantCat, tt.wantSlug)
		}
	}
}

func TestExtractKnowledgeTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"frontmatter", "---\ntitle: My Title\nstatus: active\n---\n# Heading\n", "My Title"},
		{"heading only", "# My Heading\nSome text.\n", "My Heading"},
		{"quoted title", "---\ntitle: \"Quoted Title\"\n---\n", "Quoted Title"},
		{"no title", "Just plain text.\n", ""},
		{"heading after blank lines", "\n\n# Late Heading\n", "Late Heading"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKnowledgeTitle(tt.content)
			if got != tt.want {
				t.Errorf("extractKnowledgeTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
