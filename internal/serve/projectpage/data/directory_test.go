package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirectory_Empty(t *testing.T) {
	out := LoadDirectory(DirectoryInputs{})
	if len(out.Rows) != 0 {
		t.Errorf("Rows len = %d, want 0", len(out.Rows))
	}
}

func TestLoadDirectory_SortsBySlug(t *testing.T) {
	dirA := mkProject(t, "alpha")
	dirB := mkProject(t, "bravo")
	out := LoadDirectory(DirectoryInputs{
		Projects: []DirectoryProject{
			{Slug: "bravo", ProjectRoot: dirB, HeroDir: filepath.Join(dirB, ".hero")},
			{Slug: "alpha", ProjectRoot: dirA, HeroDir: filepath.Join(dirA, ".hero")},
		},
	})
	if len(out.Rows) != 2 {
		t.Fatalf("Rows len = %d, want 2", len(out.Rows))
	}
	if out.Rows[0].Slug != "alpha" || out.Rows[1].Slug != "bravo" {
		t.Errorf("got order [%s, %s], want [alpha, bravo]", out.Rows[0].Slug, out.Rows[1].Slug)
	}
}

func TestLoadDirectory_DegradedRow_MissingPath(t *testing.T) {
	out := LoadDirectory(DirectoryInputs{
		Projects: []DirectoryProject{{Slug: "broken"}},
	})
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1", len(out.Rows))
	}
	if !out.Rows[0].Degraded {
		t.Error("expected Degraded=true for missing path row")
	}
	if out.Rows[0].Health != "unknown" {
		t.Errorf("Health = %q, want unknown", out.Rows[0].Health)
	}
}

func TestLoadDirectory_TrackerFromConfig(t *testing.T) {
	dir := mkProject(t, "with-tracker")
	heroJSON := []byte(`{"tracker":{"type":"github","project":"hero/hero"}}`)
	if err := os.WriteFile(filepath.Join(dir, ".hero", "hero.json"), heroJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	out := LoadDirectory(DirectoryInputs{
		Projects: []DirectoryProject{{
			Slug: "with-tracker", ProjectRoot: dir, HeroDir: filepath.Join(dir, ".hero"),
		}},
	})
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1", len(out.Rows))
	}
	if out.Rows[0].Tracker != "github" {
		t.Errorf("Tracker = %q, want github", out.Rows[0].Tracker)
	}
}

// mkProject creates a fixture project dir with an empty .hero subtree.
func mkProject(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
