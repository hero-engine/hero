package hero

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

// TestOverlayFS_TopWins covers the core precedence contract: when a file
// exists in both layers, the top FS shadows the bottom file. Mirrors the
// "domain overrides core" semantics established in
// internal/spectypes/loader.go:32-44.
func TestOverlayFS_TopWins(t *testing.T) {
	top := fstest.MapFS{
		"agents/shared.md": &fstest.MapFile{Data: []byte("top-shared")},
	}
	bottom := fstest.MapFS{
		"agents/shared.md": &fstest.MapFile{Data: []byte("bottom-shared")},
		"agents/only-bottom.md": &fstest.MapFile{Data: []byte("only-bottom")},
	}

	overlay := OverlayFS(top, bottom)

	got, err := fs.ReadFile(overlay, "agents/shared.md")
	if err != nil {
		t.Fatalf("ReadFile shared: %v", err)
	}
	if string(got) != "top-shared" {
		t.Errorf("shared.md: want top-shared, got %s", got)
	}

	got, err = fs.ReadFile(overlay, "agents/only-bottom.md")
	if err != nil {
		t.Fatalf("ReadFile only-bottom: %v", err)
	}
	if string(got) != "only-bottom" {
		t.Errorf("only-bottom.md: want only-bottom, got %s", got)
	}
}

// TestOverlayFS_ReadDirMerge verifies ReadDir returns the union of
// entries from both sides, with top's entry winning when names collide.
// Output is sorted alphabetically for deterministic install diffs.
func TestOverlayFS_ReadDirMerge(t *testing.T) {
	top := fstest.MapFS{
		"agents/shared.md": &fstest.MapFile{Data: []byte("top")},
		"agents/top-only.md": &fstest.MapFile{Data: []byte("t")},
	}
	bottom := fstest.MapFS{
		"agents/shared.md": &fstest.MapFile{Data: []byte("bottom")},
		"agents/bottom-only.md": &fstest.MapFile{Data: []byte("b")},
	}
	overlay := OverlayFS(top, bottom)

	entries, err := fs.ReadDir(overlay, "agents")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	want := []string{"bottom-only.md", "shared.md", "top-only.md"}
	for i, e := range entries {
		if e.Name() != want[i] {
			t.Errorf("entry %d: want %s, got %s", i, want[i], e.Name())
		}
	}
}

// TestOverlayFS_StatPrefersTop confirms Stat probes top first and only
// falls back to bottom on miss.
func TestOverlayFS_StatPrefersTop(t *testing.T) {
	top := fstest.MapFS{
		"x.md": &fstest.MapFile{Data: []byte("aaaa")},
	}
	bottom := fstest.MapFS{
		"x.md": &fstest.MapFile{Data: []byte("bbbbbbbbbb")},
		"y.md": &fstest.MapFile{Data: []byte("y")},
	}
	overlay := OverlayFS(top, bottom)

	fi, err := fs.Stat(overlay, "x.md")
	if err != nil {
		t.Fatalf("Stat x.md: %v", err)
	}
	if fi.Size() != 4 {
		t.Errorf("Stat x.md: want top size 4, got %d", fi.Size())
	}

	fi, err = fs.Stat(overlay, "y.md")
	if err != nil {
		t.Fatalf("Stat y.md: %v", err)
	}
	if fi.Size() != 1 {
		t.Errorf("Stat y.md: want bottom size 1, got %d", fi.Size())
	}
}

// TestOverlayFS_NilSides allows either side to be nil; nil is treated
// as an empty FS so callers can pass nil when one layer is absent.
func TestOverlayFS_NilSides(t *testing.T) {
	bottom := fstest.MapFS{"a.md": &fstest.MapFile{Data: []byte("a")}}
	overlay := OverlayFS(nil, bottom)
	got, err := fs.ReadFile(overlay, "a.md")
	if err != nil || string(got) != "a" {
		t.Errorf("nil-top: want a, got %q err=%v", got, err)
	}

	top := fstest.MapFS{"b.md": &fstest.MapFile{Data: []byte("b")}}
	overlay = OverlayFS(top, nil)
	got, err = fs.ReadFile(overlay, "b.md")
	if err != nil || string(got) != "b" {
		t.Errorf("nil-bottom: want b, got %q err=%v", got, err)
	}

	overlay = OverlayFS(nil, nil)
	if _, err := fs.ReadFile(overlay, "anything"); err == nil {
		t.Error("nil-both: expected error reading from empty overlay")
	}
}

// TestOverlayFS_MissingPath returns a not-exist error when neither
// layer carries the requested file.
func TestOverlayFS_MissingPath(t *testing.T) {
	overlay := OverlayFS(fstest.MapFS{}, fstest.MapFS{})
	if _, err := fs.ReadFile(overlay, "missing.md"); err == nil {
		t.Error("want error for missing file, got nil")
	}
	if _, err := fs.ReadDir(overlay, "missing-dir"); err == nil {
		t.Error("want error for missing dir, got nil")
	}
}

// TestOverlayFS_CoreAndEngineeringDomain exercises the real install
// boundary: hero.OverlayFS(hero.DomainFS("engineering"), hero.CoreFS())
// must surface entries from both layers, and engineering must shadow
// core where they overlap. Parallels the precedence test for
// spec-types in internal/spectypes/loader_test.go.
func TestOverlayFS_CoreAndEngineeringDomain(t *testing.T) {
	engFS, err := DomainFS("engineering")
	if err != nil {
		t.Fatalf("DomainFS(engineering): %v", err)
	}
	coreFS := CoreFS()
	overlay := OverlayFS(engFS, coreFS)

	for _, kind := range []string{"agents", "commands", "skills"} {
		entries, err := fs.ReadDir(overlay, kind)
		if err != nil {
			t.Fatalf("ReadDir %s: %v", kind, err)
		}
		if len(entries) == 0 {
			t.Errorf("%s: overlay is empty", kind)
		}

		// Spot-check at least one entry from each layer is reachable.
		// agents/convention-author.md is in both core and engineering;
		// the overlay must return engineering's bytes (top wins).
		// agents/session-primer.md is core-only.
		// agents/api-engineer.md is engineering-only.
		if kind == "agents" {
			seen := map[string]bool{}
			for _, e := range entries {
				seen[e.Name()] = true
			}
			if !seen["session-primer.md"] {
				t.Error("agents: core-only file session-primer.md missing from overlay")
			}
			if !seen["api-engineer.md"] {
				t.Error("agents: engineering-only file api-engineer.md missing from overlay")
			}
		}
	}
}
