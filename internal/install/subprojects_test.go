package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSubprojectsMissingFile(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m, err := LoadSubprojects(heroDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(m.Subprojects) != 0 || len(m.Excluded) != 0 {
		t.Errorf("expected empty manifest, got %+v", m)
	}
}

func TestSaveAndLoadSubprojects(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := &SubprojectsManifest{}
	m.AddSubproject(Subproject{Path: "engines/mlx", Scope: "engines/mlx", Description: "Apple Silicon"})
	m.AddSubproject(Subproject{Path: "engines/cuda", Scope: "engines/cuda"})
	m.AddExcluded("vendor")

	if err := SaveSubprojects(heroDir, m); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSubprojects(heroDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Subprojects) != 2 {
		t.Errorf("got %d subprojects, want 2", len(loaded.Subprojects))
	}
	// Check sort by path.
	if loaded.Subprojects[0].Path != "engines/cuda" {
		t.Errorf("expected engines/cuda first after sort, got %s", loaded.Subprojects[0].Path)
	}
	if !loaded.IsExcluded("vendor") {
		t.Errorf("vendor should be excluded")
	}
	if !loaded.IsDeclared("engines/mlx") {
		t.Errorf("engines/mlx should be declared")
	}
}

func TestAddSubprojectIdempotent(t *testing.T) {
	m := &SubprojectsManifest{}
	m.AddSubproject(Subproject{Path: "x", Scope: "x"})
	m.AddSubproject(Subproject{Path: "x", Scope: "x", Description: "updated"})
	if len(m.Subprojects) != 1 {
		t.Errorf("expected idempotent add, got %d entries", len(m.Subprojects))
	}
	if m.Subprojects[0].Description != "updated" {
		t.Errorf("expected update to overwrite, got %+v", m.Subprojects[0])
	}
}

func TestRemoveSubproject(t *testing.T) {
	m := &SubprojectsManifest{}
	m.AddSubproject(Subproject{Path: "a"})
	m.AddSubproject(Subproject{Path: "b"})
	if !m.RemoveSubproject("a") {
		t.Errorf("expected remove to return true")
	}
	if m.IsDeclared("a") {
		t.Errorf("a should be gone")
	}
	if !m.IsDeclared("b") {
		t.Errorf("b should remain")
	}
}

func TestNormalizeRelPath(t *testing.T) {
	cases := map[string]string{
		"a/b":    "a/b",
		"a/b/":   "a/b",
		"./a/b":  "a/b",
		"a//b":   "a/b",
		"  a/b ": "a/b",
		"":       "",
	}
	for in, want := range cases {
		if got := normalizeRelPath(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
