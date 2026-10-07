package install

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupNestedFixture creates a root workspace with a nested .hero/ at
// engines/mlx containing one spec, one knowledge file, and an events.log.
// Returns the root path.
func setupNestedFixture(t *testing.T) string {
	t.Helper()
	root := setupRoot(t) // .hero/ + .claude/ tree
	nested := filepath.Join(root, "engines", "mlx", ".hero")
	for _, sub := range []string{"planning/features/old-feature", "knowledge/notes/some-note"} {
		if err := os.MkdirAll(filepath.Join(nested, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	specBody := "---\ntitle: Old\ntype: feature\nstatus: planning\n---\n# old\n"
	if err := os.WriteFile(filepath.Join(nested, "planning/features/old-feature/spec.md"), []byte(specBody), 0o644); err != nil {
		t.Fatal(err)
	}
	noteBody := "---\ntitle: a note\ntype: note\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(nested, "knowledge/notes/some-note/spec.md"), []byte(noteBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "events.log"), []byte("{\"ts\":\"2026-01-01T00:00:00Z\",\"type\":\"foo\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestApplyMigrationDryRun(t *testing.T) {
	root := setupNestedFixture(t)
	res, err := ApplyMigration(ApplyOptions{
		RootDir: root,
		Version: "test",
		DryRun:  true,
	}, "engines/mlx")
	if err != nil {
		t.Fatalf("apply dry-run: %v", err)
	}
	if len(res.SpecsMoved) != 1 {
		t.Errorf("expected 1 spec move, got %d", len(res.SpecsMoved))
	}
	if res.NestedRemoved {
		t.Errorf("dry-run should not remove nested")
	}
	// Confirm nothing actually moved.
	if _, err := os.Stat(filepath.Join(root, "engines/mlx/.hero/planning/features/old-feature/spec.md")); err != nil {
		t.Errorf("source spec was moved during dry-run: %v", err)
	}
}

func TestApplyMigrationFull(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows")
	}
	root := setupNestedFixture(t)
	res, err := ApplyMigration(ApplyOptions{
		RootDir: root,
		Version: "test",
		Force:   true, // skip git check (no git repo in test)
	}, "engines/mlx")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(res.SpecsMoved) != 1 {
		t.Errorf("expected 1 spec moved, got %d", len(res.SpecsMoved))
	}
	if !res.NestedRemoved {
		t.Errorf("expected nested removed")
	}

	// Spec lives at root .hero/planning/...
	dst := filepath.Join(root, ".hero", "planning", "features", "old-feature", "spec.md")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("dst spec missing: %v", err)
	}
	if !strings.Contains(string(data), "subproject: engines/mlx") {
		t.Errorf("scope was not stamped on migrated spec:\n%s", string(data))
	}

	// Nested .hero/ is gone.
	if _, err := os.Stat(filepath.Join(root, "engines/mlx/.hero")); !os.IsNotExist(err) {
		t.Errorf("nested .hero should be gone")
	}

	// subprojects.json declares it.
	subs, err := LoadSubprojects(filepath.Join(root, ".hero"))
	if err != nil {
		t.Fatal(err)
	}
	if !subs.IsDeclared("engines/mlx") {
		t.Errorf("subprojects.json should declare engines/mlx after migration")
	}

	// Satellite materialized.
	if _, err := os.Stat(filepath.Join(root, "engines/mlx/.claude/agents")); err != nil {
		t.Errorf("satellite agents symlink missing after migration: %v", err)
	}

	// Events.log appended at root.
	rootEvents, _ := os.ReadFile(filepath.Join(root, ".hero/events.log"))
	if !strings.Contains(string(rootEvents), "migrated_from") {
		t.Errorf("root events.log missing migrated_from decoration:\n%s", string(rootEvents))
	}
}

func TestStampSubprojectInFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	src := "---\ntitle: x\n---\n# x\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stampSubprojectInFile(path, "engines/mlx"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "subproject: engines/mlx") {
		t.Errorf("stamp missing:\n%s", data)
	}
}

func TestCollisionSuffix(t *testing.T) {
	got := collisionSuffix("/r/.hero/planning/features/x/spec.md", "engines/mlx")
	want := "/r/.hero/planning/features/x-from-engines-mlx/spec.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// symlinked-hero-dir: a nested .hero that is a symlink is refused by both
// plan and apply, and nothing in its target or the link is touched.
func TestMigrationRefusesLinkedNestedHero(t *testing.T) {
	root := setupRoot(t)
	target := filepath.Join(t.TempDir(), "other-checkout", ".hero")
	targetSpec := filepath.Join(target, "planning", "features", "live", "spec.md")
	if err := os.MkdirAll(filepath.Dir(targetSpec), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetSpec, []byte("---\ntitle: Live\ntype: feature\nstatus: planning\n---\n# live\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "apps", "web", ".hero")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	if _, err := PlanMigration(root, "apps/web"); !errors.Is(err, ErrLinkedNestedHero) {
		t.Fatalf("PlanMigration err = %v, want ErrLinkedNestedHero", err)
	}
	if _, err := ApplyMigration(ApplyOptions{RootDir: root, Version: "test", Force: true}, "apps/web"); !errors.Is(err, ErrLinkedNestedHero) {
		t.Fatalf("ApplyMigration err = %v, want ErrLinkedNestedHero", err)
	}
	if _, err := os.Stat(targetSpec); err != nil {
		t.Errorf("the link target's spec was touched: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link was removed or replaced: %v", err)
	}
}
