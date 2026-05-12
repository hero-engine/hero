package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSubprojectFrontmatterAdds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte("---\ntitle: foo\ntype: feature\n---\n# foo\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSubprojectFrontmatter(path, "engines/mlx", false); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "subproject: engines/mlx\n") {
		t.Errorf("missing subproject line:\n%s", got)
	}
	if !strings.Contains(got, "title: foo") {
		t.Errorf("title was lost")
	}
	if !strings.Contains(got, "# foo\n\nbody") {
		t.Errorf("body was lost:\n%s", got)
	}
}

func TestWriteSubprojectFrontmatterReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	src := "---\ntitle: foo\ntype: feature\nsubproject: old/scope\n---\n# foo\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSubprojectFrontmatter(path, "new/scope", false); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "subproject: new/scope") {
		t.Errorf("subproject not updated:\n%s", got)
	}
	if strings.Contains(got, "old/scope") {
		t.Errorf("old scope still present:\n%s", got)
	}
	// Should not duplicate the line.
	count := strings.Count(got, "subproject:")
	if count != 1 {
		t.Errorf("expected exactly one subproject line, got %d", count)
	}
}

func TestWriteSubprojectFrontmatterDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	src := "---\ntitle: foo\n---\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSubprojectFrontmatter(path, "x", true); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != src {
		t.Errorf("dry-run modified file: %s", data)
	}
}

func TestWriteSubprojectFrontmatterNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte("# raw\nno frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeSubprojectFrontmatter(path, "x", false)
	if err == nil {
		t.Errorf("expected error on no frontmatter")
	}
}
