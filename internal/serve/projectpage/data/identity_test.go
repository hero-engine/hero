package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIdentity_HappyPath(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge", "conventions", "foo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "knowledge", "conventions", "foo", "spec.md"),
		[]byte("---\ntitle: Foo\ntype: convention\nstatus: active\n---\n# Foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	id := LoadIdentity(IdentityInputs{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		Slug:        "test-project",
	})
	if !id.HasProject {
		t.Fatal("expected HasProject=true")
	}
	if id.Slug != "test-project" {
		t.Errorf("Slug = %q, want test-project", id.Slug)
	}
	if id.Name != filepath.Base(dir) {
		t.Errorf("Name = %q, want %q", id.Name, filepath.Base(dir))
	}
	if id.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", id.ProjectRoot, dir)
	}
	if id.ProjectRootURL == "" || id.ProjectRootURL[:7] != "file://" {
		t.Errorf("ProjectRootURL should be file:// URL, got %q", id.ProjectRootURL)
	}
	if id.SpecCount < 1 {
		t.Errorf("SpecCount = %d, want >=1", id.SpecCount)
	}
	if id.ConventionCount != 1 {
		t.Errorf("ConventionCount = %d, want 1", id.ConventionCount)
	}
}

func TestLoadIdentity_MissingProjectRoot(t *testing.T) {
	id := LoadIdentity(IdentityInputs{Slug: "ghost"})
	if id.HasProject {
		t.Fatal("expected HasProject=false on empty ProjectRoot")
	}
	if id.Slug != "ghost" {
		t.Errorf("Slug = %q, want ghost", id.Slug)
	}
	if id.SpecCount != 0 || id.ConventionCount != 0 {
		t.Errorf("counts should all be zero, got SpecCount=%d ConventionCount=%d",
			id.SpecCount, id.ConventionCount)
	}
}
