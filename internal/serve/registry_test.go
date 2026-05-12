package serve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_LoadEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	reg, err := LoadRegistryFrom(path)
	if err != nil {
		t.Fatalf("LoadRegistryFrom: %v", err)
	}
	if reg.Count() != 0 {
		t.Errorf("count = %d, want 0", reg.Count())
	}
}

func TestRegistry_AddAndSave(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "sub", "projects.json")

	// Create a fake project directory
	projectDir := t.TempDir()
	heroDir := filepath.Join(projectDir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	reg, err := LoadRegistryFrom(regPath)
	if err != nil {
		t.Fatal(err)
	}

	slug, err := reg.Add(projectDir)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if slug == "" {
		t.Fatal("expected non-empty slug")
	}
	if slug != filepath.Base(projectDir) {
		t.Errorf("slug = %q, want %q", slug, filepath.Base(projectDir))
	}

	if reg.Count() != 1 {
		t.Errorf("count = %d, want 1", reg.Count())
	}

	// Save and reload
	if err := reg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reg2, err := LoadRegistryFrom(regPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reg2.Count() != 1 {
		t.Errorf("reloaded count = %d, want 1", reg2.Count())
	}
	if !reg2.HasProject(slug) {
		t.Errorf("expected project %q after reload", slug)
	}
}

func TestRegistry_AddIdempotent(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)

	slug1, err := reg.Add(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	slug2, err := reg.Add(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	if slug1 != slug2 {
		t.Errorf("slugs differ: %q vs %q", slug1, slug2)
	}
	if reg.Count() != 1 {
		t.Errorf("count = %d, want 1 (idempotent)", reg.Count())
	}
}

func TestRegistry_AddNoHeroDir(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	reg, _ := LoadRegistryFrom(regPath)

	_, err := reg.Add(t.TempDir()) // no .hero dir
	if err == nil {
		t.Fatal("expected error for missing .hero directory")
	}
}

func TestRegistry_Remove(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)
	slug, _ := reg.Add(projectDir)

	if err := reg.Remove(slug); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if reg.Count() != 0 {
		t.Errorf("count = %d, want 0 after remove", reg.Count())
	}
}

func TestRegistry_RemoveNotFound(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	reg, _ := LoadRegistryFrom(regPath)

	err := reg.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error for removing nonexistent project")
	}
}

func TestRegistry_Get(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)
	slug, _ := reg.Add(projectDir)

	entry := reg.Get(slug)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Path != projectDir {
		// Resolve both to handle symlinks
		absEntry, _ := filepath.EvalSymlinks(entry.Path)
		absProject, _ := filepath.EvalSymlinks(projectDir)
		if absEntry != absProject {
			t.Errorf("path = %q, want %q", entry.Path, projectDir)
		}
	}

	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent slug")
	}
}

func TestRegistry_FindByPath(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)
	slug, _ := reg.Add(projectDir)

	found := reg.FindByPath(projectDir)
	if found != slug {
		t.Errorf("FindByPath = %q, want %q", found, slug)
	}

	if reg.FindByPath("/nonexistent/path") != "" {
		t.Error("expected empty string for unknown path")
	}
}

func TestRegistry_List(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")

	dir1 := filepath.Join(t.TempDir(), "project-a")
	dir2 := filepath.Join(t.TempDir(), "project-b")
	os.MkdirAll(filepath.Join(dir1, ".hero"), 0o755)
	os.MkdirAll(filepath.Join(dir2, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)
	reg.Add(dir1)
	reg.Add(dir2)

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("list length = %d, want 2", len(list))
	}
}

func TestRegistry_Slugs(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")
	dir1 := filepath.Join(t.TempDir(), "alpha")
	dir2 := filepath.Join(t.TempDir(), "beta")
	os.MkdirAll(filepath.Join(dir1, ".hero"), 0o755)
	os.MkdirAll(filepath.Join(dir2, ".hero"), 0o755)

	reg, _ := LoadRegistryFrom(regPath)
	reg.Add(dir1)
	reg.Add(dir2)

	slugs := reg.Slugs()
	if len(slugs) != 2 {
		t.Errorf("slugs length = %d, want 2", len(slugs))
	}
}

func TestRegistry_LoadExisting(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "projects.json")

	content := `{
  "projects": {
    "myapp": {
      "path": "/home/user/myapp",
      "registered": "2026-04-12T10:00:00Z"
    }
  }
}`
	os.WriteFile(regPath, []byte(content), 0o644)

	reg, err := LoadRegistryFrom(regPath)
	if err != nil {
		t.Fatalf("LoadRegistryFrom: %v", err)
	}
	if reg.Count() != 1 {
		t.Errorf("count = %d, want 1", reg.Count())
	}
	if !reg.HasProject("myapp") {
		t.Error("expected myapp to be registered")
	}
	entry := reg.Get("myapp")
	if entry.Path != "/home/user/myapp" {
		t.Errorf("path = %q, want /home/user/myapp", entry.Path)
	}
}
