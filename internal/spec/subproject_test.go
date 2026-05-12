package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSubprojectFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	content := "---\ntitle: x\ntype: feature\nsubproject: engines/mlx\n---\n# x\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s.Subproject != "engines/mlx" {
		t.Errorf("subproject = %q, want engines/mlx", s.Subproject)
	}
}

func TestSubprojectMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	content := "---\ntitle: x\ntype: feature\n---\n# x\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Subproject != "" {
		t.Errorf("subproject = %q, want empty", s.Subproject)
	}
}
