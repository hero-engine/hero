package cli

import (
	"path/filepath"
	"testing"
)

func TestRelocatedPathPlanningTree(t *testing.T) {
	root := "/r"
	got := relocatedPath(filepath.Join(root, ".hero/planning/features/foo/spec.md"), root, "engines/mlx")
	want := filepath.Join(root, ".hero/planning/features/engines/mlx/foo/spec.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRelocatedPathRoot(t *testing.T) {
	root := "/r"
	got := relocatedPath(filepath.Join(root, ".hero/planning/features/engines/mlx/foo/spec.md"), root, "")
	want := filepath.Join(root, ".hero/planning/features/foo/spec.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRelocatedPathNoChange(t *testing.T) {
	root := "/r"
	src := filepath.Join(root, ".hero/planning/features/foo/spec.md")
	got := relocatedPath(src, root, "")
	if got != src {
		t.Errorf("expected unchanged path, got %q", got)
	}
}

func TestRelocatedPathOddPath(t *testing.T) {
	// A file outside .hero/planning shouldn't be relocated.
	src := "/r/random/path.md"
	got := relocatedPath(src, "/r", "engines/mlx")
	if got != src {
		t.Errorf("expected unchanged path for odd input, got %q", got)
	}
}

func TestFormatScope(t *testing.T) {
	if formatScope("") != "(root)" {
		t.Errorf("expected (root) for empty")
	}
	if formatScope("engines/mlx") != "engines/mlx" {
		t.Errorf("expected pass-through for non-empty")
	}
}
