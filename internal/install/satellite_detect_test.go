package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCandidates(t *testing.T) {
	root := t.TempDir()

	// engines/mlx — go.mod + nested .hero/
	mlx := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(filepath.Join(mlx, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mlx, "go.mod"), []byte("module x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// app — package.json
	app := filepath.Join(root, "app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// vendor — package.json (should be excluded if listed)
	ven := filepath.Join(root, "vendor-thing")
	if err := os.MkdirAll(ven, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ven, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// node_modules deeply nested package.json (must be ignored)
	deep := filepath.Join(app, "node_modules", "lib", "sub")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := &SubprojectsManifest{}
	manifest.AddExcluded("vendor-thing")

	cs, err := DetectCandidates(root, manifest, 4)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range cs {
		got[c.Path] = true
	}
	if !got["engines/mlx"] {
		t.Errorf("expected engines/mlx in candidates, got %+v", cs)
	}
	if !got["app"] {
		t.Errorf("expected app in candidates")
	}
	if got["vendor-thing"] {
		t.Errorf("excluded vendor-thing should not be in candidates")
	}
	// Folder with HasNestedHero should sort first.
	if cs[0].Path != "engines/mlx" {
		t.Errorf("expected engines/mlx first (has nested hero), got %s", cs[0].Path)
	}
	if !cs[0].HasNestedHero {
		t.Errorf("expected HasNestedHero=true")
	}
}

func TestDetectCandidatesSkipsDeclared(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := &SubprojectsManifest{}
	manifest.AddSubproject(Subproject{Path: "app", Scope: "app"})
	cs, err := DetectCandidates(root, manifest, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Path == "app" {
			t.Errorf("declared app should not appear in candidates")
		}
	}
}

func TestFindNestedHeroDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "engines", "mlx", ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := FindNestedHeroDirs(root)
	if len(got) != 1 || got[0] != "engines/mlx" {
		t.Errorf("got %v, want [engines/mlx]", got)
	}
}
