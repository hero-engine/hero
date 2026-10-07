package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// symlinked-hero-dir: `install satellites --migrate-nested --apply` skips a
// nested workspace whose .hero is a symlink — leaving its target and the
// link untouched — and still migrates the real nested workspaces.
func TestMigrateNestedApplySkipsLinkedHero(t *testing.T) {
	env := newTestEnv(t)
	t.Cleanup(func() {
		satellitesMigrateNested, satellitesApply, satellitesYesAll, satellitesForce = false, false, false, false
	})
	write := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A Claude harness at the root, so the migrated satellite can materialize.
	for _, sub := range []string{"agents", "commands", "skills"} {
		write(filepath.Join(env.dir, ".claude", sub, "sentinel.md"), "x")
	}
	spec := "---\ntitle: S\ntype: feature\nstatus: planning\n---\n# s\n"
	write(filepath.Join(env.dir, "engines", "mlx", ".hero", "planning", "features", "real", "spec.md"), spec)
	target := filepath.Join(t.TempDir(), "other-checkout", ".hero")
	targetSpec := filepath.Join(target, "planning", "features", "live", "spec.md")
	write(targetSpec, spec)
	link := filepath.Join(env.dir, "apps", "web", ".hero")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	out, err := runCmd("install", "satellites", "--migrate-nested", "--apply", "--yes", "--force")
	if err != nil {
		t.Fatalf("migrate: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Skipped apps/web") {
		t.Errorf("output does not report skipping the linked workspace:\n%s", out)
	}
	if _, err := os.Stat(targetSpec); err != nil {
		t.Errorf("the link target's spec was moved: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.heroDir, "planning", "features", "real", "spec.md")); err != nil {
		t.Errorf("the real nested workspace was not migrated: %v\n%s", err, out)
	}
}
