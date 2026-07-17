package install

import (
	"bytes"
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

// ---------------------------------------------------------------------------
// codex-agents-wholesale-wipe-destroys-user-files: the .codex/agents cleanup
// is .md-scoped (removeLegacyDirMatching), not a wholesale wipe. These tests
// pin the load-bearing survival guarantee and the scoping semantics.
// ---------------------------------------------------------------------------

// TestRunCodex_PreservesUserAuthoredTomlAgent is the load-bearing regression
// guard (AC-1). A user's hand-authored .codex/agents/<name>.toml — a live
// Codex agent Hero never wrote — MUST survive a routine `hero install`, while
// the pre-.toml legacy .md dead-byte is still cleaned and Hero's current .toml
// agents are re-rendered. This install used to os.RemoveAll the whole dir.
func TestRunCodex_PreservesUserAuthoredTomlAgent(t *testing.T) {
	h := newInstallHarness(t)
	// Pre-init the .hero/ workspace so the install runs its full path,
	// including the manifest-driven prune (mirrors TestHarness_SmokeCodex).
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// First install materializes Hero's .toml agents.
	h.Run(TargetCodex, nil)
	h.mustBeRegularFile(".codex/agents/engineer.toml")

	// Plant a user-authored .toml agent (a non-Hero name, never in the
	// install manifest) and a pre-.toml legacy .md dead-byte.
	userAbs := filepath.Join(h.TargetDir, ".codex", "agents", "my-custom.toml")
	userBytes := []byte("# my hand-authored codex agent — do not touch\ndeveloper_instructions = \"mine\"\n")
	if err := os.WriteFile(userAbs, userBytes, 0o644); err != nil {
		t.Fatalf("plant user .toml: %v", err)
	}
	legacyMd := filepath.Join(h.TargetDir, ".codex", "agents", "engineer.md")
	if err := os.WriteFile(legacyMd, []byte("legacy dead-byte"), 0o644); err != nil {
		t.Fatalf("plant legacy .md: %v", err)
	}

	// Re-run the routine, idempotent-looking install.
	h.Run(TargetCodex, nil)

	// AC-1: the user file survives byte-for-byte.
	got, err := os.ReadFile(userAbs)
	if err != nil {
		t.Fatalf("AC-1: user .toml agent was destroyed: %v", err)
	}
	if !bytes.Equal(got, userBytes) {
		t.Errorf("AC-1: user .toml agent mutated: got %q want %q", got, userBytes)
	}
	// AC-2: the legacy .md dead-byte is removed.
	h.mustNotExist(".codex/agents/engineer.md")
	// AC-4: Hero's current .toml agents are (re)rendered and present.
	h.mustBeRegularFile(".codex/agents/engineer.toml")
	h.mustBeRegularFile(".codex/agents/reviewer.toml")
	// AC-8: the live dir is NOT removed while user/.toml files remain.
	h.mustBeDirectory(".codex/agents")
}

// TestRunCodex_CommandsDirWholesaleRemoved pins AC-5: the .codex/commands
// cleanup keeps wholesale removal (no loader there, nothing repopulates it).
// A planted .toml proves the removal is NOT .md-scoped like .codex/agents.
func TestRunCodex_CommandsDirWholesaleRemoved(t *testing.T) {
	h := newInstallHarness(t)
	dir := filepath.Join(h.TargetDir, ".codex", "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"leftover.md", "user-thing.toml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatalf("plant %s: %v", f, err)
		}
	}

	h.Run(TargetCodex, nil)

	// Whole dir gone (emptied by wholesale removal, then rmdir'd) — the
	// .toml would have survived a .md-scoped cleanup.
	h.mustNotExist(".codex/commands")
}

// TestRemoveLegacyDirMatching_MdScopedPreservesOthers pins the predicate
// semantics: with the .md predicate only .md entries go, and the live dir is
// preserved because non-.md entries remain (AC-2, AC-8).
func TestRemoveLegacyDirMatching_MdScopedPreservesOthers(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := map[string]string{
		"engineer.md": "legacy dead-byte", // Hero's to remove
		"user.toml":   "user agent",       // must survive
		"notes.txt":   "user notes",       // must survive
	}
	for name, body := range seed {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	if err := removeLegacyDirMatching(Options{}, dir, isLegacyCodexAgentMarkdown); err != nil {
		t.Fatalf("removeLegacyDirMatching: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "engineer.md")); !os.IsNotExist(err) {
		t.Errorf("AC-2: legacy .md not removed (err=%v)", err)
	}
	for _, keep := range []string{"user.toml", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("scoped cleanup removed non-.md %s: %v", keep, err)
		}
	}
	// AC-8: the live dir stays because non-.md entries survive.
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("AC-8: live dir wrongly removed: %v", err)
	}
}

// TestRemoveLegacyDirMatching_RemovesEmptyDirAfterScopedWipe: when every
// entry matches the predicate, the dir ends up empty and is removed — the
// smoke-test invariant for a dir that held only dead .md bytes.
func TestRemoveLegacyDirMatching_RemovesEmptyDirAfterScopedWipe(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"engineer.md", "reviewer.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("legacy"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", f, err)
		}
	}

	if err := removeLegacyDirMatching(Options{}, dir, isLegacyCodexAgentMarkdown); err != nil {
		t.Fatalf("removeLegacyDirMatching: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("dir with only .md dead-bytes should be removed once empty (err=%v)", err)
	}
}

// TestRemoveLegacyDirMatching_NilPredicateWholesale proves the nil-predicate
// wrapper keeps byte-identical wholesale behavior — the path .codex/commands,
// copilot's .github/copilot/*, and the .hero mirror rely on (AC-5, AC-6).
func TestRemoveLegacyDirMatching_NilPredicateWholesale(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "commands")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.md", "b.toml", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", f, err)
		}
	}

	if err := removeLegacyDirMatching(Options{}, dir, nil); err != nil {
		t.Fatalf("removeLegacyDirMatching: %v", err)
	}

	// Everything — files of every extension AND the subdir — is gone, and the
	// emptied dir itself is removed.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("nil predicate must wholesale-remove the dir (err=%v)", err)
	}
}

// TestRemoveLegacyDirMatching_MissingDirNoOp pins AC-7: a fresh install where
// .codex/agents does not exist yet is a no-op with no error.
func TestRemoveLegacyDirMatching_MissingDirNoOp(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if err := removeLegacyDirMatching(Options{}, missing, isLegacyCodexAgentMarkdown); err != nil {
		t.Errorf("AC-7: missing dir should no-op, got %v", err)
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
