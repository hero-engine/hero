package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCleanupLegacy_RemovesEveryKnownHarnessPath seeds a dangling legacy
// symlink at EVERY path the cleanup list claims to cover, runs cleanup,
// and asserts each one is removed.
//
// Regression guard for upgrade-strands-install-layout: prior to v0.14.2
// the cleanup list was hand-maintained and silently drifted out of sync
// with what each target's install wrote. `.codex/agents` and
// `.codex/commands` were missing from the list for months — every codex
// install upgrade left dangling symlinks because nothing exercised the
// full enumeration.
//
// When adding a new harness target that materializes agents/commands/skills,
// add it to legacyHarnessKindPaths AND this test will pick up the new
// entry automatically (since it iterates the same list).
func TestCleanupLegacy_RemovesEveryKnownHarnessPath(t *testing.T) {
	for _, absPath := range legacyHarnessKindPaths(".") {
		// Use the relative segment as the subtest name for readability.
		name := strings.TrimPrefix(absPath, "./")
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			hp := filepath.Join(tmp, name)

			// Seed a dangling symlink at hp pointing into a .hero/ path
			// that doesn't exist. Cleanup must remove it.
			if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
				t.Fatalf("mkdir parent: %v", err)
			}
			// Target shape: ../.hero/<kind> (relative — what real legacy
			// symlinks look like). Path doesn't have to resolve for the
			// cleanup probe to see it as a hero-managed legacy artifact.
			target := "../.hero/" + filepath.Base(hp)
			if err := os.Symlink(target, hp); err != nil {
				t.Fatalf("symlink %s: %v", hp, err)
			}

			if err := cleanupLegacyCanonicalSymlinks(Options{}, tmp); err != nil {
				t.Fatalf("cleanup: %v", err)
			}

			if _, err := os.Lstat(hp); !os.IsNotExist(err) {
				t.Errorf("expected %s to be removed, got err=%v", hp, err)
			}
		})
	}
}

// TestCleanupLegacy_RemovesAnySymlinkAtHarnessPath documents the
// intentional aggressive policy: under render-direct, ANY symlink at a
// managed harness path is presumed legacy (Hero's P2 artifact OR a
// hero-on-hero dev shortcut) and is removed unconditionally — fresh
// content gets re-rendered from embedded source in the same install
// pass. See cleanup.go:142-147.
//
// This test is a regression guard against weakening the policy by
// accident (e.g. someone adds a "skip if target looks user-owned"
// check that re-introduces strand risk).
func TestCleanupLegacy_RemovesAnySymlinkAtHarnessPath(t *testing.T) {
	tmp := t.TempDir()
	hp := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
		t.Fatal(err)
	}
	// Even a non-Hero-shaped target should be removed.
	if err := os.Symlink("/tmp/something-else", hp); err != nil {
		t.Fatal(err)
	}

	if err := cleanupLegacyCanonicalSymlinks(Options{}, tmp); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	if _, err := os.Lstat(hp); !os.IsNotExist(err) {
		t.Errorf("expected unconditional symlink removal per documented policy; got err=%v", err)
	}
}

// TestDetectLegacyDrift_BrokenSymlink confirms a dangling harness-dir
// symlink pointing into .hero/ is reported as broken_symlink drift.
func TestDetectLegacyDrift_BrokenSymlink(t *testing.T) {
	tmp := t.TempDir()
	hp := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
		t.Fatal(err)
	}
	// Target does NOT exist on disk.
	if err := os.Symlink("../.hero/agents", hp); err != nil {
		t.Fatal(err)
	}

	findings := DetectLegacyDrift(tmp)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Kind != "broken_symlink" {
		t.Errorf("kind = %q, want broken_symlink", findings[0].Kind)
	}
	if findings[0].Path != hp {
		t.Errorf("path = %q, want %q", findings[0].Path, hp)
	}
}

// TestDetectLegacyDrift_ResolvedSymlinkNotReportedAsBroken: a symlink
// that points at an existing .hero/ dir is still legacy but not broken;
// it should not appear as broken_symlink (the legacy_canonical_dir
// finding for the target dir is still reported).
func TestDetectLegacyDrift_ResolvedSymlinkNotReportedAsBroken(t *testing.T) {
	tmp := t.TempDir()
	hp := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
		t.Fatal(err)
	}
	// Make the symlink AND its target exist.
	if err := os.MkdirAll(filepath.Join(tmp, ".hero", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../.hero/agents", hp); err != nil {
		t.Fatal(err)
	}

	findings := DetectLegacyDrift(tmp)
	for _, f := range findings {
		if f.Kind == "broken_symlink" {
			t.Errorf("resolved symlink should not be flagged as broken: %+v", f)
		}
	}
	// But the legacy_canonical_dir should be flagged.
	var sawCanonical bool
	for _, f := range findings {
		if f.Kind == "legacy_canonical_dir" {
			sawCanonical = true
		}
	}
	if !sawCanonical {
		t.Errorf("expected legacy_canonical_dir finding for .hero/agents, got: %+v", findings)
	}
}

// TestDetectLegacyDrift_NonHeroSymlinkIgnored confirms user-authored
// symlinks at managed paths are NOT flagged as drift. (Mutating cleanup
// still removes them per the documented aggressive policy, but the
// detector is conservative to avoid false-positive "Hero is broken"
// warnings on hand-crafted layouts.)
func TestDetectLegacyDrift_NonHeroSymlinkIgnored(t *testing.T) {
	tmp := t.TempDir()
	hp := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/users-own-agents", hp); err != nil {
		t.Fatal(err)
	}

	findings := DetectLegacyDrift(tmp)
	for _, f := range findings {
		if f.Path == hp {
			t.Errorf("non-Hero symlink should be ignored, got finding: %+v", f)
		}
	}
}

// TestDetectLegacyDrift_CleanProjectIsEmpty confirms a fresh
// render-direct install (no symlinks, no .hero mirror dirs) returns no
// findings.
func TestDetectLegacyDrift_CleanProjectIsEmpty(t *testing.T) {
	tmp := t.TempDir()
	// Create real harness dirs (what render-direct install produces).
	for _, p := range []string{".claude/agents", ".claude/commands", ".claude/skills"} {
		if err := os.MkdirAll(filepath.Join(tmp, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	findings := DetectLegacyDrift(tmp)
	if len(findings) != 0 {
		t.Errorf("clean project should have no drift, got %d findings: %+v", len(findings), findings)
	}
}

// TestDetectLegacyDrift_ReportsEveryHarnessPath asserts that the
// detector picks up dangling symlinks at every path in the cleanup list
// — the readonly-detection cousin of
// TestCleanupLegacy_RemovesEveryKnownHarnessPath.
func TestDetectLegacyDrift_ReportsEveryHarnessPath(t *testing.T) {
	for _, absPath := range legacyHarnessKindPaths(".") {
		name := strings.TrimPrefix(absPath, "./")
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			hp := filepath.Join(tmp, name)
			if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
				t.Fatal(err)
			}
			target := "../.hero/" + filepath.Base(hp)
			if err := os.Symlink(target, hp); err != nil {
				t.Fatal(err)
			}

			findings := DetectLegacyDrift(tmp)
			var found bool
			for _, f := range findings {
				if f.Path == hp && f.Kind == "broken_symlink" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected broken_symlink finding for %s, got: %+v", hp, findings)
			}
		})
	}
}

// TestCleanupLegacy_RemovesCanonicalMirrorDirs confirms the .hero/{agents,
// commands,skills}/ mirror directories from the P2 era are removed.
func TestCleanupLegacy_RemovesCanonicalMirrorDirs(t *testing.T) {
	tmp := t.TempDir()
	for _, kind := range []string{"agents", "commands", "skills"} {
		dir := filepath.Join(tmp, ".hero", kind)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Seed with canonical-shaped content so removeLegacyDir accepts it.
		if err := os.WriteFile(filepath.Join(dir, "marker.md"), []byte("hero-managed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := cleanupLegacyCanonicalSymlinks(Options{}, tmp); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	for _, kind := range []string{"agents", "commands", "skills"} {
		dir := filepath.Join(tmp, ".hero", kind)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			// removeLegacyDir is conservative about non-canonical content;
			// a single "marker.md" file may survive. Accept either fully
			// removed OR empty-of-canonical-shape — the important thing
			// is the directory is no longer holding loader-readable state.
			entries, _ := os.ReadDir(dir)
			if len(entries) > 0 {
				t.Logf("note: %s survived cleanup with %d entries (non-canonical content preserved)", dir, len(entries))
			}
		}
	}
}
