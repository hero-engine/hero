package spec

import (
	"os"
	"path/filepath"
	"testing"
)

// symlinked-hero-dir: a hero folder that is a symlink to a directory
// elsewhere is discovered like a real one, with paths under the link.
func TestDiscoverFollowsSymlinkedHeroDir(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "shared-hero")
	write := func(rel, content string) {
		path := filepath.Join(real, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("planning/features/a/spec.md", "---\ntitle: A\nslug: a\ntype: feature\nstatus: planning\n---\n# A\n")
	write("specs/b/spec.md", "---\ntitle: B\nslug: b\ntype: feature\nstatus: completed\n---\n# B\n")
	heroDir := filepath.Join(base, "repo", ".hero")
	if err := os.MkdirAll(filepath.Dir(heroDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, heroDir); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	specs, err := Discover(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*Spec{}
	for _, s := range specs {
		got[s.Slug] = s
	}
	a, b := got["a"], got["b"]
	if a == nil || b == nil {
		t.Fatalf("discovered %d specs through the symlink, want a and b", len(specs))
	}
	if a.Path != filepath.Join(heroDir, "planning", "features", "a", "spec.md") {
		t.Errorf("a.Path = %s, want it under the symlinked hero dir", a.Path)
	}
	if a.Archived || !b.Archived {
		t.Errorf("Archived: a=%v b=%v, want a=false b=true", a.Archived, b.Archived)
	}
}
